package schema

import (
	"github.com/alchematik/athanor/provider"
)

var Schema = map[string][]provider.Field{
	"bucket": {
		{
			Name: "project",
			Type: "string",
		},
		{
			Name: "region",
			Type: "string",
		},
		{
			Name: "name",
			Type: "string",
		},
	},
	"bucket_object": {
		{
			Name: "bucket",
			Type: "bucket",
		},
		{
			Name: "name",
			Type: "string",
		},
	},
	"resource_policy": {
		{
			Name:  "resource",
			Type:  "oneof",
			Oneof: []string{"bucket"},
		},
		{
			Name: "name",
			Type: "string",
		},
	},
}

var ConfigSchema = map[string][]provider.Field{
	"bucket": {},
	"bucket_object": {
		{
			Name: "contents",
			Type: "string",
		},
	},
	"resource_policy": {},
}
