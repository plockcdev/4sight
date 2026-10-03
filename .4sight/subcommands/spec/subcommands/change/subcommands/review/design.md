## 4sight spec chage review design

After the user has imported design file changes (either directly or indirectly
through a request), the agent will make a review agent prompt.

Agent will be instructed to perform a high level change review,
and per file qualitative review and provide an agent response.

The review is for the current "workspace" change pointer with respect to the current "submitted" change pointer while the filesystem and db agree on contents.
