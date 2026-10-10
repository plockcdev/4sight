# 4sight spec implement prompt [--ignore-workspace]

Called to generate a prompt to instruct an agent to read code spec and update
codebase

## Check Change State

Read the current change pointers.

### Check submitted vs workspace pointer

If the submitted and workspace pointers are different, abort and tell the user
the changes and their summaries since the submitted pointer, and that they can
either submit or send the ignore flag.

### Check implemented vs submitted

If the implemented and submitted pointers are the same, then nothing to be done,
stop after telling user nothing to do.

### Check matching

If ignore-workspace flag was sent, skip the checks in this section.

If the specs in workspace does not match current workspace changes (the last
body of each file going up the change stack), then abort and tell end user to
import or sync the files.

## Read Specs

Tell agent to read:

- all the design.md and cmd.md docs for top level and all
subcommands
- all the schemas in `.4sight/schemas`

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

Also separately list the schemas found in the `.4sight/schemas` directory
telling the agent to read those as well, group by the db type.

Then tell agent update code implementation, but do not update the schema, with
additional instructions from the subsections below:

### Command Help

Update the help when the subcommand structure changes such as changed arguments,
new or removed subcommands.

Help is also shown for "--help" after any subcommand or the main command with a
clean exit code. When help argument is used, any other data argument causes an
error return code (still howing the help because it's a syntax error).

When a subcommand has no data arguments, then infer the need for a help
subcommand, it does not need a cmd.md specification. This subcommand should not
have an error exit.

The help should list all the available subcommands, options, and arguments, but
not usage

Help is shown for any argument syntax issues. Usage / Semantic argument issues
do not show the help, but should provide enough context about the error that an
agent can work to fix the arguments.

### Usage

Maintain a docs/USAGE.txt file. It should describe in a minimal amount of words
how to use app in the most straightforward manner. The docs/USAGE.txt is
different than help, it is not a list of all subcommands, options, and
arguments. It should instead give example use of the tool and describe when to
use it.

Also maintain `docs/USAGE_<subcommand>.txt` files for each subcommand
(recursively)

the "usage" subcommand at the top level (or the --usage flag on the main command
or any subcommand) will simply print the relevant usage file. If a usage
argument is present, then any other data arguments or flags are an error (show
help)

## Completing Implmentation

After successfully implementing the submitted change, instruct the agent to run
`4sight spec change implement agent done` to mark the implementation complete.
