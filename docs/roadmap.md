# Roadmap

These page lists features and improvements that I'd like to make to Athanor.

## CLI

Work needs to go into what the best UI/UX is for athanor.
The CLI currently is only used for testing.


## Using blueprints within blueprints 

Users should be able to embed other blueprints in their own blueprint.
This would allow folks to reuse blueprints. Blueprint authors should be
able to pass in inputs into the blueprint that propogate to resources defined
in the sub-blueprint. Blueprint authors should also be able to use outputs of the sub-blueprint
to be used as fields in other resources in the main blueprint.
Users should be able to use bluepints that have been made available in a registry or repository.

## Downloading Plugins

Currently provider and translator plugins must be built on the host machine. Users should
be able to fetch pre-built plugin binaries from a remote registry or repository.



