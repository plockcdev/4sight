# 4sight spec change author design

The author prompt prepares an agent to apply a saved spec change request to the
workspace specs. It is separate from the review prompt: authoring edits specs
and imports them, while review assesses imported changes without authoring from
the saved request.

When authoring `4sight_AGENTS.md`, if an instruction delegates a workflow to a
4sight prompt command, keep the asset focused on when to invoke the command,
which prompt to select, and following its output. Specify the substantive steps,
scope-specific behavior, and completion instructions in the delegated prompt's
specification so its generated output carries them. Do not duplicate or override
those steps in the asset. Preserve routing between prompts and genuinely global
rules that apply beyond one workflow. Explicit user instructions take precedence.

Schema specs describe the logical data model. Describe references by their
conceptual names and explicitly identify the referenced table and relevant
constraints. Do not require implementation-specific `_id` suffixes in schema
spec names. SQL migration generation applies its own physical column naming
convention.

Refer to it simply as the schema spec, without naming its filename. The
schema-spec location and naming convention are defined in the inherited spec
design.

For entity tables, the ordinary `id` primary key is implied and should not be
listed as a schema-spec field. Pure join tables do not need a surrogate `id`;
their referenced fields form a composite primary key unless the spec says
otherwise. Specify a key when it has a meaningful natural form, custom format,
or nonstandard constraint.
