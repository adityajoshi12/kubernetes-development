package main

import "minio/minio"

const (
	bucket   = "test000001"
	filePath = "/Users/a0j0buc/development/kubernetes-development/minio/client/go.mod"
)

func main() {
	endpoint := "localhost:9000"
	accessKeyID := ""
	secretAccessKey := ""
	useSSL := false

	mClient := minio.NewClient(endpoint, accessKeyID, secretAccessKey, useSSL)
	mClient.UploadFile(bucket, filePath)

}
