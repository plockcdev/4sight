Unless specified, use go for the application, and for scripting needs, unless
specified otherwise, use nushell

If the specifications have logic problems or need clarifications before writing
code, stop and alert user.

## Coding Language Specifications

read `coding/languages/*.md` - general coding instructions for specific
languages (go, bash, python, nushell, rust, etc.)

## Help and Usage

When adding subcommands or command line arguments, ensure to update the help.

Help is also shown for "--help" after any subcommand or the main command.

When a subcommand has no data arguments, then infer the need for a help
subcommand, it does not need a cmd.md specification.

The help should list all the available subcommands, options, and arguments, but
not usage

Help is shown for any argument syntax issues. Usage / Semantic argument issues
do not show the help, but should provide enough context about the error that an
LLM can work to fix the arguments.

Maintain a docs/USAGE.txt file. It should describe in a minimal amount of words
how to use app in the most straightforward manner. The docs/USAGE.txt is
different than help, it is not a list of all subcommands, options, and
arguments. It should instead give example use of the tool and describe when to
use it.

Also maintain `docs/USAGE_<subcommand>.txt` files for each subcommand
(recursively)

the "usage" subcommand at the top level (or the --usage flag on the main command
or any subcommand) will simply print the relevant usage file.
