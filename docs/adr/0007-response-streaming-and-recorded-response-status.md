# Response streaming and a recorded response status

## Status

accepted

## Context and decision

`Call` had sinks for bodies it could hold in memory — `Text`, `HTML`, `JSON`, `Redirect` — and nothing
for a body it could not. An application serving a file, a proxied object or any large payload had to
write to `*call.Raw.W` itself, which silently broke the response lifecycle: `net/http` emitted the
header on the first write, the **buffered status** was never marked as flushed, and the
end-of-lifecycle **status flush** wrote it a second time. The wire result was correct and the log line
was not — `superfluous response.WriteHeader call` on every such request. The documented way to send a
large body was therefore "don't use `Call`".

Govalin had hit this itself, twice, in `static.go`, and both sites carried the same four-line comment
setting the unexported `bypassLifecycle`. That remedy was available to govalin and to nobody else: the
only escape hatch an application had was `HTTPServe`, which drops it to a bare `http.Handler` and gives
up path parameters, every `Call` helper and the framework's error conventions.

### The response records what it wrote, so nobody has to declare it

Every call now writes through a `responseWriter` that wraps the `http.ResponseWriter` and records
whether the response has been committed, and with which status. `sendStatusOrDefault` asks the writer
instead of a flag it maintains itself, which makes the flush idempotent against *any* write path — a
`Call` sink, `http.ServeContent`, a raw write, third-party middleware — rather than only against the
ones that remember to bypass. A **lifecycle bypass** is now about ownership (`HTTPServe`, a hijacked
websocket), not about avoiding a double write.

Two defects fall out of the same wrapper:

- The access log recorded `call.status`, the *buffered* value. Any handler writing its own header —
  including govalin's static serving — was logged with whatever `Status()` happened to hold, so a
  `206 Partial Content` was logged as `200`. It now logs the status recorded by the writer, falling
  back to the buffered one only when nothing reached the wire through govalin.
- A handler that wrote to the raw writer without setting a status was treated as not having handled
  the request, and the lifecycle appended a 404 JSON body to what it had already sent. A committed
  response is now enough to count as handled.

The wrapper forwards what handlers reach for through the writer: `Flush` for a streamed body,
`Hijack` because a websocket upgrade type-asserts on `http.Hijacker` directly, `ReadFrom` so
`io.Copy` keeps `net/http`'s sendfile path instead of falling back to a buffered copy, and `Unwrap`
for `http.ResponseController`. It deliberately does **not** swallow a second `WriteHeader`: a genuine
double write in application code should still be reported by `net/http`, and the framework's own
flush is guarded instead.

`http.MaxBytesReader` is the one caller that gets the *unwrapped* writer. It type-asserts against an
unexported `net/http` interface to close the connection on an over-long body, and an assertion does
not see through a wrapper.

### Sinks matched to what the content can do

- `Stream(contentType, reader)` — an `io.Reader` of unknown length. Chunked, no ranges, nothing
  buffered.
- `ServeContent(name, modTime, io.ReadSeeker)` — delegates to `http.ServeContent`, which owns Range,
  If-Range, 206, multipart ranges, 416, 304, HEAD and Content-Length. Reimplementing byte-range
  parsing to own the status would be the worse trade.
- `ServeContentAt(name, modTime, io.ReaderAt, size)` — the same, for content that reads at an offset
  but cannot seek. A remote object store exposes ranged GETs, not a seekable handle;
  `io.NewSectionReader` turns the size into the seekable view `http.ServeContent` wants, and a seek
  becomes offset arithmetic the next ranged read resolves. Without it, "serve a 4 GB object with
  working Range requests" means buffering it first, which is the case the feature exists for.
- `Download(filename, modTime, content)` — `ServeContent` with `Content-Disposition: attachment`,
  built by `mime.FormatMediaType` so a name with spaces or non-ASCII characters is encoded rather
  than pasted into a header.

`Stream` distinguishes the two failures `io.Copy` reports identically, by wrapping the source in a
reader that keeps its own error. A source that fails is the handler's to know about and is returned.
A failure on the way out is not: the header is long gone and the body is partly on the wire, so
there is nothing left to send and nothing for the handler to do. Returning it invites `call.Error` on
a dead connection, and logging it at `ERROR` makes a routine cancelled download look like a fault; it
is logged at debug level and reported as success. Classifying the error itself — matching `EPIPE`,
`ECONNRESET`, a cancelled request context — was tried first and rejected: the list is per-platform
guesswork, and the context check swallows a genuine source failure during a graceful shutdown, when
every request's context is cancelled.

For the same reason, a static file that fails part-way through its body is only logged. `call.Error`
would append a JSON error to a response that is already committed.

`static.go` now serves files through `ServeContent`, so both hand-rolled bypass sites and their
duplicated comment are gone — and static mounts answer Range requests, which they never did.
Directories still go to `http.FileServer`, which owns the trailing-slash canonicalization and the
directory index; it no longer needs a bypass either.

## Considered options

- **A status-recording writer (chosen)** — see above.
- **Exporting `bypassLifecycle`** — the smallest change: applications could do what `static.go` did.
  Rejected: it makes every caller responsible for remembering a framework invariant, fixes neither
  the access log nor the appended-404, and leaves third-party middleware unable to comply at all.
- **Swallowing a repeated `WriteHeader` in the wrapper** — would silence the warning everywhere,
  including for application bugs the warning exists to report. Rejected; the framework guards its own
  flush instead.
- **`Stream` only, no `ServeContent`** — smaller surface, but a large download could not be resumed
  and a client could not seek within a file, which is most of the reason to serve one.
- **Requiring `io.ReadSeeker` everywhere** — would force a remote object to be buffered to disk or
  memory before it could be served with ranges. Rejected in favour of `ServeContentAt`.
- **Returning the copy error on a client disconnect** — honest in isolation, but every caller then
  has to classify a disconnect itself to avoid answering a connection that is gone. Rejected.
- **Classifying the copy error by errno** — see above. Rejected in favour of recording which side of
  the copy failed, which is knowable rather than inferred.

## Consequences

- Sending a large body no longer means leaving `Call`, and no longer logs a superfluous
  `WriteHeader`.
- The access log reports the status actually sent. A deployment parsing access logs will see `206`,
  `301` and `416` where it previously saw the buffered value — usually `200`.
- **Behaviour change**: a static mount now answers `Range` requests with `206`, advertises
  `Accept-Ranges: bytes` and sets `Last-Modified` from the file. Clients that resume downloads will
  start doing so.
- **Behaviour change**: static routes no longer bypass the lifecycle, so `After` handlers registered
  on a path that also serves static files now run for those requests.
- **Behaviour change**: an explicit `index.html` request is served rather than redirected.
  `http.FileServer` canonicalised `/mount/index.html` to `/mount/` with a 301, so a client paid a
  second round trip for a file it had named exactly; serving through `Call` skips that.
- A handler writing to `*call.Raw.W` without setting a status is now treated as having handled the
  request, instead of having a 404 body appended to its response.
- `govalintest` gained `GetRange`, so a ranged response can be asserted on without hand-building a
  request.

## Measured cost

Benchmarks (`bench_test.go`, `bench_stream_test.go`) compared against the commit before this change,
10 runs each through `benchstat`, on an M4:

- Ordinary responses pay **one 16-byte allocation** for the recording writer and **+1% to +3%**
  latency (7–20 ns on a ~700 ns request): `Text` +2.9%, `JSON` +1.7%, `StatusOnly` +1.1%,
  `HTTPServe` +1.5%, before/after handlers +0.35%.
- A large body over a real connection shows **no measurable change** (`HTTPServe` with an 8 MB body:
  1.478 ms → 1.487 ms, p=0.24), because the wrapper forwards `ReadFrom` rather than downgrading the
  copy.
- Static files got **14% faster** with one allocation fewer, serving through `ServeContent` instead
  of `http.FileServer`.
- `ServeContent` moves 8 MB at 8.5 GB/s against 5.6 GB/s for a hand-rolled `io.Copy` to the raw
  writer — **50% faster with a quarter of the allocations** — so the sink that replaces the
  workaround is also the faster one.
