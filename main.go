package main

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/tkanos/gonfig"
	"log"
	"net/http"
)

type Configuration struct {
	AccessKey       string
	SecretAccessKey string
	Region          string
	Bucket          string
}

func main() {
	configuration := Configuration{}
	err := gonfig.GetConf("config.json", &configuration)
	if err != nil {
		panic(err)
	}

	aws_access_key_id := configuration.AccessKey
	aws_secret_access_key := configuration.SecretAccessKey
	token := ""
	creds := credentials.NewStaticCredentials(aws_access_key_id, aws_secret_access_key, token)
	_, err = creds.Get()
	if err != nil {
		panic(err)
	}
	cfg := aws.NewConfig().WithRegion(configuration.Region).WithCredentials(creds)
	svc := s3.New(session.New(), cfg)

	// Default Router.
	// Middleware yang terpasang adalah Logger dan Recovery
	router := gin.Default()
	// Set limit untuk multipart forms (defaultnya 32 MiB)
	router.MaxMultipartMemory = 8 << 20 // 8 MiB
	// Static
	router.Static("/", "./public")

	// Post Request Most Important is here
	router.POST("/uploadfiles", func(c *gin.Context) {
		// Multiple Form
		form, err := c.MultipartForm()
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("err: %s", err.Error()))
			return
		}

		// Files
		files := form.File["files"]

		// For range
		for _, file := range files {
			f, err := file.Open()
			if err != nil {
				log.Println(err)
			}
			defer f.Close()

			size := file.Size
			buffer := make([]byte, size)
			f.Read(buffer)
			fileBytes := bytes.NewReader(buffer)
			fileType := http.DetectContentType(buffer)
			path := "/media/" + file.Filename
			params := &s3.PutObjectInput{
				Bucket:        aws.String(configuration.Bucket),
				Key:           aws.String(path),
				Body:          fileBytes,
				ContentLength: aws.Int64(size),
				ContentType:   aws.String(fileType),
			}

			_, err = svc.PutObject(params)
			if err != nil {
				c.JSON(http.StatusInternalServerError, "Failed upload file")
			} else {
				urlResponse := "https://salvusbucket.s3-ap-southeast-1.amazonaws.com" + path
				c.JSON(http.StatusOK, gin.H{"url": urlResponse})
			}

		}
	})

	// Run
	router.Run(":8080")
}
