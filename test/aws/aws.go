package main

import (
	"context"

	gen "github.com/alchematik/athanor/gen/aws/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/aws/v0.0.1/bucket"
)

func Registry() any {
	return &gen.ClientRegistry{
		BucketClient: &s3Client{},
	}
}

func Parser() any {
	return &gen.Parser{}
}

type s3Client struct {
}

func (c *s3Client) GetBucket(ctx context.Context, id *bucket.Identifier) (*bucket.Bucket, error) {
	return nil, nil
}

func (c *s3Client) CreateBucket(ctx context.Context, id *bucket.Identifier, config *bucket.Config) error {
	return nil
}
