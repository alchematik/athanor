# Reconciliation

You've told Athanor what you'd like the world to look like by authoring a blueprint.
Athanor finds out what the world currently looks like with the help of [provider plugins](./providers.md).
Athanor then produces a [diff](./diffs.md) between the real-world state of the resources and the desired state.
The diff informs Athanor on how to reconcile the desired state of the resource with the actual state it is in. 
Athanor performs CRUD operations for each resource which needs to be reconciled using the provider plugins.

## Reconciling the Resource 

Athanor decides on what action to take based on the [diff](./diffs.md) of the resource.
Athanor will take one of the following actions listed below.

### Create

Athanor will send a create command to the provider plugin responsible for the resource.
The identifier and config of the resource will be sent with the command, and the provider will
materialize the resource using this information.

### Update

Athanor will send an update command to the provider plugin responsible for the resource.
The identifier, config, and a mask of the config fields which need to be updated will be sent with the command.
The provider will make the appropriate changes to the resource.

### Delete

Athanor will send a delete command to the provider plugin responsible for the resource.
The identifier of the resource will be sent along with the command.
The provider plugin will then delete the resource.

### No-op

Athanor will not issue any commands to the provider plugin if the desired state and the real-world state for the resource match.
