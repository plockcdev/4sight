# 4sight spec change review agent response

Allows agent to persist its generated review content as directed by the agent
prompt command.

Follows the Agent Prompt and Response Design

Create a change review for the current workspace change pointer, create one or
more complete file change reviews, and save each associated body summary as a
snapshot.

Db links should be written by this command the agent shouldn't have to see
internal ids as paths should be sufficient to identify db objects as we know the
workspace change pointer.

If the current workspace pointer already has a change review, tell agent to
stop, and have it tell the user how to submit a request or import a spec change
to allow a review.

Read the response data through stdin (if 1 second passes waiting for data before
EOF, error out, and indicate the submission requirement)

For each response file change review, ensure the filesystem and body table agree
for the current workspace change pointer, otherwise tell user to import spec
changes and stop.

If a response file change is missing a body summary, reject the request and tell
agent why.

Validate the entire response before writing. Persist the overall change review
and all file change reviews in one transaction. Each file change review must
reference both the overall change review being created and the file determined
from its path within the current workspace change or its ancestry. Persist
`body_summary` on the file change review as the summary snapshot and update the
matching workspace body summary.

For each path changed between the submitted and workspace pointers, select the
latest file_change for that path in that pointer range and require exactly one
review linked to that file_change. Reject duplicate reviews, missing reviews,
and reviews linked to file changes outside the range. Print a success message
identifying the reviewed change after commit.
