package events

import (
	"time"

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

// DocumentMovedEvent is emitted when a document moves to a new collection or is
// renamed (both rewrite the key). CollectionID is the new parent (nil = app
// root); Key is the new materialized key.
type DocumentMovedEvent struct {
	CollectionID *string
	DocumentID   string
	TenantID     string
	Key          string
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
	documentMovedSignal       = capitan.NewSignal("barbara.document.moved", "Document moved to a new collection or renamed")
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
	Moved       sum.Event[DocumentMovedEvent]
	Deleted     sum.Event[DocumentDeletedEvent]
	Published   sum.Event[DocumentPublishedEvent]
	Unpublished sum.Event[DocumentUnpublishedEvent]
	RolledBack  sum.Event[DocumentRolledBackEvent]
	TagAdded    sum.Event[DocumentTagAddedEvent]
	TagRemoved  sum.Event[DocumentTagRemovedEvent]
}{
	Created:     sum.NewInfoEvent[DocumentCreatedEvent](documentCreatedSignal),
	Renamed:     sum.NewInfoEvent[DocumentRenamedEvent](documentRenamedSignal),
	Moved:       sum.NewInfoEvent[DocumentMovedEvent](documentMovedSignal),
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

// --- Collection ---

// CollectionCreatedEvent is emitted when a collection is created.
type CollectionCreatedEvent struct {
	CollectionID string
	TenantID     string
	AppID        string
	Name         string
}

// CollectionRenamedEvent is emitted when a collection's name changes. Name is
// the new name.
type CollectionRenamedEvent struct {
	CollectionID string
	TenantID     string
	AppID        string
	Name         string
}

// CollectionMovedEvent is emitted when a collection moves to a new parent.
// ParentID is the new parent (nil = app root).
type CollectionMovedEvent struct {
	ParentID     *string
	CollectionID string
	TenantID     string
	AppID        string
}

// CollectionDeletedEvent is emitted when a collection is deleted.
type CollectionDeletedEvent struct {
	CollectionID string
	TenantID     string
	AppID        string
}

var (
	collectionCreatedSignal = capitan.NewSignal("barbara.collection.created", "Collection created")
	collectionRenamedSignal = capitan.NewSignal("barbara.collection.renamed", "Collection name changed")
	collectionMovedSignal   = capitan.NewSignal("barbara.collection.moved", "Collection moved to a new parent")
	collectionDeletedSignal = capitan.NewSignal("barbara.collection.deleted", "Collection deleted")
)

// Collection groups the collection lifecycle events.
var Collection = struct {
	Created sum.Event[CollectionCreatedEvent]
	Renamed sum.Event[CollectionRenamedEvent]
	Moved   sum.Event[CollectionMovedEvent]
	Deleted sum.Event[CollectionDeletedEvent]
}{
	Created: sum.NewInfoEvent[CollectionCreatedEvent](collectionCreatedSignal),
	Renamed: sum.NewInfoEvent[CollectionRenamedEvent](collectionRenamedSignal),
	Moved:   sum.NewInfoEvent[CollectionMovedEvent](collectionMovedSignal),
	Deleted: sum.NewInfoEvent[CollectionDeletedEvent](collectionDeletedSignal),
}

// --- Release ---

// ReleaseCutEvent is emitted when a release is cut (the app's pointer moved).
type ReleaseCutEvent struct {
	ReleaseID string
	AppID     string
	TenantID  string
	Number    int
}

// ReleaseRolledBackEvent is emitted when a rollback cuts a new release copying an
// older one's entries forward.
type ReleaseRolledBackEvent struct {
	ReleaseID string
	AppID     string
	TenantID  string
	Number    int
}

var (
	releaseCutSignal        = capitan.NewSignal("barbara.release.cut", "Release cut")
	releaseRolledBackSignal = capitan.NewSignal("barbara.release.rolledback", "Release rolled back")
)

// Release groups the release lifecycle events.
var Release = struct {
	Cut        sum.Event[ReleaseCutEvent]
	RolledBack sum.Event[ReleaseRolledBackEvent]
}{
	Cut:        sum.NewInfoEvent[ReleaseCutEvent](releaseCutSignal),
	RolledBack: sum.NewInfoEvent[ReleaseRolledBackEvent](releaseRolledBackSignal),
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

// AssetWrittenEvent is emitted when an asset is stored (put/overwrite). An
// overwrite carries the previous object's size and last-modified time, so a
// consumer can move it between buckets rather than count it twice.
type AssetWrittenEvent struct {
	// PrevLastModified is when the overwritten object was written (zero
	// unless Overwrote, or when the bucket did not report it).
	PrevLastModified time.Time
	Key              string
	TenantID         string
	AppID            string
	ContentType      string
	Size             int64
	// PrevSize is the overwritten object's size (0 unless Overwrote).
	PrevSize int64
	// Overwrote is true when the key already held an object.
	Overwrote bool
}

// AssetDeletedEvent is emitted when an asset is deleted, with the removed
// object's metadata.
type AssetDeletedEvent struct {
	LastModified time.Time
	Key          string
	TenantID     string
	AppID        string
	ContentType  string
	Size         int64
}

// AssetMovedEvent is emitted when an asset's key changes — moved to another
// folder, renamed, or both.
type AssetMovedEvent struct {
	Key         string
	NewKey      string
	TenantID    string
	AppID       string
	ContentType string
	Size        int64
}

// AssetBookkeepingFailedEvent is emitted when an asset write or delete landed
// in the bucket but its bookkeeping update did not. The object is the truth
// and the request succeeds; the rows stay stale until the next rebuild.
type AssetBookkeepingFailedEvent struct {
	Err      error
	Key      string
	TenantID string
	AppID    string
}

var (
	assetWrittenSignal           = capitan.NewSignal("barbara.asset.written", "Asset written to object storage")
	assetDeletedSignal           = capitan.NewSignal("barbara.asset.deleted", "Asset deleted from object storage")
	assetMovedSignal             = capitan.NewSignal("barbara.asset.moved", "Asset moved to a new key")
	assetBookkeepingFailedSignal = capitan.NewSignal("barbara.asset.bookkeeping.failed", "Asset bookkeeping update failed after the bucket write")
)

// Asset groups the asset lifecycle events.
var Asset = struct {
	Written           sum.Event[AssetWrittenEvent]
	Deleted           sum.Event[AssetDeletedEvent]
	Moved             sum.Event[AssetMovedEvent]
	BookkeepingFailed sum.Event[AssetBookkeepingFailedEvent]
}{
	Written:           sum.NewInfoEvent[AssetWrittenEvent](assetWrittenSignal),
	Deleted:           sum.NewInfoEvent[AssetDeletedEvent](assetDeletedSignal),
	Moved:             sum.NewInfoEvent[AssetMovedEvent](assetMovedSignal),
	BookkeepingFailed: sum.NewErrorEvent[AssetBookkeepingFailedEvent](assetBookkeepingFailedSignal),
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
