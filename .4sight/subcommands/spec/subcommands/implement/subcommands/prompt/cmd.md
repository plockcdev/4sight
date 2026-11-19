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

Tell agent to read:

- all the design spec for top level and all subcommands
- all the schemas in `.4sight/schemas`

Do not list text assets in the code spec list. Instruct the agent to inspect text asset
changes in the composite diff and read relevant text asset files from disk.

## Check Spec Problems

The code and schema specifications work together.

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

### Shared implementation instructions

Instruct the agent to compare implemented and submitted spec bodies, including
schema additions, changes, and deletions, and inspect existing migration state.
Do not edit or import specs during implementation. Shared preflight is read-only.
If an internal upgrade predating the requested implementation is required to
access the project, report the dependency and stop; do not upgrade the working
project during preparation or testing. Use isolated test projects and databases;
stop if the requested scope cannot be safely performed.

### Full implementation

For every database whose submitted schema requires a migration, first run
`4sight migrate implement prompt <path to schema spec relative to schemas dir>`
and follow it to prepare and submit the migration, reusing an existing submitted
migration when its schema metadata matches. Do not generate migrations for
unchanged schemas merely because they exist. Prepare internal and application
migrations according to their respective migration paths. For deleted schemas, pass their former paths to the migration prompt and prepare
retirement migrations. If a new database or migration dependency cannot be handled by the migration prompt
without changing specs, stop and report the issue rather than inventing a schema
or marking the change implemented. Check the migration chain and dependencies;
if the requested ordering cannot safely support the change, stop and explain.

After preparing the migrations and their required tracking metadata, update the
code implementation using the code instructions below. Do not apply the new
migrations to the working project during preparation, coding, or testing. Run
new code that needs the target schema against isolated ephemeral databases, not
the working database. Configure the entire test project, database endpoints,
migration tracking, and migration symlinks so tests cannot alter working state.

When schemas or migrations change, apply the target migration chains to fresh
ephemeral databases and run the new code's relevant tests. Separately construct
representative previous-version databases with migration histories and data,
apply the pending migrations, and verify schema and data preservation, tracking
records, migration ordering, and relevant new-code behavior. Cover migration
failure and retry or repair behavior appropriate to the backend. Use representative
fixtures or safe copies; do not test against a live database. Resolve failures
and rerun the affected checks before proceeding. For a code-only change with no
schema migration, perform the relevant code verification without requiring an
unrelated migration test.

Once code and migration checks pass, migrate the working project and verify the
result. For 4sight self-hosting, use the newly built binary's `4sight upgrade`
with its validated embedded internal chains; reconcile source `current` links
from the authoritative recorded state as specified by upgrade. For ordinary
application databases, run `4sight migrate up` after any required internal
upgrade. Verify applied histories and applicable current links. Then run
`4sight spec change implement agent done` only if all required work succeeds.
If migration or verification fails, report the failure, partial application,
and any repair needed; do not mark the change complete. The older binary may
reject the upgraded internal history. If the migration, deployment, or testing
cannot safely follow this order, stop and explain the required staged rollout.

When implementing this behavior, verify application tracking-table creation,
internal upgrade gating, and migration ordering.

### Code-only implementation

For an explicitly code-only request, follow the code instructions below; do not
generate or apply migrations. If the code requires a pending schema migration,
stop and report the dependency. After verification, run `4sight spec change
implement agent done` only if all other required work is already complete;
otherwise report the remaining work without marking the change implemented.

### Schema-only implementation

For an explicitly schema-only request, use `4sight migrate implement prompt
<path to schema spec relative to schemas dir>` and follow its schema-only
instructions. Report the pending migration and how to apply it; do not apply it
or mark the submitted change implemented as part of this request.

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
includes the usage command, but not contents of any USAGE docs.

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

the "usage" subcommand at the top level (or the --usage flag on the main command
or any subcommand) will simply print the relevant usage file. If a usage
argument is present, then any other data arguments or flags are an error (show
help)

Embed the usage docs into the application

## Shared code implementation checks

At the end of implementation, instruct the agent to ensure that each current
4sight internal database migration ID is built into the binary as an embedded
string available to the shared startup check, and that the embedded migration
chains include all migrations through those IDs.

