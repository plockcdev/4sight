# migrate implement prompt `<path to schema spec relative to schemas dir>`

Output will be a prompt to instruct an agent to read spec and generate a migration

## Prompt Output Specification

Tell agent to read all the design.md and cmd.md docs to inform what the schema needs to provide
(list the files that exist), inferring indexes, foreign keys, primary keys, constraints like
uniqueness, etc.

Schema specs imply a primary key named `id` for entity tables unless they
specify a different key. Generate this implied `id` primary key even though it
is omitted from the schema spec. Do not add a surrogate `id` to a pure join
table; use its referenced columns as a composite primary key unless the schema
spec specifies otherwise. Preserve any specified natural key, custom
identifier format, or nonstandard primary-key constraint.

Resolve each reference against the referenced table's primary key.

- For a single-column primary key, name the SQL foreign-key column by appending
  `_id` to the conceptual reference name. For example, schema-spec
  `current_filepath` becomes SQL `current_filepath_id`.
- For a composite primary key, create one SQL foreign-key column for each
  primary-key component. Name each column
  `<reference_name>_<component_name>_id`, use a type compatible with its target
  component, and create one composite foreign-key constraint over all columns
  in the same order as the target key. Do not use a single
  `<reference_name>_id` column for a composite key.
- If the schema spec does not make the component mapping clear, stop and report
  the ambiguity rather than inventing a mapping.

Keep `_id` as an SQL implementation convention; schema specs describe
references semantically.

program will determine the last submitted spec for that db by looking in the migrations dir
for that db, and seeing what submitted points to. This will be known as the parent.

If there was no migrations, then this is the first one.  Prompt the user to add to the
dolt_4sight_migration.md a migration tracking table for the new db like we track them for 4sight_migation then ask it to add the initial migration file to create the (empty) database and create submitted link.

For any migration, tell the agent to generate the migration file for requested
spec for the db:

- show target path for migration calculated from input arg
- show the schema spec path and tell agent to read it

Tell the agent after writing the migration file, to move submitted to the
new migration file

For a schema-only request, instruct the agent to report the pending migration
and tell the user how to run `4sight migrate up`. When invoked as part of a full
spec implementation, defer applying migrations to the full implementation's
final migration step; do not ask the user to run them separately.
