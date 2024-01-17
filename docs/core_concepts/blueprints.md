# Blueprints

A blueprint is the configuration that defines the state in which your resources should be in.

## Configuring Resources

One of the main building blocks of blueprints are resources.
Resources correspond to an object managed by a provider.  
Bindings for resources in your preferred supported programming language are made available by provider authors by using translators.

Below is an example blueprint in Go, where a `"bucket"` resource provided by the `"gcp"` provider is being configured.
The `"gcp"` provider is a provider which manages resources in Google Cloud Platform (GCP).

```go
package main

import (
    "log"

    gcp "github.com/alchematik/athanor-provider-gcp/gen/sdk/go"
    sdk "github.com/alchematik/athanor-go/sdk/consumer"
)

func main() {
    provider := sdk.Provider{Name: "gcp", Version: "v0.0.1"}

    bp := sdk.Blueprint{}

    bucketID := gcp.BucketIdentifier{
        Alias: "my-bucket",
        Project: "my-cool-project",
        Location: "us-east4",
        Name: "athanor-test-bucket",
    }
    bucketConfig := gcp.BucketConfig{
        Labels: map[string]any{
            "test": "true",
        },
    }
    bucket := sdk.Resource{
        Provider: provider,
        Identifier: bucketID,
        Config: bucketConfig,
        Exists: true,
    }
    bp = bp.WithResource(bucket)

    if err := sdk.Build(bp); err != nil {
        log.Falatf("error building blueprint: %v", err)
    }
}
```

The above blueprint creates an object storage bucket named `athanor-test-bucket` for the `my-cool-project` GCP project in the `us-east4` region.
The bucket will have the label `test:true`.


## Dependencies

Often times resources have dependencies on other resources, and the order in which resources are created, updated, and deleted matter.
Blueprints provide two ways to configure a dependent relationship among resources.

### Identifiers

The first way is if a resource requires the identifier of another resource.
In this case, the provider schema author has configured the resource schema so that one of the fields in the identifier or config of a resource
takes in another resource's identifier. Below is an example that builds on the previous example, where a GCP object storage bucket is a dependency for an
object stored in that bucket.

```go
// ...code to configure bucket is above...

objectID := gcp.BucketObjectIdentifier{
    Alias: "my-bucket-object",
    Bucket: bucketID,
    Name: "test-object",
}
objectConfig := gcp.BucketObjectConfig{
    Contents: sdk.File{Path: "./some/path/test.json"},
}
object := sdk.Resource{
    Provider: provider,
    Identifier: objectID,
    Config: objectConfig,
    Exists: true,
}
bp = bp.WithResource(object)

// ...code to build the blueprint is below...
```

A GCP storage bucket object named `test-object` will be created in the bucket configured earlier. The contents of the object will be that of a 
local file. Since the `Bucket` field in the object identifier is a `bucket` resource identifier, Athanor knows to create the `bucket` resource first when
evaluating the blueprint.

### Using Attributes From Other Resources

The other way to inform Athanor of a dependant relationship is to use the attributes of another resource when configuring a resource. 
In the contrived example below, the "created at" timestamp of a bucket is used as label for a different bucket.
The "created at" timestamp of a GCP bucket resource is made available as a read-only attribute (learn more about read-only attributes).

```go
// ...code to configure other bucket is above...

otherBucketID := gcp.BucketIdentifier{
    Alias: "my-other-bucket",
    Project: "my-cool-project",
    Location: "us-east1",
    Name: "other-athanor-test-bucket",
}
otherBucketConfig := gcp.BucketConfig{
    Labels: map[string]any{
        "test": sdk.GetResource("my-bucket").Get("attrs").Get("created"),
    },
}
otherBucket := sdk.Resource{
    Provider: provider,
    Identifier: otherBucketID,
    Config: otherBucketConfig,
    Exists: true,
}
bp = bp.WithResource(otherBucket)

// ...code to build the blueprint is below...
```

Another storage bucket named `other-athanor-test-bucket` will be created when Athanor evaluates the blueprint.
The `created` attribute is fetched from the first bucket using its identifier alias.
When Athanor evaluates the blueprint, it will first create the bucket with the alias `my-bucket`, and then
use the `created` attribute as an input to create the second bucket with the alias `my-other-bucket`.


## Using Other Blueprints

You can use use blueprints in other blueprints. This enables you to re-use blueprints that you've made,
or use blueprints that others have made avaliable for public use. 

This feature isn't available yet, but it's on the [roadmap](../roadmap.md)!

```go

remoteBlueprint := sdk.Blueprint{
    Repo: sdk.RemoteRepo{
        URL: "github.com/alchematik/shared-blueprints/my-cool-blueprint",
    },
    Translator: sdk.Translator{
        Name: "go",
        Version: "v0.0.1",
    },
    Input: map[string]any{
        "foo": "bar",
    },
}
bp = bp.WithBlueprint(remoteBlueprint)

localBlueprint := sdk.Blueprint{
    Repo: sdk.LocalRepo{
        Path: "./shared-blueprints/my-cool-blueprint",
    },
    Translator: sdk.Translator{
        Name: "go",
        Version: "v0.0.1",
    },
    Input: map[string]any{
        "foo": "bar",
    },
}
bp = bp.WithBlueprint(localBlueprint)

```
