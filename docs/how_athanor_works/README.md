# How Athanor Works

Athanor is internally comprised of four main parts: the interpreter, the evaluator, the differ, and the reconciler.
These four components along with provider and translator plugins work together to reconcile
the state described in a blueprint with the real world state.

The components act as a pipeline, where each step processes data, transforms it, and is picked up by the next component in the pipeline.


## Interpreter

The interpreter is the first component in the pipeline that processes a blueprint after it's been translated into
a form that Athanor understands (read the [docs on translators](../core_concepts/translators.md) to learn more about them). 
The purpose of the interpreter is to process the blueprint [AST](https://en.wikipedia.org/wiki/Abstract_syntax_tree) and 
produce a [dependency graph](https://en.wikipedia.org/wiki/Dependency_graph) of resources and sub-blueprints. 


## Evaluator

The evaluator is up next in the pipeline. The purpose of the evaluator is to traverse the dependency graph created by
the interpreter and evaluate each component (resource, sub-blueprint, etc.). "Eavaluating" in this case means fetching the state of the component.
Athanor makes two evaluation passes. The first one constructs a graph of components in their ideal state by using a stubbed API that 
returns the resources as-is. The second pass uses the provider plugins to fetch the actual state that each resource is in.

At this point Athanor has two views of the world: what it should be and how it actually is. These two views are passed to the differ.


## Differ

The differ takes the results of the evaluator and compares the desired state of the components to the actual state they are in and produces a unified diff.
The diff is a representation of what's different between the desired state and actual state of the component.
The diff captures if a component was added, changed, or removed, and also captures the sub-changes for each field (i.e. which fields were added, changed, or removed).


## Reconciler

The reconciler reconciles the difference in states as indicated by the differ. Based on the diff, the reconciler will leverate the 
provider plugins to create, update, or delete resources. The reconciler reconciles each component in the order of the dependency graph.


