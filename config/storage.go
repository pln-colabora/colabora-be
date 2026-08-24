package config

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

// SetUpStorageClient builds an S3 client pointed at the Garage endpoint. Garage speaks the
// S3 API, so this same client works against any other S3-compatible backend (SeaweedFS, AWS
// S3 itself) by changing GARAGE_S3_ENDPOINT/GARAGE_REGION — nothing here is Garage-specific.
func SetUpStorageClient() *s3.Client {
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("no .env file found, relying on process environment variables: %v", err)
	}

	endpoint := os.Getenv("GARAGE_S3_ENDPOINT")
	region := os.Getenv("GARAGE_REGION")
	accessKey := os.Getenv("GARAGE_ACCESS_KEY")
	secretKey := os.Getenv("GARAGE_SECRET_KEY")

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		panic(err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
}
