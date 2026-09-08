# 4sight spec implement prompt

Called to generate a prompt to instruct an agent to read code spec and update codebase

## Output

Tell agent to read all the design.md and cmd.md docs for top level and all subcommands
to inform the app behavior
(list the files that exist)

Also list the schemas found in the `.4sight/schemas` directory telling the agent
to read those as well.

Then tell agent to review the application code spec and update code to match the spec,
but do not update the schema.  If the schema needs a spec change or a migration to be compatible with the code spec, stop and tell the user the gap (schema spec or
migration, and what kinds of changes are needed)
