# 4sight spec change design

## Change pointer lifecycle

The workspace pointer tracks spec edits that can be reviewed and should line up
with the workspace except for unimported specs.

The submitted pointer tracks the latest spec change accepted for implementation.

The implemented pointer tracks the latest submitted change whose implementation
is complete.
