// Package events provides event definitions for the application.
package events

import "github.com/zoobz-io/capitan"

// Startup signals for server lifecycle.
// These are direct capitan signals (not sum.Event) since they're
// operational events, not domain lifecycle events for consumers.
var (
	StartupDatabaseConnected = capitan.NewSignal("barbara.startup.database.connected", "Database connection established")
	StartupStorageConnected  = capitan.NewSignal("barbara.startup.storage.connected", "Object storage connection established")
	StartupSearchConnected   = capitan.NewSignal("barbara.startup.search.connected", "OpenSearch connection established")
	StartupIndicesReady      = capitan.NewSignal("barbara.startup.indices.ready", "OpenSearch indices ensured")
	StartupJobsStarted       = capitan.NewSignal("barbara.startup.jobs.started", "Jobs pipeline runner started")
	StartupServicesReady     = capitan.NewSignal("barbara.startup.services.ready", "All services registered")
	StartupOTELReady         = capitan.NewSignal("barbara.startup.otel.ready", "OpenTelemetry providers initialized")
	StartupApertureReady     = capitan.NewSignal("barbara.startup.aperture.ready", "Aperture observability bridge initialized")
	StartupServerListening   = capitan.NewSignal("barbara.startup.server.listening", "HTTP server listening")
	StartupFailed            = capitan.NewSignal("barbara.startup.failed", "Server startup failed")
)

// Startup field keys for direct emission.
var (
	StartupPortKey    = capitan.NewIntKey("port")
	StartupWorkersKey = capitan.NewIntKey("workers")
	StartupErrorKey   = capitan.NewErrorKey("error")
)
