# 4sight spec sync-check

Checks if the collection of bodies in the database matches the on disk files

Output two sections, one for user to import, and other for user to export bodies
that are not matching.

Example export command to overwrite what is on disk:
`4sight spec export <submcommand locator> <filename>` 

Example import command to overwrite what is in db:
`4sight spec import <submcommand locator> <filename>` 

The locator is the subcommand, e.g. for the `spec implement prompt` subcommand:
`4sight spec export spec implement prompt cmd.md` 
