package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"gopkg.in/yaml.v3"
	"log"
	"os"

	conf "BMSTU_RIP/config"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {

	yamlFile, err := os.ReadFile("internal/config/config.yaml")
	if err != nil {
		panic(err)
	}

	config := conf.Config{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(config.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.User, config.Minio.Pass, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Make a new bucket called testbucket.
	bucketName := "cnc"
	location := "eu-east-1"

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}
}
