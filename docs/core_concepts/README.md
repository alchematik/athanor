# Core Concepts

Athanor is a tool that reconciles the state of the world with your desired state.
You inform Athanor of what the desired state is by providing a blueprint.  
A blueprint is a configuration of one or more resources and the states in which the resources should exist in.
A resource is an object that's managed by a provider. For example, an AWS S3 bucket. 
A provider is a service that exposes an API to manage objects. For example, AWS.
Blueprints can be configured using any supported programming language. For example, Go.
Athanor uses translators to translate the configuration into a representation that can be evaluated.

The above is a brief description of the different parts of Athanor that work together to make the configuration
you define in blueprints match reality. The links below cover each concept in greater detail.

* [Blueprints](./blueprints.md)
* [Resources](./resources.md)
* [Providers](./providers.md)
* [Translators](./translators.md)

