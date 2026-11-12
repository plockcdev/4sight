# 4sight spec sync-check

Checks if workspace file records match the on-disk specs and text assets in both
directions, reporting missing and extra files. Use the inherited spec design
for discovery and comparison rules, and the schema spec for file presence and
deletion semantics.

Output two sections, one for user to import, and other for user to export bodies
that are not matching.  The import section should call out any new files that would
be imported.

Example export command to overwrite what is on disk:
`4sight spec export <submcommand locator> <filename>`

Example import command to overwrite what is in db:
`4sight spec change import`

The locator is the subcommand, e.g. for the `spec implement prompt` subcommand:
`4sight spec export spec implement prompt cmd.md`
