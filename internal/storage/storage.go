package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/swmh/gopetbin/pkg/retry"
)

type Storage struct {
	client *minio.Client
	bucket string
}

type Config struct {
	Addr            string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

func New(c Config) (*Storage, error) {
	minioClient, err := minio.New(c.Addr, &minio.Options{
		Creds: credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create minioClient: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = retry.Retry(ctx, func(ctx context.Context) error {
		_, err = minioClient.BucketExists(ctx, c.BucketName)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to storage: %w", err)
	}

	ok, err := minioClient.BucketExists(ctx, c.BucketName)
	if err != nil {
		return nil, err
	}

	if !ok {
		err = minioClient.MakeBucket(ctx, c.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("cannot create bucket: %w", err)
		}
	}

	return &Storage{
		client: minioClient,
		bucket: c.BucketName,
	}, nil
}

func (s *Storage) IsNoSuchPaste(err error) bool {
	return minio.ToErrorResponse(err).StatusCode == 404
}

func (s *Storage) GetFile(ctx context.Context, name string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get file: %w", err)
	}

	return obj, nil
}

func (s *Storage) DeleteFile(ctx context.Context, name string) error {
	err := s.client.RemoveObject(ctx, s.bucket, name, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("cannot delete file: %w", err)
	}

	return nil
}

func (s *Storage) PutFile(ctx context.Context, name string, data io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, name, data, size, minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		return fmt.Errorf("cannot put file: %w", err)
	}

	return nil
}

func (s *Storage) IsPasteExist(ctx context.Context, name string) bool {
	_, err := s.client.StatObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	return err == nil
}
