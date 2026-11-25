# 4sight spec implement prompt

Called to generate a prompt for implementing a submitted change, including its
schema migrations, code, and migration application.

## Check Change State

Read the current change pointers.

### Check matching

If the specs in workspace does not match current workspace changes (the last
body of each file going up the change stack), then abort and tell end user to
import or sync the files.

### Check submitted vs workspace pointer

If the submitted and workspace pointers are different, abort and tell the user
the changes and their summaries since the submitted pointer, and that they can
submit.

### Check implemented vs submitted

If the implemented and submitted pointers are the same, then nothing to be done,
stop after telling user nothing to do.

## Read Specs

Embed the 4sight migration design from `.4sight/subcommands/migrate/design.md`
in the binary at build time and include its contents in the implementation
prompt as 4sight's migration guidance. This guidance applies to ordinary
application projects as well as 4sight itself; it is not an application spec.
Keep the embedded copy synchronized with the source design when implementing
changes to it. Do not require the target project to contain this file.

Tell agent to read:

- all the design spec for top level and all subcommands
- all the schemas in `.4sight/schemas`

Do not list text assets in the code spec list. Instruct the agent to inspect text asset
changes in the composite diff and read relevant text asset files from disk.

## Check Spec Problems

The code and schema specifications work together. Tell the agent that schema
specs define persisted data structure, meaning, relationships, and constraints;
application design and command specs define operational behavior such as
database location, initialization, and access. If a requirement is misplaced,
report it as a spec issue rather than silently moving or editing it during
implementation.

If code spec has logic problems, schema has issues such as integrity or
performance, code mismatches schema, or you cannot reasonably infer proper
behavior, then stop and indicate the issues, then give them your recommendations
on how to solve, including how to work with 4spec commands. Provide file and
line numbers but also give a search string to locate because if they edit the
line numbers will be outdated.

## Output

Tell agent the following code spec files inform the app behavior, then list the
spec files that exist

Text assets are omitted from that list. The composite diff may include text asset
changes, and every changed text asset receives a file-level review entry.

Also separately list the schemas found in the `.4sight/schemas` directory
telling the agent to read those as well, group by the db type.

### Identify the project

Determine the project identity using the inherited spec design. Tell the agent
whether this is 4sight developing itself or an ordinary application. If the
identity requires a user answer, instruct the agent to stop and ask the user,
continue only after an explicit answer, and retain it for the migration step.

### Implementation task list

Instruct the agent to make and maintain a task list for the applicable steps below.
Use a durable task tracking system if one is available, so progress survives
interruptions. Mark a step complete only after it succeeds. On resuming, check
actual files, migration history, and test results before proceeding; do not
assume a task's recorded status proves the work is complete.

First compare implemented and submitted specs, including schema additions,
changes, and deletions, and inspect migration state. Do not edit or import
specs during implementation. If an internal upgrade predating this change is
needed to access the project, stop and report the dependency; do not upgrade the
working project during preparation or testing. Use isolated test projects and
databases and stop if the requested scope cannot be performed safely.

### Prepare migrations

Instruct the agent to read the applicable command and design specs and schema
specs, and to prepare all migrations needed for the submitted schemas before
changing code. Infer indexes and constraints needed to meet those specs. Follow
the migration design for chain names, metadata, and retirement. Do not prepare
migrations for unchanged schemas merely because they exist.

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

For each existing live schema, compare the implemented body directly with the
final submitted body and establish the verified applied chain and actual
database state. Prepare at most one SQL successor for the net changes required
for that database; do not create migrations for intermediate spec revisions.
Account for data transformations as well as DDL. Check that the resulting schema
and application satisfy the submitted requirements. Explain the evidence when
no SQL is needed. If the baseline or result cannot be verified, stop and request
reconciliation.

Inspect any prepared successor and its application history before reusing it.
Reuse it if it correctly implements the net delta; matching schema identity and
normalized CRC32 identify its intended body but do not prove SQL correctness or
application. Do not use SHA-256 for matching. Update or remove a known unapplied
prepared migration if the net delta requires it, preserving applied entries and
avoiding conflicting chain children. If no SQL is needed, point submitted at the
verified applied entry. Never modify an applied migration. If application is
uncertain, follow the migration recovery rules. Keep passing test results only
when the tested migration and relevant code and requirements remain unchanged;
rerun affected checks after changes.

For a new database, there is no implemented body or working database to inspect.
Create the initial migration that creates the empty database, then, if SQL is
needed, one successor to reach the final submitted schema. The successor contains
the schema metadata required by the migration design; point submitted at the
successor. If no SQL is needed after creating the database, explain that and
leave submitted at the initial migration. On retry, validate and reuse correct
prepared entries and update or remove known unapplied entries as above. Verify
the resulting schema in an isolated database before applying to the working
project. Stop on ambiguous or conflicting state rather than create duplicate
migrations.

For a deleted submitted schema, use its former path and last implemented body
from spec history. If the database was created, prepare a final retirement entry
and point submitted at it; otherwise report that no retirement is needed. Reject
internal database deletion and ambiguous or conflicting database lifecycles.
Check dependencies and migration ordering. If the change cannot be implemented
safely without changing specs, stop and explain rather than inventing a schema.

### Full implementation

Instruct the agent to carry out these steps in order:

1. Prepare the migrations as above, then make the required code changes. Do not
   apply new migrations to the working project during preparation, coding, or
   testing. Configure test projects, database endpoints, tracking records, and
   links so tests cannot change working state.
2. When migrations change, apply the target chains to fresh isolated databases
   and run relevant new-code tests. Also test upgrades from representative prior
   databases with histories and data. Check schema and data preservation,
   tracking, ordering, new-code behavior, and failure and retry or repair
   behavior as appropriate. Fix failures and rerun affected checks. For a
   code-only change with no migration, run relevant code checks instead.
3. Once checks pass, run `4sight spec change implement agent migrate up` to apply
   the prepared application migrations to the working project. Verify histories
   and current links. If the application specs require an application-owned
   migration history, reconcile it with the verified 4sight history and actual
   database state as specified before using the working database or completing
   implementation. Stop for repair if reconciliation is unsafe. For 4sight developing itself, run the new source with
   `go run ./cmd/4sight spec change implement agent migrate up` from the project
   root, then use `go run ./cmd/4sight` for verification and completion. Do not
   run `4sight upgrade` for these application migrations. If project identification
   required a user answer, provide it to the migration command on standard input
   as described in its spec. If the transition cannot be performed safely, stop
   and explain the staged rollout needed.
4. Run `4sight spec change implement agent done` only after all required work and
   verification succeeds, including no-SQL schema equivalence and any application
   work required by the submitted specs. On failure, stop, report partial results
   and repair needed, and do not mark the change implemented.

If testing, deployment, or migration cannot safely follow this order, stop and
explain the required staged rollout. Verify application tracking-table creation,
internal upgrade gating, and migration ordering when implementing this behavior.

### Code-only implementation

For an explicitly code-only request, do not generate or apply migrations. If
code requires a pending migration, stop and report the dependency. After code
verification, run `4sight spec change implement agent done` only if all other
required work is complete; otherwise report the remaining work.

### Schema-only implementation

For an explicitly schema-only request, prepare migrations using the rules above,
report what remains pending and how it will be applied during full
implementation, and stop. Do not apply migrations or mark the submitted change
implemented. For 4sight's own internal databases, explain that a newly built
binary must embed and validate the complete new chains before it can use them.

### Code instructions

Remove command implementations whose cmd.md specs no longer exist. Update
command routing, help, and usage accordingly.

### Command Help

Update the help when the subcommand structure changes such as changed arguments,
new or removed subcommands.

Help is also shown for "--help" after any subcommand or the main command with a
clean exit code. When help argument is used, any other data argument causes an
error return code (still howing the help because it's a syntax error).

When a subcommand has no data arguments, then infer the need for a help
subcommand, it does not need a cmd.md specification. This subcommand should not
have an error exit.

The help should list all the available subcommands, options, and arguments which
includes the top-level usage command and the --usage flag, but not contents of
any USAGE docs. Do not list a usage subcommand under other commands.

Help is shown for any argument syntax issues. Usage / Semantic argument issues
do not show the help, but should provide enough context about the error that an
agent can work to fix the arguments.

### Usage

Maintain a docs/USAGE.txt file. It should describe in a minimal amount of words
how to use app in the most straightforward manner. The docs/USAGE.txt is
different than help, it is not a list of all subcommands, options, and
arguments. It should instead give example use of the tool and describe when to
use it.

Instruct the agent to explain design principles and conventions from the app's
specifications when users need them to understand how to use the app. Place
each explanation in the relevant usage document and keep it concise and
practical, with examples where helpful.

Also maintain `docs/USAGE_<subcommand>.txt` files for each subcommand
(recursively)

Provide a "usage" subcommand only at the top level. Support the --usage flag
on the main command and every subcommand. Each prints the relevant usage file.
Do not provide nested "usage" subcommands. If a usage argument is present,
then any other data arguments or flags are an error (show help).

Embed the usage docs into the application

## Shared code implementation checks

At the end of implementation, instruct the agent to ensure that each current
4sight internal database migration ID is built into the binary as an embedded
string available to the shared startup check, and that the embedded migration
chains include all migrations through those IDs.
