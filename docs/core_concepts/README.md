# Core Concepts

Athanor is a tool that reconciles the state of the world with your desired state.
You inform Athanor of what the desired state is by providing a [_blueprint_](./blueprints.md).  
Blueprints can be configured using any [supported programming language](../../README.md#translators). For example, Go.
A blueprint has one or more [_resources_](./resources.md).
A resource belongs to a [_provider_](./providers.md).
Athanor uses [_translators_](./translators.md) to translate the blueprint into a representation that it understands.
Athanor produces a [_diff_](./diffs.md) given a blueprint.
Athanor [reconciles](./reconciliation.md) the diff, making the world match what's configured in the blueprint.

## Blueprints

A blueprint is a configuration of one or more resources.
You can also use other blueprints _within_ your blueprint, making the reuse of blueprints possible.

[Learn more about blueprints](./blueprints.md).

## Resources

A resource is an object which has state and is managed by a [_provider_](./providers.md). For example, an AWS S3 bucket. 

[Lean more about resources](./resources.md).

## Providers

A provider makes resources available for you to use in blueprints.
For example, the [GCP proovider](https://github.com/alchematik/athanor-provider-gcp) makes cloud resources on [Google Cloud Platform](https://console.cloud.google.com/) available to be managed in blueprints.
A provider is a plugin which Athanor uses to perform [CRUD](https://en.wikipedia.org/wiki/Create,_read,_update_and_delete) operations on resources.

[Learn more about providers](./providers.md).

## Translators

Translators are plugins that translate a configurations written in a programming language into a form that Athanor understands.
Translators also generate source code in a programming language to help provider authors and consumers.

[Learn more about translators](./translators.md).


## Diffs

Athanor produces a comparison where it shows you how the real-world state is different from 
the desired state configured in a blueprint. This is called a diff.

[Learn more about diffs](./diffs.md).

## Reconciliation

The main purpose of Athanor is to make reality match the state configured in blueprints.
The process of making reality match the blueprint is called reconciliation. 

[Learn more about reconciliation](./reconciliation.md).

