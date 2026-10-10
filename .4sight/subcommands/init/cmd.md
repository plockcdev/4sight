# 4sight init

Initialize a git repo to have source, schema, and specs managed by 4sight.

Follow the instructions below, implementation must be safely idempotent.

Any failures should indicate the problem and a suggestion on how to solve.

Source paths are relative to the init command directory, target paths are
relative to the git repo root.

## Verify git root

Verify that the current directory is the root of a git repo. If not, then see if
we are in a git repo and instruct on where to run the init command. If not in a
git repo, then give user an example of how to initialize a git repo in a target
directory

## .4sight directory

Create target directory if missing.

## DB

Run `dolt init` inside `.4sight/db` for first time setup only.

## Write Embedded Assets

Create symlinks in an `4sight` directory under the init implementation directory
pointing to the files in the assets directory, and then embed the directory into
the 4sight binary.

Leverage the asset contents as instructed, source comes first:

```
- root_AGENTS.md:
    contents appears once at the end of the repo root AGENTS.md within armor like
    "---- BEGIN 4SIGHT ----" and ending with "---- END 4SIGHT ----"
- 4sight_AGENTS.md:
    overwrite as AGENTS.md in the .4sight directory
- coding.md:
    if .4sight/coding.md does not exist, create it, otherwise leave alone
```
