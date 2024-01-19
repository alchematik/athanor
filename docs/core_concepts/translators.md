# Translators

Translators translate configuration written in a programming language into a representation that Athanor
understands and vice versa.
Translators are plugins, and anyone can create a translator plugin as long as it conforms to the translator plugin interface.
Athanor uses the [hashicorp/go-plugin](https://github.com/hashicorp/go-plugin) library to enable a pluggable architecture.
This means that plugins can be written in any programming language supported by [gRPC](https://grpc.io/docs/languages/).
To create a translator plugin, plugin authors should implement the [translator gRPC service](../../proto/translator/v1/translator.proto).

## Translating Blueprints

Translator plugins are able to translate blueprints.
Athanor will invoke the `TranslateBlueprint` RPC of the translator plugin.
The translator plugin should read the blueprint files located at the input path provided in the request
body, translate them into the format which Athanor understands, and write the contents to a file 
located at the output path provided in the request body.


## Translating Provider Schemas

Translator plugins are able to translate [provider schemas](./providers.md#schema).
Athanor will invoke the `TranslateProviderSchema` RPC of the translator plugin.
The translator plugin should read the provider schema files located at the input path provided in the request
body, translate them into the format which Athanor understands, and write the contents to a file 
located at the output path provided in the request body.


## Generating Provider SDKs

Translator plugins are able to generate source code to be used to implement the provider in the programming language that the translator supports
given a representation of the provider schema. 
Athanor will invoke the `GenerateProviderSDK` RPC of the translator plugin.
The translator plugin should read the provider schema representation located at the input path provided in the 
request body, and generate source code to support implementing the provider.


## Generating Consumer SDKs

Translator plugins are able to generate source code to be used by blueprint authors in the programming language that the translator supports
given a representation of the provider schema. 
Athanor will invoke the `GenerateConsumerSDK` RPC of the translator plugin.
The translator plugin should read the provider schema representation located at the input path provided in the 
request body, and generate source code to support consumers managing resources in their blueprints.
