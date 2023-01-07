package minio

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"path/filepath"
)

type Minio struct {
	client *minio.Client
}

func NewClient(endpoint, accessKeyID, secretAccessKey string, useSSL bool) *Minio {

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("minio client created") // minioClient is now set up
	m := &Minio{
		client: minioClient,
	}
	return m
}

func (s Minio) UploadFile(bucketName, filePath string) {
	ctx := context.Background()
	err := s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := s.client.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	fileName := filepath.Base(filePath)

	// Upload the zip file with FPutObject
	info, err := s.client.FPutObject(ctx, bucketName, fileName, filePath, minio.PutObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Successfully uploaded %s of size %d byte %s\n", fileName, info.Size)

}
