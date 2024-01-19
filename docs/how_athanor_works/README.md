# How Athanor Works

This section of the docs describes the inner-workings of Athanor.
Athanor is internally comprised of four main parts: the interpreter, the evaluator, the differ, and the reconciler.
These four components along with provider and translator plugins work together to reconcile
the state described in a blueprint with the real world state.

The components act as a pipeline, where each step processes data, transforms it, and is picked up by the next component in the pipeline.


## Interpreter

The interpreter is the first component in the pipeline that processes a blueprint after it's been translated into
a form that Athanor understands (read the [docs on translators](../core_concepts/translators.md) to learn more about them). 
The purpose of the interpreter is to process the blueprint [AST](https://en.wikipedia.org/wiki/Abstract_syntax_tree) and 
produce a [dependency graph](https://en.wikipedia.org/wiki/Dependency_graph) of resources and sub-blueprints. 

[Learn more](./interpreter.md)

## Evaluator

The evaluator is up next in the pipeline. The purpose of the evaluator is to evaluate components in the order of the dependency graph
created by the interpreter, from parents to their children. The evaluator fetches the current remote state of the resources by
using the provider plugins.

[Learn more](./evaluator.md)

## Differ

The differ takes the results of the evaluator and compares the desired state of the resources versus the actual state they are in.
The differ produces a diff of the resources.

[Learn more](./differ.md)

## Reconciler

The reconciler reconciles the difference in states as indicated by the differ. Based on the diff, the reconciler will leverate the 
provider plugins to create, update, or delete resources.

[Learn more](./reconciler.md)
