# Diffs

Athanor works to make the state configured in a blueprint match the state in the real world.
In order to do this, Athanor has to find out what's different between the two.
Athanor fetches the current state of each resource using the provider plugin for the resource and compares it to the resource configuration in the blueprint. 
Once it has a view of what the resource looks like currently according to the provider, Athanor is able to create a diff
of the desired state versus the actual state of the resource. 
Athanor constructs a diff of each resource until it has a diff of the whole blueprint.
Once the diff for the whole blueprint is constructed, Athanor what actions it has to take to reconcile the diff.

## Fetching the Resource

To know how the current state of the resource is different than what's defined in a blueprint, Athanor fetches
the resource using the provider plugin responsible for the resource.
Athanor uses the [identifier](./resources.md#identifier) of the resource.
If you change any part of the identifier of a resource, it will be considered a different resource. This may
result in a different diff.

## Constructing the Diff

Athanor looks at two things when constructing the resource diff: the [existance](./resources.md#existance) and the [config](./resources.md#config). 

### Create

If the existance of the resource is enabled in the blueprint but the provider cannot find the resource using its identifier,
the diff will show that the resource needs to be created.


### Update

If the existance of the resource is enabled in the blueprint and the provider is able to find the resource but the config differs,
the diff will show that the resource needs to be updated.


### Delete

If the existance of the resource is disabled in the blueprint but the provider is able to find the resource, the diff will
show that the resource needs to be deleted.


### No-op

If the existance of the resource is enabled, the provider is able to fetch the resource, and the configs match, the diff
will show that there's nothing to do for the resource since reality matches the configuration.
Similarily, if the existance of the resource is disabled and the provider does not find the resource, the diff will
show that there's nothing to do for the resource.


## Viewing the blueprint diff

Once work has been done on the CLI ([roadmap](../roadmap.md)), you'll be able to see the blueprint diff, and
what actions Athanor would take to reconcile the diff.
