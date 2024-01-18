# Providers

A provider provides resources that can be configured by consumers. 

## Schema

The resources that are available to consumers are defined in a schema.
The schema is configured using a supported programming language of the provider author.
A translator translates the schema into a representation that Athanor understands.
Athanor generates source code to implement the provider interface in the supported language that the provider author prefers.
Athanor also generates SDKs for each target programming language configured.

## Generated provider SDK

The generated provider source code should be used to implement the provider interface for each resource.
Provider authors should implement CRUD operations for each resource. 
The following is the interface that's expected for each resource:

* Get -- fetches the resource using the provided identifier.
* Create -- creates the resource using the provided identifier and config values.
* Update -- updates the resource using the provided identifier, config values, and a mask of all the fields which need to be udpated.
* Delete -- deletes the resource using the provided identifier.

## Generated client SDK

Athanor generates client code for each supported target programming language.
Blueprint authors can use the generated resource code to manage resources in blueprints. 
