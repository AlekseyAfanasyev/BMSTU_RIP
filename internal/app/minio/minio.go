package minio

import (
	"github.com/minio/minio-go/v7"
	"gopkg.in/yaml.v3"
	"log"
	"os"

	conf "BMSTU_RIP/config"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	*minio.Client
}

func NewMinioClient() *MinioClient {

	yamlFile, err := os.ReadFile("config/config.yaml")
	if err != nil {
		log.Fatalln(err)
	}

	config := conf.Config{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalln(err)
	}
	useSSL := false

	minioClient, err := minio.New(config.Minio.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Minio.User, config.Minio.Pass, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &MinioClient{
		minioClient,
	}
}
