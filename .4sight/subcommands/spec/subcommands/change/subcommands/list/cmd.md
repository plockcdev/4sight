# 4sight spec change list

Lists specification changes with their IDs and summaries.

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

This command does not modify the database.
