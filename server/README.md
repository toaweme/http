# http/server

[![Quality](https://github.com/toaweme/http/actions/workflows/tests.yml/badge.svg)](https://github.com/toaweme/http/actions/workflows/tests.yml)
[![Go Reference](https://img.shields.io/badge/Docs-pkg.go.dev-blue)](https://pkg.go.dev/github.com/toaweme/http/server)
[![License](https://img.shields.io/badge/License-MIT-blue)](../LICENSE)

## A small HTTP server, batteries where you want them

`github.com/toaweme/http/server` wraps `net/http.Server` behind a chi-backed `Router` and a `{Name, Start, Stop}` lifecycle, and bundles the middleware, params, JSON helpers, and Server-Sent Events you reach for on every service. chi stays an implementation detail - handlers never import it. It is the server half of [`github.com/toaweme/http`](https://github.com/toaweme/http) and depends only on `go-chi/chi`.

## Install

```sh
go get github.com/toaweme/http/server
```

## Module

- `server.NewRouter()` builds a [Router]; register handlers with `Get`/`Post`/`Put`/`Delete`/`Patch` (or `Handle`), nest with `Group`, and add middleware with `Use`/`With`.
- `server.NewServer(Config, *Router, Logger, ...Option)` wraps the router in a `*net/http.Server`; `Start` blocks until `Stop(ctx)` shuts it down gracefully.
- `server.Param`, `server.Wildcard`, `server.RoutePattern` read path data without exposing chi to handlers.
- `server.SlogMiddleware(SlogConfig, Logger)` logs every request; `server.AuthMiddleware(ClaimsExtractor, Logger)` enforces Bearer auth and injects claims.
- `server.WriteJSON` / `WriteError` / `WriteBadRequest` / `ReadJSON` / `ReadRawJSON` are the request/response helpers.
- `sse.NewHub()` (sub-package `server/sse`) broadcasts Server-Sent Events to subscribers.

## Overview

### Routing

`Router` is chi underneath, with method helpers, groups, and scoped or inline middleware. Handlers use plain `net/http` signatures.

```go
r := server.NewRouter()
r.Use(server.SlogMiddleware(server.SlogConfig{}, logger)) // root middleware

r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
	server.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
})

r.Group("/api", func(api *server.Router) {
	api.Use(server.AuthMiddleware(extractClaims, logger)) // scoped to /api
	api.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request) {
		server.WriteJSON(w, http.StatusOK, map[string]string{"id": server.Param(req, "id")})
	})
})

r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
	_ = server.Wildcard(req) // everything after /files/
})
```

`Use` panics if called after a route is registered on the same scope (a chi guard), so add middleware first. `With(mw...)` returns a sub-router for one-off inline middleware. `LogRoutes(logger)` walks and logs every registered route.

### Server lifecycle

`NewServer` builds the underlying `*http.Server` eagerly with a sane `ReadHeaderTimeout` default (Slowloris protection). `Start` serves and blocks; `Stop` shuts down gracefully within the context deadline. The type implements the `{Name, Start, Stop}` service contract.

```go
srv := server.NewServer(server.Config{Host: "127.0.0.1", Port: 8080}, r, logger)

go func() {
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}()

// ... on shutdown signal:
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = srv.Stop(ctx)
```

### Configuring the underlying server

`Config` only holds the listen address. Everything else is set with functional options, or by mutating the raw `*http.Server`:

```go
srv := server.NewServer(cfg, r, logger,
	server.WithReadHeaderTimeout(5*time.Second),
	server.WithReadTimeout(10*time.Second),
	server.WithWriteTimeout(10*time.Second),
	server.WithIdleTimeout(120*time.Second),
)

srv.HTTP().TLSConfig = tlsCfg // escape hatch for anything no Option covers
```

Options run after the defaults, so they override; `WithReadHeaderTimeout(0)` disables the timeout. Mutate `HTTP()` before calling `Start`.

### Auth middleware

`AuthMiddleware` pulls the Bearer token, runs your `ClaimsExtractor` to parse it, and injects org/user/scopes into the request context (or aborts with 401). Read them back in handlers:

```go
extract := func(token string) (*server.Claims, error) {
	return &server.Claims{OrgID: "org-1", UserID: "user-1", Scopes: []string{"read"}}, nil
}
r.Use(server.AuthMiddleware(extract, logger))

// inside a handler:
org, _ := server.OrgIDFromContext(req.Context())
user, _ := server.UserIDFromContext(req.Context())
scopes := server.ScopesFromContext(req.Context())
```

The `Authorizer` interface and `ContextWith*` helpers are available for building authorization checks on top.

### Request logging

`SlogMiddleware` logs method, url, duration, and status for every request. Opt in to capturing headers and bodies (with a size cap) for local debugging:

```go
r.Use(server.SlogMiddleware(server.SlogConfig{
	LogRequestBody:  true,
	LogResponseBody: true,
	MaxBodyBytes:    4096, // 0 means no cap
}, logger))
```

### JSON helpers

```go
var in CreateItem
if err := server.ReadJSON(req, &in); err != nil {
	server.WriteBadRequest(w, err)
	return
}
server.WriteJSON(w, http.StatusCreated, Item{ID: "1"})
server.WriteError(w, http.StatusNotFound, errors.New("not found"))
```

### Server-Sent Events

The `server/sse` sub-package provides an SSE `Writer` and a topic-scoped `Hub`. `ServeStream` ties a hub topic to a handler, with heartbeats and slow-subscriber handling:

```go
hub := sse.NewHub()

r.Get("/stream", func(w http.ResponseWriter, req *http.Request) {
	_ = sse.ServeStream(w, req, hub, "updates") // blocks until the client disconnects
})

// from anywhere:
hub.Publish("updates", sse.Event{Type: "tick", Data: "hello"})
```

Subscribers that fall behind have their channel closed rather than blocking the producer; `Subscribers(topic)` reports the current count. For one-off writing, `sse.NewWriter(w)` returns a `*Writer` with `Start`, `Write(Event)`, and `Ping`.

## Features

- **chi-backed router** - `Get`/`Post`/`Put`/`Delete`/`Patch`/`Handle`, `Group` nesting, `Use`/`With` middleware, and `LogRoutes`, without leaking chi into handlers.
- **Server lifecycle** - `Name`/`Start`/`Stop` over `net/http.Server` with graceful shutdown.
- **Configurable transport** - `WithReadHeaderTimeout`/`WithReadTimeout`/`WithWriteTimeout`/`WithIdleTimeout` options plus a `HTTP()` escape hatch; secure `ReadHeaderTimeout` by default.
- **Param access** - `Param`, `Wildcard`, `RoutePattern`.
- **Auth middleware** - Bearer-token extraction into request context via a pluggable `ClaimsExtractor`; `Claims`, `Authorizer`, and `*FromContext` / `ContextWith*` helpers.
- **Request logging** - structured method/url/duration/status, with optional headers and size-capped bodies.
- **JSON helpers** - `WriteJSON`, `WriteError`, `WriteBadRequest`, `ReadJSON`, `ReadRawJSON`.
- **SSE hub** - `Writer` for well-formed events, `Hub` for topic fan-out with slow-subscriber drop, and `ServeStream` with heartbeats.
- **Injectable logger** - a minimal `Logger` interface defined locally so the server module never depends on the client; satisfied structurally by `github.com/toaweme/log`.

## Runnable examples

The package tests double as usage references: [`router_test.go`](./router_test.go), [`server_test.go`](./server_test.go), [`auth_middleware_test.go`](./auth_middleware_test.go), and [`sse/sse_test.go`](./sse/sse_test.go).

```sh
go test ./...
```
