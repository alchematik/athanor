# Resources

A resource is an object that's managed by a provider.
A resource is comprised of five parts: its provider, its identifier, its configuration, its attributes, and the state of the resource.

## Provider

The provider informs Athanor of which provider is should use when managing the resource.
You must specify the name and version of the provider.

## Identifier

The identifier is used to uniquely identifiy a resource. The information required for a given resource 
is specified by the provider in the provider schema (learn more about provider schemas). The identifier
could be comprised of several fields (for example: account, region, and name of bucket).
An identifier requires an alias to be set so that Athanor can create a dependency graph of resources.

## Configuration

The configuration field of a resource is used to configure the resource. The information required to configure  
a resource is defined by the provider author in the provider schema. Changing the configuration and 
re-evaluating the blueprint will trigger an update to the resource.

## Attributes

Attributes are read-only attributes of the resource. The value of these fields are not known until
the blueprint that the resource belongs to is being evaluated. Using an attribute field of a resource
as configuration for another resource may trigger an update, depending on if the value has changed since
the last time the blueprint was evaluated. 
