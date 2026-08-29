package models

// Document lifecycle statuses, derived from the app's current release rather
// than stored on the document. A document is published iff the current release
// carries it; if the release carries an older version than the document's head,
// there is a newer draft.
const (
	StatusDraft                   = "draft"
	StatusPublished               = "published"
	StatusPublishedWithNewerDraft = "published-with-newer-draft"
)
