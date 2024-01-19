# Resources

A resource is an object that's managed by a provider.
A resource is comprised of five parts: its provider, its identifier, its configuration, its attributes, and the existance of the resource.

## Provider

The provider informs Athanor of which provider plugin should be used when managing the resource.
You must specify the name and version of the provider.

## Existance

A resource requires a flag that indicates whether the resource should exist or not. 
If the existance flag is enabled, Athanor will create the resource if it doesn't exist yet.
If the existance flag is disabled, Athanor will destroy the resource if it exists.

## Identifier

The identifier is used so that providers can uniquely identify a resource. The information required for a given resource 
identifier is specified by the provider in the [provider schema](./providers.md#schema). The identifier
could be comprised of several fields (for example: account, region, and name for a bucket resource identifier).
An identifier requires an alias to be set so that Athanor can create a dependency graph of resources.

## Configuration

The configuration field of a resource is used to configure the resource. The information required to configure  
a resource is defined by the provider author in the provider schema. Changing the configuration and 
re-evaluating the blueprint will trigger an update to the resource.

## Attributes

Attributes are read-only attributes of a resource. The value of these fields are not known until
the blueprint is being evaluated. Using an attribute field of a resource
as configuration for another resource may trigger an update, depending on if the value has changed since
the last time the blueprint was evaluated. 
