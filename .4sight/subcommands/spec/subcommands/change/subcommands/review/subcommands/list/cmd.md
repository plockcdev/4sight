# 4sight spec change review list `[change-id]`

Lists saved reviews without modifying the database.

With no argument, list each change that has at least one review, in ascending
change creation order. Show the change ID and summary from its latest review.
When a change has more than one review, also show the total number of review
attempts.

When `change-id` is supplied, list every review attempt for exactly that change
in ascending attempt-number order, showing the attempt number, summary, and
ready value for each. Report an error if the change does not exist or has no
reviews.
