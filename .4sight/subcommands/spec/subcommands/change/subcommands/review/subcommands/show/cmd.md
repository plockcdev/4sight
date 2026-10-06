# 4sight spec change review show `[change-id]`

Displays a saved specification change review without modifying the database.

If `change-id` is omitted, use the current workspace change. If it is provided,
look up the review for exactly that change.

If the change ID is unknown, report that the change does not exist. If the change
exists but has no saved review, report that no review is available for it.

Render a human-readable report with these sections:

- Change ID
- Overall review: why, summary, description, details, review, and ready
- File reviews: one entry for every file review linked to that overall review,
  showing the spec path, why, summary, description, details, and body summary

Resolve each file path through its file change. Sort file review entries by
spec path in ascending lexical order so output is deterministic. Show empty
description or details as empty values. The body summary is the snapshot stored
with the file review, so it remains accurate for historical reviews.
