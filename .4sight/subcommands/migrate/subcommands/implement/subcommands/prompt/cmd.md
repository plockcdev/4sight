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

For a live submitted schema, inspect the database's submitted migration before
allocating a new migration. Resolve the schema identity from its path and the
recorded change ID, and compare its recorded CRC32 with the normalized submitted
schema body as specified in the migration design. If identity and CRC32 match,
report the existing migration as already prepared and do not create another or
move the submitted link. A matching CRC32 is a practical match, not proof that
the SQL is correct or that the migration has been applied. Use migration state
to determine whether it is applied. Do not use SHA-256 for matching.

If the submitted migration lacks valid matching metadata, do not infer that it
implements the schema. For missing or malformed metadata on a pre-existing
migration where the correct baseline cannot otherwise be established, stop and
report the ambiguity and request reconciliation. For a valid different schema
checksum, determine whether the change needs SQL; a mismatch alone does not
require a migration. If no SQL is needed, report that determination without
creating an empty migration solely to record metadata.

If there are no migrations, this is the first one. Instruct the agent to create
the initial migration for the empty database and its submitted link. For an
application database, `migrate up` prepares its standard tracking table before
applying the migration. For internal databases, migrations must be embedded in
the 4sight binary and are applied by `4sight upgrade`, not `migrate up`.

For an ordinary SQL migration, tell the agent to generate the migration file for requested
spec for the db:

- show target path for migration calculated from input arg
- show the schema spec path and tell agent to read it
- instruct the agent to include the schema metadata comments defined in the
  migration design for a migration implementing a live submitted schema

Tell the agent after writing the migration file, to move submitted to the
new migration file

For a schema-only request, instruct the agent to report the prepared migration
as pending and not yet applied. For an internal migration, explain that the new
migration must be packaged into a newly built 4sight binary before that binary's
`4sight upgrade` can apply it. The existing binary may not include the migration.
The build must include and validate the complete internal migration chain through
the new migration. For an application migration, tell the user how to apply it
with `4sight migrate up`. Do not apply migrations or mark the submitted change
implemented as part of a schema-only request. When invoked as part of a full
spec implementation, defer applying migrations to the full implementation's
final migration step; do not ask the user to run them separately.


For a deleted submitted schema, accept its former relative schema path even
though the file is absent. Resolve its last implemented body and identity from
spec history, and instruct the agent to read that body and the existing chain.
Create a standalone final retirement entry as described in the migration design,
then update submitted to it. Do not generate a corresponding SQL file. Any
necessary schema or data changes belong in ordinary preceding migrations. If the database was never created,
report that no retirement migration is needed. Reject internal database deletion
and ambiguous or conflicting database lifecycles as spec defects.
