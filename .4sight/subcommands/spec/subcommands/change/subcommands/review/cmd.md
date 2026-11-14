# 4sight spec change review

Review commands manage saved specification change reviews.

## Commands

- `list [change-id]` lists the latest review for each reviewed change, or all
  review attempts for the specified change.
- `show [attempt]` shows a review for the current workspace change. With no
  argument it shows the latest attempt; a positive integer selects an attempt.
  A change ID may also be supplied before an optional attempt to select a
  different change.
- `agent prompt` generates instructions for a first review or an explicitly
  requested re-review.
- `agent response <change-id>` persists a review attempt supplied as JSON.
