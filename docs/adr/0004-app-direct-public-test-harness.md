# App-direct public test harness

## Status

accepted

## Context and decision

Users of govalin have no supported way to test their applications: the HTTP test harness lives in
`internal/govalintesting` and is invisible to consumers. It was also designed strictly for testing
govalin itself — every failure calls `os.Exit(1)` (killing the whole test binary instead of failing
one test), its signatures expose the dormant `ddliu/go-httpclient` library, it builds the app under
test itself (so it cannot exercise a user's own constructor), and it pre-allocates a free port with
a TOCTOU race plus a 50ms sleep for readiness.

We ship a purpose-designed public package, `govalintest`, rather than exporting the internal helper:

- **App-direct**: `govalintest.Test(t, app, fn)` takes the user's already-constructed **app-under-test**
  — built by their production constructor with their config, plugins, and routes. The harness starts
  it, hands the test a client, and shuts it down via `t.Cleanup`. It never builds or reconfigures the
  app (log verbosity included — that stays the app owner's choice).
- **Test-scoped failure**: the client holds the `*testing.T`; transport errors fail the calling test
  with `t.Fatalf` (`t.Errorf` from non-test goroutines), never the process.
- **Stdlib surface**: response access is `*http.Response`; escape hatches are `Do(*http.Request)` and
  `HTTP() *http.Client`. The one third-party type kept is `*gorilla/websocket.Conn` from
  `Websocket(path)`, because core's `App.Ws()` already makes gorilla a dependency every consumer ships.
- **Status-agnostic body access**: the string-returning verbs return the body regardless of status
  code; tests that exercise error handlers depend on this, and status assertions stay explicit via the
  `*Response` variants.
- **Startup-event readiness**: a new core method, `App.Events(...)` (the post-construction counterpart
  of `Config.Events`), lets the harness register an `OnServerStartup` callback. The callback fires in
  the `Start` goroutine after the bound port is written, so signalling over a channel gives the harness
  a race-clean happens-before edge to `App.Port()`. The server starts on port 0; `freePort()` and the
  readiness sleep are deleted.
- **One body convention**: `Post`/`Put`/`Patch` take `body any` — `string`/`[]byte`/`io.Reader` sent
  as-is, anything else JSON-encoded — replacing the three inherited ddliu conventions. Bodyless verbs
  take only a path; anything exotic uses `Do`.
- **Options struct, not option funcs**: `TestWithOptions(t, app, opts, fn)` keeps the test closure
  last at the call site; the zero value of `Options` always means the defaults (startup timeout 5s).

Govalin's own tests migrate to `govalintest` and `internal/govalintesting`'s HTTP harness is deleted,
so the public API is proven sufficient by dogfooding. Delivery is a two-PR rollout: PR1 adds
`App.Events`, the package, and README testing docs; PR2 is the mechanical test migration.

## Considered options

- **Purpose-designed public package, app-direct (chosen)** — see above.
- **Export the internal helper as-is** — fastest, but freezes `os.Exit` failure semantics, a dormant
  third-party client, and the setup-callback shape into the public API. Rejected: a public test util
  that can kill the consumer's test binary is hostile, and pre-1.0 freedom is the cheapest it will
  ever be to do this properly.
- **Setup-callback harness** (harness builds the app, user configures it in a callback) — matches the
  internal design and lets the harness silence logs, but cannot exercise the user's real application
  constructor, forcing users to duplicate their wiring in tests. Rejected: the goal is testing *their*
  application.
- **Harness-only readiness (poll/pre-allocated port, no core change)** — avoids touching core, but
  polling `App.Port()` while `Start` writes it is a data race (forbidden by the race-clean stability
  gate), and pre-allocating a port resurrects the TOCTOU flake. Rejected.
- **Generic typed responses** (`JSON[T]` package funcs, or generic methods) — Go has no generic
  methods, so the ergonomic form is impossible today and the package-func workaround would freeze an
  awkward shape into the API. Deferred, not rejected: revisit if generic methods land.

## Consequences

- `github.com/pkkummermo/govalin/govalintest` becomes public API with the usual compatibility
  expectations; `Options`'s zero value must remain "sane defaults" forever.
- Core gains `App.Events(...)`, independently useful beyond testing (e.g. adding shutdown hooks to an
  app built elsewhere). It is valid before `Start`.
- The **plugin complexity boundary** is clarified rather than violated: a test package in the core
  module adds no dependency cost to consumers who never import it, and `govalintest` needs no
  dependency core does not already have.
- Migrated internal tests need small mechanical edits where they relied on ddliu form-map magic or
  `Get`-with-params passthrough (query strings go in the path).
- `internal/govalintesting/exit.go` stays internal; only the HTTP harness is replaced.
