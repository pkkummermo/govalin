# End-of-lifecycle status flush

## Status

accepted

## Context and decision

`Status()` sets a **buffered status** on the call; it is committed to the writer only by a
**status flush**. Until this decision the only flush sites were the body-writing methods
(`JSON`/`Text`/`HTML`/`Redirect`) via `sendStatusOrDefault()`. A handler that wanted to reply
"status only, no body" — e.g. `call.Status(204); return` — therefore never flushed: `net/http`
finished the request with an implicit `200 OK`, discarding the intended status. The same silent-200
bug applied to a `Before` handler that short-circuited with a status but no body (`call.Status(401);
return false`).

We make the buffered status flush at the end of the request lifecycle on every non-bypass exit path.
A deferred closure in `rootHandlerFunc` calls `sendStatusOrDefault()` (idempotent — guarded by
`statusWritten`), so a status set by a handler or a short-circuiting before handler is always
committed. After this change, setting a status is sufficient to send it; no new public API is needed.

The bypass guard is evaluated when the defer **runs**, not when it is registered. `bypassLifecycle`
is set to `true` inside the handler (`HTTPServe`), which executes after the defer is registered, so a
guard snapshotted at registration time would wrongly flush over a raw response the handler already
wrote. The closure reads `call.bypassLifecycle` at run time and skips the flush for **lifecycle
bypass** requests, leaving response finalization entirely to the handler.

## Considered options

- **End-of-lifecycle auto-flush (chosen)** — buffered status is committed once at the end of every
  non-bypass path. Makes `Status()` honest, kills the silent-200 class for both no-body handlers and
  before-handler short-circuits, and adds no public surface.
- **Explicit `Send()`/`SendStatus()` method** — handler must call it to flush a no-body status.
  Rejected: a status set without the follow-up call still silently becomes 200, preserving the exact
  trap and adding API the framework can forget to require.
- **Flush only on the normal-completion path** — explicit flush at the end of `rootHandlerFunc`.
  Rejected: leaves the before-handler short-circuit path emitting a silent 200 for status-only
  rejects.

## Consequences

- A request that sets a status and writes no body now sends that status; `Status()`'s doc contract
  is updated to say the status is flushed on a body write **or** at end of lifecycle.
- `HTTPServe` (lifecycle bypass) is untouched: the run-time guard means govalin never writes a status
  for a request whose handler owns the raw writer.
- The defer registers after the access-log defer, so LIFO ordering flushes the status before the
  access log records it — the log reports the status actually sent.
