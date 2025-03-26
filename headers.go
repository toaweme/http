package http

import "fmt"

// ClientUserAgentHeaderName is the standard header for identifying the client software making the request
// in browsers, this is inherited from navigator.userAgent
// in cli, mobile, and desktop apps, this should be set manually (e.g. "awee-cli/1.0.0")
// maps to: http.user_agent (otel semantic convention)
const ClientUserAgentHeaderName = "User-Agent"

// ClientPlatformHeaderName identifies the type of client making the request (e.g. web, mobile, desktop, cli, service)
// helps disambiguate clients that may share the same app version
// maps to: baggage key "client.platform"
const ClientPlatformHeaderName = "X-Client-Platform"

// ClientAppVersionHeaderName identifies the version of the app making the request
// used for debugging, feature flags, and context tagging
// should be used alongside client platform to avoid ambiguity
// maps to: baggage key "client.version"
const ClientAppVersionHeaderName = "X-Client-Version"

// ClientIDHeaderName is a persistent identifier for the client install or device
// stored locally (e.g. config, localStorage, secure storage), and stable across app restarts
// changes only if the app is reinstalled or reset
// maps to: baggage key "client.client_id"
const ClientIDHeaderName = "X-Client-ID"

// ClientSessionIDHeaderName identifies a session or user login context
// optional, but useful to correlate related requests within the same user session
// maps to: baggage key "client.session_id"
const ClientSessionIDHeaderName = "X-Session-ID"

// ClientRequestIDHeaderName is a unique identifier for the individual request
// used to trace a single request across services
// maps to: traceparent (otel standard); can be included for systems that don't yet support it
const ClientRequestIDHeaderName = "X-Request-ID"

// ServiceNameHeaderName identifies the originating service in server-to-server communication
// used in background jobs, cron, and internal microservices to clarify the source of the request
// maps to: service.name (otel resource attribute)
const ServiceNameHeaderName = "X-Service-Name"

// UserAgent returns a formatted user agent string
// e.g. "awee-cli/1.0.0 (darwin ?; amd64)" or
func UserAgent(app, version, os, osVersion, arch string) string {
	return fmt.Sprintf("%s/%s (%s %s; %s)", app, version, os, osVersion, arch)
}
