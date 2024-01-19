# Core Concepts

Athanor is a tool that reconciles desired state that you've configured with reality.
You inform Athanor of what the desired state is by providing a [_blueprint_](./blueprints.md).
Blueprints can be configured using any [supported programming language](../../README.md#translators).
A blueprint has one or more [_resources_](./resources.md).
A resource belongs to a [_provider_](./providers.md).
Athanor uses [_translators_](./translators.md) to translate the blueprint into a representation that it understands.
Athanor produces a [_diff_](./diffs.md) given a translated blueprint.
Athanor [reconciles](./reconciliation.md) the diff, making reality match what's configured in the blueprint.

## Blueprints

A blueprint is a configuration of one or more resources.
Blueprints can contain _other_ blueprints within them, allowing for the reuse of blueprints.

[Learn more about blueprints](./blueprints.md).

## Resources

A resource is an object which has state and is managed by a provider.
For example, the `"bucket"` resource provided by the [GCP provider](https://github.com/alchematik/athanor-provider-gcp) allows blueprint
authors to configure [GCP Cloud Storage Buckets](https://cloud.google.com/storage/docs/buckets).

[Lean more about resources](./resources.md).

## Providers

A provider is a plugin which makes resources available for you to use in blueprints.
For example, the [GCP provider](https://github.com/alchematik/athanor-provider-gcp) makes cloud resources on [Google Cloud Platform](https://console.cloud.google.com/) available to be managed in blueprints.

[Learn more about providers](./providers.md).

## Translators

Translators are plugins that translate configurations written in a programming language into a form that Athanor understands.
Translators also generate source code in a programming language to help provider authors and consumers.

[Learn more about translators](./translators.md).


## Diffs

Diffs are the comparisons Athanor produces where it shows how the real-world state is different from 
the desired state configured in a blueprint.

[Learn more about diffs](./diffs.md).

## Reconciliation

The main purpose of Athanor is to make reality match the state configured in blueprints.
The process of making reality match the blueprint is called reconciliation. 

[Learn more about reconciliation](./reconciliation.md).

