# 4sight spec change author design

The author prompt prepares an agent to apply a saved spec change request to the
workspace specs. It is separate from the review prompt: authoring edits specs
and imports them, while review assesses imported changes without authoring from
the saved request.

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
