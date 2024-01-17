# Core Concepts

Athanor is a tool that reconciles the state of the world with your desired state.
You inform Athanor of what the desired state is by providing a _blueprint_.  
A _blueprint_ is a configuration of one or more _resources_ and the states in which the _resources_ should exist in.
A _resource_ is an object that's managed by a _provider_. For example, an AWS S3 bucket. 
A _provider_ is a service that exposes an API to manage objects. For example, AWS.
_Blueprints_ can be configured using any supported programming language. For example, Go.
Athanor uses _translators_ to translate the configuration into a representation that can be evaluated.

The above is a brief description of the different parts of Athanor that work together to make the configuration
you define in blueprints match reality. The links below cover each concept in greater detail.

* [Blueprints](./blueprints.md)
* [Resources](./resources.md)
* [Providers](./providers.md)
* [Translators](./translators.md)

