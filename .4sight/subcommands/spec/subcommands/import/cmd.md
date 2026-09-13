# 4sight spec import `<spec file location description>`

imports a spec file into 4sight persistence

the spec file location description is the command/subcommand path as
separate args, then the file name, for example, for the cmd.md spec for
"myprog foo bar", the import command is:
`4sight spec import myprog foo bar cmd.md`

If the file does not exist, fail

Create an body entry for the imported spec, remove trailing whitespace
from lines, keep empty lines except at the end of the document

