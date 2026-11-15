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
Do not edit or import specs during implementation. Account for the shared
startup preflight: newly built 4sight binaries may apply embedded internal
migrations on startup. Do not invoke them against the working project if doing
so would apply migrations outside the requested scope or before the permitted
step. Use isolated test projects as needed; if the requested scope cannot be
safely performed, stop and explain.

### Full implementation

For every database whose submitted schema requires a migration, first run
`4sight migrate implement prompt <path to schema spec relative to schemas dir>`
and follow it to create and submit the migration. Do not generate migrations for
unchanged schemas merely because they exist. If a schema was deleted, or a new
database or migration dependency cannot be handled by the migration prompt
without changing specs, stop and report the issue rather than inventing a schema
or marking the change implemented. Check the migration chain and dependencies;
if the requested ordering cannot safely support the change, stop and explain.

After preparing all required migrations, update the code implementation using
the code instructions below. Do not apply the
new migrations before the code changes for a full implementation. Code and tests
must be designed so they do not require running the new code against the old
schema; use an isolated migrated test database when necessary. If this is not
possible, stop and explain the required staged rollout.

After code changes and appropriate verification, run `4sight migrate up` and
verify success before marking the change complete. If it fails, report the
failure and any partial application or repair required; do not mark the change
complete. Run `4sight spec change implement agent done` only after all required
work succeeds. If startup preflight or migration ordering makes this sequence
impossible, stop and report the conflict.

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

