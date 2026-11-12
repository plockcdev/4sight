# 4sight spec change author agent prompt

Generate instructions for an agent to author specs from the saved user request
on the current workspace change. This command does not edit or import files.

## Verify Request and Workspace

Read workspace and submitted change pointers and the workspace change prompt.
If the prompt is empty, report that there is no pending spec authoring request
and stop.

Verify the on-disk specs match their workspace database bodies, including both
directions and explicit deletion state. Use the spec and text asset comparison rules
in spec design. If any files are out of sync, list them with import/export
remediation and stop.

## Build Author Prompt

Create the composite diff from the submitted pointer to the workspace pointer
at `.4sight/changes/<workspace_change_id>/unified.diff`, including code specs,
schema specs, and tracked text assets. Use the file path at each pointer when
rendering path transitions.

Output the workspace change ID, submitted change ID, the composite diff path,
all live workspace code spec paths, and all live workspace schema spec paths.
Do not list assets as specs. Instruct the agent to read the composite diff, the
full relevant workspace specs, all relevant database schema specs, and related
design and coding standards.

Include the saved user request. Instruct the agent to edit the relevant code or
schema specs on disk to satisfy it. Schema specs must describe references
semantically: state the conceptual reference name, target table, and relevant
nullability, uniqueness, or other constraints without requiring an `_id`
suffix. For example, describe `current_filepath` as a reference to `filepaths`;
the SQL migration prompt defines the physical SQL column name.

When referring to a schema from a code spec, call it the schema spec without
naming its filename. Use inherited spec design to locate it.

Do not list a generic `id` primary-key field for entity tables; it is implied.
Do not add a surrogate `id` to a pure join table; its referenced fields form a
composite primary key unless the request specifies otherwise. Describe natural
keys, custom identifier formats, or nonstandard primary-key constraints when
they are part of the data model.

This prompt is only for specification authoring. Do not generate SQL migrations
or modify application code. If the agent changes any specs, instruct it to run
`4sight spec change import`, then run `4sight spec change review agent prompt`
and follow the refreshed review instructions. If the request causes no spec
changes, instruct it to tell the user that no review is needed because no specs
changed, then stop.
