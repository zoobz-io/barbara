package models

// DocumentHead pairs a document with its head (latest) version. Head is nil when
// the document has no versions yet — an empty document, not an error. It is the
// read shape behind opening a document for editing and behind the draft/published
// status on document responses: the status compares the document's published
// pointer against the head.
type DocumentHead struct {
	Document *Document
	Head     *Version
}
