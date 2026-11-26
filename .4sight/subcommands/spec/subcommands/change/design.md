# Guidelines when changing design

Determine the scope of the request: is it for a specific command behavior, a app
wide data representation change, or a component level?

When updating the design/coding/schema/etc., choose file(s) of the appropriate
scope to edit Always _read_ the parent scoped design docs, and the schema design
to ensure changes are aligned.

Do not write to design of higher scope unless specifically given that scope.

If asked to edit a higher scope design document, all design docs underneath that
design doc must be evaluated for change as they need to obey the higher level
design.

A mismatch of an edited design file to a parent scoped design file should be
called out in review, even when not introduced.
