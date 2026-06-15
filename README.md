# http

[![Quality](https://github.com/toaweme/http/actions/workflows/tests.yml/badge.svg)](https://github.com/toaweme/http/actions/workflows/tests.yml)
[![Go Reference](https://img.shields.io/badge/Docs-pkg.go.dev-blue)](https://pkg.go.dev/github.com/toaweme/http)
[![GitHub Tag](https://img.shields.io/github/v/tag/toaweme/http?label=Tag&color=green)](https://github.com/toaweme/http/releases)
[![License](https://img.shields.io/badge/License-MIT-blue)](/LICENSE)

## HTTP client and server

Zero dependency, lightweight HTTP client with a struct based config.

A tiny wrapper around `chi` so that module consumers wouldn't need to import chi directly.
It's unlikely to ever need to switch routers, but years ago I thought the same thing about Gin and now migrating to chi.

## Modules

Each with their own `go.mod`, but both here for convenience.

- `github.com/toaweme/http` - the HTTP **client**. Pure stdlib, no third-party dependencies. Go 1.19+.
- `github.com/toaweme/http/server` - the HTTP **server**: router, middleware, auth, JSON helpers. Depends only on `go-chi/chi`.
  - `github.com/toaweme/http/server/sse` - the **Server-Sent Events** writer and broadcast hub used by the server.

## Install

```sh
go get github.com/toaweme/http          # client
go get github.com/toaweme/http/server   # server
```

> The package is named `http`, so import it as plain `http`. It only collides with stdlib `net/http` in a file that imports both - alias it (`thttp`) only there.

## The client

### A request is a struct

Build a `Client` from a `Config`, then call one method per verb. Every call takes a context and a request struct; you get back a `*Response` with the status, body, and headers.

```go
client := http.NewClient(http.Config{
	BaseURL:   "https://api.example.com",
	UserAgent: "demo/1.0.0",
	Platform:  "cli",
})

resp, err := client.Get(ctx, http.GetRequest{
	Request: http.Request{Path: "/users/1"},
})
if err != nil {
	return err
}
fmt.Println(resp.StatusCode, string(resp.Body))
```

`Request` carries the per-request knobs - `Path`, `Query` (`url.Values`), `Headers`, plus `ID` and `SessionID` which are emitted as `X-Request-ID` / `X-Session-ID`. `GetRequest` and the body-carrying `PostRequest` / `PutRequest` / `PatchRequest` embed it:

```go
body, _ := http.JSON(map[string]string{"name": "ada"})
resp, err := client.Post(ctx, http.PostRequest{
	Request: http.Request{Path: "/users", ID: "req-123"},
	Body:    []byte(body),
})

user, err := http.FromJSON[User](resp.Body) // typed decode helper
```

### Streaming (Server-Sent Events)

`GetStream` / `PostStream` open an SSE connection and decode the wire format into typed `StreamResponse` values on a channel you own. The call returns once the reader goroutine is running; the channel is closed on EOF.

```go
stream := make(chan http.StreamResponse)
if err := client.GetStream(ctx, stream, http.Request{Path: "/events"}); err != nil {
	return err
}
for ev := range stream {
	switch ev.Type {
	case http.StreamResponseTypeData:
		fmt.Println("data:", string(ev.Body))
	case http.StreamResponseTypeEvent, http.StreamResponseTypeID, http.StreamResponseTypeRetry:
		// event:/id:/retry: lines, value in ev.Body
	case http.StreamResponseTypeEOF:
		if ev.Error != nil {
			return ev.Error
		}
	}
}
```

### Config, headers, and identity

`Config` seeds the client-wide headers used for tracing and client identification, each mapped to a documented header constant (`User-Agent`, `X-Client-Platform`, `X-Client-Version`, `X-Client-ID`, `X-Service-Name`). Anything in `Config.Headers` is sent on every request; per-request `Headers` override them. The `UserAgent(app, version, os, osVersion, arch)` helper formats a conventional UA string.

### Bring your own `*http.Client` and logger

Construction options stay out of your way by default - `http.DefaultClient` and a silent logger:

```go
client := http.NewClient(cfg,
	http.WithHTTPClient(&http.Client{Timeout: 5 * time.Second}), // custom timeout/transport, or a test stub
	http.WithLogger(logger),                                     // any leveled logger
)
```

`Logger` is a minimal `Trace/Debug/Info/Warn/Error` interface, satisfied structurally by `github.com/toaweme/log` with no adapter.

## The server

The `server` module wraps `net/http.Server` behind a chi-backed `Router` and a `{Name, Start, Stop}` lifecycle, and keeps chi out of your handlers.

```go
import (
	"net/http"

	"github.com/toaweme/http/server"
)

r := server.NewRouter()
r.Use(server.SlogMiddleware(server.SlogConfig{}, logger)) // request logging
r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
	server.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
})
r.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
	server.WriteJSON(w, http.StatusOK, map[string]string{"id": server.Param(req, "id")})
})

srv := server.NewServer(server.Config{Host: "127.0.0.1", Port: 8080}, r, logger)
if err := srv.Start(); err != nil { // blocks; Stop(ctx) for graceful shutdown
	log.Fatal(err)
}
```

Tune the underlying server with options (`WithReadHeaderTimeout`, `WithReadTimeout`, `WithWriteTimeout`, `WithIdleTimeout`) or reach the raw `*http.Server` via `srv.HTTP()` for anything they do not cover (TLS, connection hooks). Bearer-token auth is one middleware away:

```go
r.Use(server.AuthMiddleware(extractClaims, logger))
// inside a handler:
org, _ := server.OrgIDFromContext(req.Context())
```

Broadcast SSE with the hub:

```go
hub := sse.NewHub()
r.Get("/stream", func(w http.ResponseWriter, req *http.Request) {
	_ = sse.ServeStream(w, req, hub, "updates")
})
// elsewhere:
hub.Publish("updates", sse.Event{Type: "tick", Data: "hello"})
```

## Features

**Client (`github.com/toaweme/http`)**

- **Zero dependencies** - pure stdlib `net/http`, nothing transitive.
- **Struct requests, one method per verb** - `Get`, `Post`, `Put`, `Patch`, `Delete` returning `*Response` (status, body, headers).
- **SSE streaming** - `GetStream` / `PostStream` decode `data:`/`event:`/`id:`/`retry:`/comment lines into typed `StreamResponse` values, with explicit EOF and errors.
- **Config-driven identity** - base URL, user-agent, platform, app version, client/service IDs, and custom headers, each behind a documented header constant.
- **Per-request overrides** - path, query, headers, request ID, session ID.
- **Swappable transport** - `WithHTTPClient` for custom timeouts/transports or a stub in tests; `http.DefaultClient` by default.
- **Injectable logger** - leveled `Logger` interface, silent by default, satisfied structurally by `github.com/toaweme/log`.
- **JSON helpers** - `JSON(v)` and generic `FromJSON[T](body)`.

**Server (`github.com/toaweme/http/server`)**

- **chi-backed router** - method helpers, groups, scoped/inline middleware, and route logging, without exposing chi to handlers.
- **Server lifecycle** - `Name`/`Start`/`Stop` over `net/http.Server` with graceful shutdown and functional options (timeouts) plus a `HTTP()` escape hatch.
- **Param access** - `Param`, `Wildcard`, `RoutePattern`.
- **Auth middleware** - Bearer-token extraction into request context (org/user/scopes) via a pluggable `ClaimsExtractor`.
- **Request logging** - structured method/url/duration/status, with optional headers and size-capped bodies.
- **JSON helpers** - `WriteJSON`, `WriteError`, `WriteBadRequest`, `ReadJSON`, `ReadRawJSON`.
- **Local logger interface** - defined in the module so the server never depends on the client.

**SSE hub (`github.com/toaweme/http/server/sse`)**

- **Writer** - emits well-formed SSE events (id/event/multi-line data) and flushes.
- **Hub** - topic fan-out to subscribers; slow subscribers are dropped rather than blocking producers; `ServeStream` ties it to a handler with heartbeats.

## Runnable examples

The test files double as usage references: [`client_test.go`](./client_test.go) for the client, and the `server/` package tests ([`router_test.go`](./server/router_test.go), [`server_test.go`](./server/server_test.go), [`sse/sse_test.go`](./server/sse/sse_test.go)) for routing, lifecycle, and streaming.

```sh
go test ./...
```
