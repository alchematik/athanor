package main

import (
	"context"

	gen "github.com/alchematik/athanor/gen/gcp/v0.0.1"
	bucket "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket"
	bucketobject "github.com/alchematik/athanor/gen/gcp/v0.0.1/bucket_object"
	resourcepolicy "github.com/alchematik/athanor/gen/gcp/v0.0.1/resource_policy"
)

func Registry() any {
	r := gen.ClientRegistry{
		BucketClient:         &bucketClient{},
		BucketObjectClient:   &bucketClient{},
		ResourcePolicyClient: &iamClient{},
	}
	return &r
}

func Parser() any {
	return &gen.Parser{}
}

type bucketClient struct {
}

func (c *bucketClient) GetBucket(ctx context.Context, id *bucket.Identifier) (*bucket.Bucket, error) {
	return nil, nil
}

func (c *bucketClient) CreateBucket(ctx context.Context, id *bucket.Identifier, config *bucket.Config) error {
	return nil
}

func (c *bucketClient) GetBucketObject(ctx context.Context, id *bucketobject.Identifier) (*bucketobject.BucketObject, error) {
	return nil, nil
}

func (c *bucketClient) CreateBucketObject(ctx context.Context, id *bucketobject.Identifier, config *bucketobject.Config) error {
	return nil
}

type iamClient struct {
}

func (c *iamClient) GetResourcePolicy(ctx context.Context, id *resourcepolicy.Identifier) (*resourcepolicy.ResourcePolicy, error) {
	return nil, nil
}

func (c *iamClient) CreateResourcePolicy(ctx context.Context, id *resourcepolicy.Identifier, config *resourcepolicy.Config) error {
	return nil
}
