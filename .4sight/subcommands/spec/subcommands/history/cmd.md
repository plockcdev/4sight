# 4sight spec history

With no file path, list all specification changes with their IDs and summaries.

Read changes from the spec database. For each change, display its ID and the
summary from its saved overall change review. If a change has no saved review,
display `no summary yet`.

List changes in ascending creation order, with the oldest change first and the
newest change last, so older entries scroll off the screen as the user moves
down toward newer changes. Use the change ID as a stable tie-breaker.
When creation times are equal, sort change IDs in ascending lexicographic
order.

Render a human-readable list with one entry per change. If there are no changes,
print `no changes found`.

Render a left gutter with two marker columns before each change entry: `I` for
the change identified by the implemented pointer and `S` for the change
identified by the submitted pointer. Leave a marker column blank for all other
entries. If both pointers identify the same change, show both markers on that
entry (`I S`).

This command does not modify the database.

## File history

Show the history of a tracked specification or text asset file on the current
workspace change's ancestry. Resolve a relative file argument against the
current working directory; accept an absolute path as well. Normalize `.` and
`..` components lexically, then convert the result to the corresponding path
stored in the spec database relative to the project's `.4sight` directory. For
example, from the project root, `.4sight/coding.md` identifies the same file as
`coding.md` when run from `.4sight`. Reject paths outside `.4sight` with a
clear error. Do not require the file or its parent directories to exist on disk:
historical and deleted paths must remain addressable. This command reads the
database and does not modify it or inspect unimported disk changes.

Resolve the path against file changes on the workspace ancestry, including
historical paths and deleted files. If it identifies more than one distinct
file identity on that ancestry (for example, a path reused after deletion),
report that the path is ambiguous and do not display a partial history. If no
identity matches, report that the file has no tracked history. Follow the
resolved identity through renames, but do not include the history of a copy's
source or other identities.

List every change on that ancestry with a file change for the resolved identity,
including additions, deletions, and renames, in ancestry order from oldest to
newest. For each entry show the change ID and the description from that change's
latest overall change review, selected by highest attempt number. If there is
no review, show `no review yet`; if the latest review has an empty description,
show `no description`. Do not substitute an older review or a file-level review.

Render a human-readable list, with the change ID identifying each entry. Wrap
review descriptions to a maximum of 80 columns including indentation, breaking
at whitespace where possible. Preserve paragraph breaks; split an individual
word longer than the available width as needed to maintain the limit. Other
output lines must also fit within 80 columns.
