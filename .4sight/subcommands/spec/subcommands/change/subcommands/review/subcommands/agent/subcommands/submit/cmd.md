# 4sight spec change review agent submit

If the current workspace already has a review, have agent stop and tell the user how
to submit a request or import a spec change to allow a review.

Read stdin json (timeout after 1 second, and indicate the submission requirement)

Create a new change_review attached to the current workspace change

The user must submit all the other data for the change review in the body.

Create a new file_change_review for each file review in the body, setting the change
to the current workspace change and all other fields are coming from the body.

Each attribute must be present even if empty (see schema for details)
