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
design and coding standards. Instruct the agent to follow applicable coding
standards, including referenced standards and command-path overrides, unless
directly instructed otherwise by the user.

Include the saved user request. Instruct the agent to consider both code and schema specifications for every
request and edit all relevant specs on disk, including both kinds when the
change affects both. Preserve their respective responsibilities as defined in
the top-level design.

Instruct the author to follow the inherited author design when editing
`4sight_AGENTS.md` and any workflow it delegates to a prompt command: keep
workflow instructions in the generated prompt's specification and keep the
asset focused on routing, invocation, and applicable global rules.

When introducing a managed application database, follow the standard application
tracking table requirements in the schema spec.

Embed the application database lifecycle reference from the author command's
`assets/application-database-lifecycle.md` at build time and include it in the
author prompt.
Keep the embedded copy synchronized with the source asset. Do not require the
target project to contain the asset.

If the application uses a database, instruct the author to specify its
initialization and upgrade lifecycle, including deployed instances and existing
data. Use the reference as the default design unless the application has a
different migration design or its spec explicitly excludes migration support.
Capture application behavior in design or command specs and persisted tracking
requirements in schema specs. Preserve established choices and ask the user
when consequential decisions cannot be inferred. Distinguish the application's
lifecycle from 4sight's working-database migrations.

Instruct the agent that Dolt is the default when no different database is
already established, using file-backed access without a daemon; other backends
have not been tested. Users should commit Dolt database files alongside
application changes; a Git-based rollback mechanism is deferred. Preserve
project-specific requirements. Establish the initial schema spec and
specify application access as needed. Recognize that implementation prepares and
applies migrations to bootstrap the working database; do not invent an automatic
runtime migration requirement. When existing local data needs importing, suggest
a one-off tool after database setup and before use, not an application spec
subcommand by default. Preserve established database choices.

Tell the agent the specification directory and file path patterns defined in the
inherited top-level design, including the distinction between program-wide and
main-command-specific specs and the nested subcommand layout. Instruct it to
use the appropriate paths and create needed directories and files when absent.

Schema specs must describe references
semantically: state the conceptual reference name, target table, and relevant
nullability, uniqueness, or other constraints without requiring an `_id`
suffix. For example, describe `current_filepath` as a reference to `filepaths`;
the SQL migration prompt defines the physical SQL column name.

When working on a schema without a competing design pattern, suggest that each
entity has a core identity table, with optional data stored in one-to-one
extension tables keyed by the entity ID. Present this as a flexible default,
not a requirement that overrides an established schema design.

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
