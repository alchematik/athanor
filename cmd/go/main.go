package main

import (
	sdk "github.com/alchematik/athanor/sdk/go"
)

func main() {
	bp := sdk.Blueprint{}

	gcp := sdk.Provider("gcp", sdk.String("gcp"), sdk.String("v0.0.1"))

	bucketID := BucketIdentifier{
		Alias:   "my-bucket",
		Account: sdk.String("12345"),
		Region:  sdk.String("us-east-1"),
		Name:    sdk.String("my-cool-bucket"),
	}
	bucketConfig := BucketConfig{
		Expiration: sdk.String("12h"),
	}

	bp = bp.WithResource(sdk.Resource(sdk.Bool(true), gcp, bucketID, bucketConfig))

	objectID := BucketObjectIdentifier{
		Alias:  "my-object",
		Bucket: bucketID,
		Name:   sdk.String("my-object"),
	}
	objectConfig := BucketObjectConfig{
		Contents:  sdk.String("blabla"),
		SomeField: sdk.GetResource(bucketID.Alias).GetAttrs().IOGet("foo").Get("bar"),
	}
	bp = bp.WithResource(sdk.Resource(sdk.Bool(true), gcp, objectID, objectConfig))

	if err := sdk.Build(bp); err != nil {
		panic(err)
	}
}

type BucketIdentifier struct {
	Alias   string
	Account sdk.Type
	Region  sdk.Type
	Name    sdk.Type
}

func (id BucketIdentifier) ToExpr() sdk.Expr {
	return sdk.ResourceIdentifier(
		"bucket",
		id.Alias,
		sdk.Map(map[string]sdk.Type{
			"account": id.Account,
			"region":  id.Region,
			"name":    id.Name,
		}),
	).ToExpr()
}

type BucketConfig struct {
	Expiration sdk.Type
}

func (c BucketConfig) ToExpr() sdk.Expr {
	return sdk.Map(map[string]sdk.Type{
		"expiration": c.Expiration,
	}).ToExpr()
}

type BucketObjectIdentifier struct {
	Alias  string
	Bucket sdk.Type
	Name   sdk.Type
}

func (id BucketObjectIdentifier) ToExpr() sdk.Expr {
	return sdk.ResourceIdentifier(
		"bucket_object",
		id.Alias,
		sdk.Map(map[string]sdk.Type{
			"bucket": id.Bucket,
			"name":   id.Name,
		}),
	).ToExpr()
}

type BucketObjectConfig struct {
	Contents  sdk.Type
	SomeField sdk.Type
}

func (config BucketObjectConfig) ToExpr() sdk.Expr {
	return sdk.Map(map[string]sdk.Type{
		"contents":   config.Contents,
		"some_field": config.SomeField,
	}).ToExpr()
}
