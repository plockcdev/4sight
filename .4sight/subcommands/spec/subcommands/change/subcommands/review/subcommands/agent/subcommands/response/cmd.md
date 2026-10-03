# 4sight spec change review agent response

Allows agent to persist its generated review content as directed by the agent prompt command.

Follows the Agent Prompt and Response Design

Create a change review for the current workspace change pointer, create one or more
complete file change reviews, and create an associated body summary for each file change reviewed.

Db links should be written by this command the agent shouldn't have to see internal
ids as paths should be sufficient to identify db objects as we know the workspace change
pointer.

If the current workspace pointer already has a change review, tell agent to stop, and have it tell
the user how to submit a request or import a spec change to allow a review.

Read the response data through stdin (if 1 second passes waiting for data before EOF, error out,
and indicate the submission requirement)

For each response file change review, ensure the filesystem and body table agree for the
current workspace change pointer, otherwise tell user to import spec changes and stop.

If a response file change is missing a body summary, reject the request and tell agent why.

For each file review:

- the file change for the review is determined from the provided file change path and the
  current workspace change or it's ancestry
