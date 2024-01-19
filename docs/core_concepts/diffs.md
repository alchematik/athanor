# Diffs

Athanor works to make the state configured in a blueprint match the state in the real world.
In order to do this, Athanor has to find out what's different between the two.
Athanor fetches the current state of each resource using the provider plugin for the resource and compares it to the resource configuration in the blueprint. 
Once it has a view of what the resource looks like currently according to the provider, Athanor is able to create a diff
of the desired state versus the actual state of the resource. 
Athanor constructs a diff of each resource until it has a diff of the whole blueprint.
Once the diff for the whole blueprint is constructed, Athanor what actions it has to take to reconcile the diff.

## Viewing the blueprint diff

Once work has been done on the CLI ([roadmap](../roadmap.md)), you'll be able to see the blueprint diff, and
what actions Athanor would take to reconcile the diff.
