# 4sight spec sync-check

Checks if the collection of workspace bodies in the database matches the on disk
spec files in both directions,
but compare without considering whitespace at end of lines and empty lines at the
end of the document (as we clear those on import)

Output two sections, one for user to import, and other for user to export bodies
that are not matching.

Example export command to overwrite what is on disk:
`4sight spec export <submcommand locator> <filename>`

Example import command to overwrite what is in db:
`4sight spec change import`

The locator is the subcommand, e.g. for the `spec implement prompt` subcommand:
`4sight spec export spec implement prompt cmd.md`
