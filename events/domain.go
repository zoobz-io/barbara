package events

import (
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/sum"
)

// Domain lifecycle events, emitted AFTER the originating transaction commits so
// a listener never sees an event for work that rolled back. They are typed
// sum.Event payloads grouped by entity, distinct from the operational capitan
// signals in startup.go. Consumers listen for observability and downstream
// reactions; the emitters are the feature paths in database/stores and the jobs
// pipeline.

// --- Document ---

// DocumentCreatedEvent is emitted when a document is created.
type DocumentCreatedEvent struct {
	DocumentID string
	TenantID   string
	Key        string
}

// DocumentRenamedEvent is emitted when a document's key changes. Key is the new key.
type DocumentRenamedEvent struct {
	DocumentID string
	TenantID   string
	Key        string
}

// DocumentDeletedEvent is emitted when a document is deleted.
type DocumentDeletedEvent struct {
	DocumentID string
	TenantID   string
}

// DocumentPublishedEvent is emitted when a document's published pointer moves to
// a version (publish or rollback both point the document at a version; the
// distinct RolledBack event names the intent).
type DocumentPublishedEvent struct {
	DocumentID string
	TenantID   string
	VersionID  string
}

// DocumentUnpublishedEvent is emitted when a document's published pointer is cleared.
type DocumentUnpublishedEvent struct {
	DocumentID string
	TenantID   string
}

// DocumentRolledBackEvent is emitted when a document is republished at an older version.
type DocumentRolledBackEvent struct {
	DocumentID string
	TenantID   string
	VersionID  string
}

// DocumentTagAddedEvent is emitted when a tag is added to a document (a no-op
// re-add emits nothing).
type DocumentTagAddedEvent struct {
	DocumentID string
	TenantID   string
	Tag        string
}

// DocumentTagRemovedEvent is emitted when a tag is removed from a document (a
// no-op remove emits nothing).
type DocumentTagRemovedEvent struct {
	DocumentID string
	TenantID   string
	Tag        string
}

var (
	documentCreatedSignal     = capitan.NewSignal("barbara.document.created", "Document created")
	documentRenamedSignal     = capitan.NewSignal("barbara.document.renamed", "Document key changed")
	documentDeletedSignal     = capitan.NewSignal("barbara.document.deleted", "Document deleted")
	documentPublishedSignal   = capitan.NewSignal("barbara.document.published", "Document published")
	documentUnpublishedSignal = capitan.NewSignal("barbara.document.unpublished", "Document unpublished")
	documentRolledBackSignal  = capitan.NewSignal("barbara.document.rolledback", "Document rolled back to an older version")
	documentTagAddedSignal    = capitan.NewSignal("barbara.document.tag_added", "Tag added to a document")
	documentTagRemovedSignal  = capitan.NewSignal("barbara.document.tag_removed", "Tag removed from a document")
)

// Document groups the document lifecycle events.
var Document = struct {
	Created     sum.Event[DocumentCreatedEvent]
	Renamed     sum.Event[DocumentRenamedEvent]
	Deleted     sum.Event[DocumentDeletedEvent]
	Published   sum.Event[DocumentPublishedEvent]
	Unpublished sum.Event[DocumentUnpublishedEvent]
	RolledBack  sum.Event[DocumentRolledBackEvent]
	TagAdded    sum.Event[DocumentTagAddedEvent]
	TagRemoved  sum.Event[DocumentTagRemovedEvent]
}{
	Created:     sum.NewInfoEvent[DocumentCreatedEvent](documentCreatedSignal),
	Renamed:     sum.NewInfoEvent[DocumentRenamedEvent](documentRenamedSignal),
	Deleted:     sum.NewInfoEvent[DocumentDeletedEvent](documentDeletedSignal),
	Published:   sum.NewInfoEvent[DocumentPublishedEvent](documentPublishedSignal),
	Unpublished: sum.NewInfoEvent[DocumentUnpublishedEvent](documentUnpublishedSignal),
	RolledBack:  sum.NewInfoEvent[DocumentRolledBackEvent](documentRolledBackSignal),
	TagAdded:    sum.NewInfoEvent[DocumentTagAddedEvent](documentTagAddedSignal),
	TagRemoved:  sum.NewInfoEvent[DocumentTagRemovedEvent](documentTagRemovedSignal),
}

// --- App ---

// AppCreatedEvent is emitted when an app is created.
type AppCreatedEvent struct {
	AppID    string
	TenantID string
	Name     string
}

// AppRenamedEvent is emitted when an app's name changes. Name is the new name.
type AppRenamedEvent struct {
	AppID    string
	TenantID string
	Name     string
}

// AppDeletedEvent is emitted when an app is deleted.
type AppDeletedEvent struct {
	AppID    string
	TenantID string
}

var (
	appCreatedSignal = capitan.NewSignal("barbara.app.created", "App created")
	appRenamedSignal = capitan.NewSignal("barbara.app.renamed", "App name changed")
	appDeletedSignal = capitan.NewSignal("barbara.app.deleted", "App deleted")
)

// App groups the app lifecycle events.
var App = struct {
	Created sum.Event[AppCreatedEvent]
	Renamed sum.Event[AppRenamedEvent]
	Deleted sum.Event[AppDeletedEvent]
}{
	Created: sum.NewInfoEvent[AppCreatedEvent](appCreatedSignal),
	Renamed: sum.NewInfoEvent[AppRenamedEvent](appRenamedSignal),
	Deleted: sum.NewInfoEvent[AppDeletedEvent](appDeletedSignal),
}

// --- Version ---

// VersionSavedEvent is emitted when a new version of a document is saved.
type VersionSavedEvent struct {
	VersionID     string
	DocumentID    string
	TenantID      string
	VersionNumber int
}

var versionSavedSignal = capitan.NewSignal("barbara.version.saved", "Document version saved")

// Version groups the version lifecycle events.
var Version = struct {
	Saved sum.Event[VersionSavedEvent]
}{
	Saved: sum.NewInfoEvent[VersionSavedEvent](versionSavedSignal),
}

// --- Asset ---

// AssetWrittenEvent is emitted when an asset is stored (put/overwrite).
type AssetWrittenEvent struct {
	Key         string
	TenantID    string
	ContentType string
	Size        int64
}

// AssetDeletedEvent is emitted when an asset is deleted.
type AssetDeletedEvent struct {
	Key      string
	TenantID string
}

var (
	assetWrittenSignal = capitan.NewSignal("barbara.asset.written", "Asset written to object storage")
	assetDeletedSignal = capitan.NewSignal("barbara.asset.deleted", "Asset deleted from object storage")
)

// Asset groups the asset lifecycle events.
var Asset = struct {
	Written sum.Event[AssetWrittenEvent]
	Deleted sum.Event[AssetDeletedEvent]
}{
	Written: sum.NewInfoEvent[AssetWrittenEvent](assetWrittenSignal),
	Deleted: sum.NewInfoEvent[AssetDeletedEvent](assetDeletedSignal),
}

// --- Index (jobs pipeline) ---

// IndexWriteSucceededEvent is emitted when the jobs pipeline lands an OpenSearch
// write for a document (index or delete).
type IndexWriteSucceededEvent struct {
	DocumentID string
	Operation  string
}

// IndexWriteFailedEvent is emitted when the jobs pipeline exhausts its retries
// and the OpenSearch write fails terminally.
type IndexWriteFailedEvent struct {
	DocumentID string
	Operation  string
	Error      string
}

var (
	indexWriteSucceededSignal = capitan.NewSignal("barbara.index.write_succeeded", "OpenSearch write landed")
	indexWriteFailedSignal    = capitan.NewSignal("barbara.index.write_failed", "OpenSearch write failed terminally")
)

// Index groups the OpenSearch-write outcome events, emitted from the jobs
// pipeline where the write actually resolves.
var Index = struct {
	WriteSucceeded sum.Event[IndexWriteSucceededEvent]
	WriteFailed    sum.Event[IndexWriteFailedEvent]
}{
	WriteSucceeded: sum.NewInfoEvent[IndexWriteSucceededEvent](indexWriteSucceededSignal),
	WriteFailed:    sum.NewWarnEvent[IndexWriteFailedEvent](indexWriteFailedSignal),
}
