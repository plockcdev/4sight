# 4sight spec export `<subcommand ...>` `<filename>`

exports the named spec for `<subcommand ...>` to `<filename>` from the database

For example, to export the command spec for the command `spec export`:
`4sight spec export spec export cmd.md`

Export also restores tracked text asset files at their current paths from the
database. Follow the inherited spec design for text asset contents and path
discovery, and the schema spec for current path and deletion semantics. Deleted
files are not written. Exporting a deleted file as
a spec is an error.


