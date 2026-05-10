package s3

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"net/url"
	"os"
	"time"
)

type FileStorage interface {
	GetFile(fileName string) (*os.File, error)
	GetFileLink(fileName string) (string, error)
}

type Minio struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
}

func NewMinio(endpoint, access, secret, bucket, publicEndpoint, region string) *Minio {
	newClient := func(ep string) *minio.Client {
		c, err := minio.New(ep, &minio.Options{
			Creds:  credentials.NewStaticV4(access, secret, ""),
			Region: region,
		})
		if err != nil {
			log.Fatalln(err)
		}
		return c
	}

	main := newClient(endpoint)
	presign := main
	if publicEndpoint != "" && publicEndpoint != endpoint {
		presign = newClient(publicEndpoint)
	}

	return &Minio{
		client:        main,
		presignClient: presign,
		bucket:        bucket,
	}
}

func (m *Minio) GetFile(fileName string) (*os.File, error) {
	ctx := context.Background()

	object, err := m.client.GetObject(ctx, m.bucket, fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	// Create local file
	file, err := os.Create("/temp/" + fileName)
	if err != nil {
		return nil, err
	}

	// Copy object to file
	_, err = file.ReadFrom(object)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (m *Minio) GetFileLink(fileName string) (string, error) {
	ctx := context.Background()

	u, err := m.presignClient.PresignedGetObject(ctx, m.bucket, fileName, 15*time.Minute, url.Values{})
	if err != nil {
		return "", err
	}

	return u.String(), nil
}
