## 4sight spec chage review design

The user will have made a change request (prompt) or done an import,
and we want the agent to review the design changes with respect to
the latest submitted change and submit those reviews in a cli response.

The current workspace change is what is getting reviewed, the review
only commences when the db and filesystem agrees.

The response format, each changed file should only have 1 review, each db attribute
should be filled out that the submit command can't figure out.
The body summary is for the current workspace body for the file.

```
{
  "review": {
    ... attributes of change_review
  }
  "file_reviews": [
    {... attributes of file_change_review, and full body summary},
    {... attributes of file_change_review, and full body summary}
  ]
}


