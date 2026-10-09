# 4sight spec implement prompt [--ignore-workspace]

Called to generate a prompt to instruct an agent to read code spec and update
codebase

## Check Change State

If ignore-workspace flag was sent, skip the checks in this section.

Read the current change pointers.

If the specs in workspace does not match current workspace changes (the last
body of each file going up the change stack), then abort and tell end user to
import or sync the files.

If the submitted and workspace pointers are different, tell the user the changes
and their summaries since the submitted pointer, and that they can either submit
or send the ignore flag.

If the implemented and submitted pointers are the same, then nothing to be done,
stop after telling user nothing to do.

## Output

Tell agent to read all the design.md and cmd.md docs for top level and all
subcommands to inform the app behavior (list the files that exist)

Also list the schemas found in the `.4sight/schemas` directory telling the agent
to read those as well.

Then tell agent to review the application code spec and update code to match the
spec, but do not update the schema. If the schema needs a spec change or a
migration to be compatible with the code spec, stop and tell the user the gap
(schema spec or migration, and what kinds of changes are needed)

After successfully implementing the submitted change, instruct the agent to run
`4sight spec change implement agent done` to mark the implementation complete.
