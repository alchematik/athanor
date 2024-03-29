# Athanor

Athanor is an Infrastructure as code (IaC) tool that I ([@khisakuni](https://github.com/khisakuni)) am hacking on.

The goal of this project for now is primarily to serve as a personal project for learning.
Long term, I would love it to become a useful tool that's delighful for people to use.


Athanor is still in the early stages of development! It is not meant for use outside of experimentation and development. 
Features and improvements that I want to make can be found in the [Roadmap](#Roadmap).


## Documentation

Below is a table of contents for the documentation.

* [Core Concepts](./docs/core_concepts)
    * [Blueprints](./docs/core_concepts/blueprints.md)
    * [Resources](./docs/core_concepts/resources.md)
    * [Providers](./docs/core_concepts/providers.md)
    * [Translators](./docs/core_concepts/translators.md)
    * [Diffs](./docs/core_concepts/diffs.md)
    * [Reconciliation](./docs/core_concepts/reconciliation.md)
* [Athanor under the hood](./docs/how_athanor_works/)
    * [Interpreter](./docs/how_athanor_works/README.md#interpreter)
    * [Evaluator](./docs/how_athanor_works/README.md#evaluator)
    * [Differ](./docs/how_athanor_works/README.md#differ)
    * [Reconciler](./docs/how_athanor_works/README.md#reconciler)
* [Roadmap](./docs/roadmap.md)
* Tutorials &mdash; _coming soon!_ Tutorials will show you how to use Athanor using step-by-step instructions.
* Examples &mdash; _coming soon!_ Examples will show you example blueprint and provider configurations.

## Providers

* [GCP](https://github.com/alchematik/athanor-provider-gcp): Cloud resources on Google Cloud Platform.

## Translators

* [Go](https://github.com/alchematik/athanor-go): Translator for the Go programming language.


## License

Athanor is released under the [MIT License](./LICENSE).
