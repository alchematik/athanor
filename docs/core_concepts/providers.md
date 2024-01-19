# Providers

A provider provides resources that can be configured by consumers. 
Providers are plugins, and anyone can create a provider as long as it conforms to the provider plugin interface.

## Schema

The resources that are available to consumers are defined in a schema.
Provider authors can create a provider schema using one of the supported programming languages.
Athanor uses the provider schema to generate source code for two audiences: the provider author and blueprint authors.
Provider authors can use the generated source code to implement the provider plugin interface.
Blueprint authors can use the generated source code to configure resources in blueprints.
Source code can be generated for any language supported by a [translator](./translators.md).

## Provider Plugin Interface 

Athanor uses the [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin) library to enable a pluggable architecture.
This means that plugins can be written in any programming language supported by [gRPC](https://grpc.io/docs/languages/).
While technically a provider plugin can be implemented by implementing the [provider gRPC service](../../proto/provider/v1/provider.proto),
Athanor aims to make this process easier by generating source code that provider authors can use to implement the plugin interface.
Provider authors will need to implement [CRUD](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete) operations for each resource.

### Create

Given the identifier and configuration for a resource, the provider plugin should create the resource.
The plugin should return the created resource, including the [read-only attributes](./resources.md#attributes) of the resource.

### Get

Given the identifier of the resource, the provider plugin should fetch the resource and return it, including 
the current configuration and read-only attributes.

### Update 

Given the identifier of the resource, configuration, and a list of the fields which have changed since the last time the blueprint
was evaluated, the provider should update the resource.

### Delete

Given the identifier of the resoure, the provider should delete the resource.

