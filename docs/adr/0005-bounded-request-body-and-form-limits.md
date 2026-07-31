# Bounded request body and form limits

## Status

accepted

## Context and decision

`ServerMaxBodyReadSize` documents itself as "the max read size to accept from POST requests ... to
control DDoS attacks using big body sizes", but only `readBody` ever honoured it. `parseForm` called
`ParseForm`/`ParseMultipartForm` directly, so form POSTs silently fell back to net/http's own 10MB
allowance, and `ParseMultipartForm(0)` gave uploads a zero in-memory budget — every part was written
to a temp file, with the on-disk copy bounded by nothing at all.

Applying `maxBodyReadSize` to every body shape does not work: its 4096 default is sized for JSON, and
would reject every file upload. Multipart therefore gets its own limits, and the configured cap now
binds all three paths:

- **Body reads and URL-encoded forms** are bound by `ServerMaxBodyReadSize` (default 4096). These are
  the payload shapes that default sizing was written for.
- **Multipart bodies** are bound by `ServerMaxMultipartSize` (default 128 MiB) — a **total upload
  size** ceiling, replacing the previous absence of one.
- **`ServerMaxMultipartMemory`** (default 32 MiB, matching net/http's own `defaultMaxMemory`) is a
  distinct axis: how much of a multipart form is held in memory before parts spill to temporary files.
  It bounds memory pressure, not what the server will accept.

Limits are enforced by one helper, `limitBody`, shared by both paths. It rejects a body whose declared
`Content-Length` already exceeds the limit *before* reading a byte, and otherwise wraps the body in
`http.MaxBytesReader`. The declared-length pre-check is deliberate: tripping `MaxBytesReader` makes
net/http half-close the connection and linger 500ms to avoid RSTing the client, so a body that is
barely over would cost a teardown. Left unread instead, net/http drains a small overage and keeps the
connection alive, and abandons a large one.

Exceeding any limit is reported as **413**, replacing a 500 for oversized bodies and a generic 400
"Invalid form data" for oversized forms.

## Considered options

- **Separate multipart limits (chosen)** — see above.
- **One limit for every body shape** — simplest surface, but either 4096 breaks all uploads or an
  upload-sized default removes the small, safe JSON ceiling that the option exists to provide.
  Rejected.
- **A single multipart limit doubling as both size cap and memory budget** — fewer knobs, but the two
  bound different resources: accepting a 100MB upload streamed to disk is reasonable where holding
  100MB in memory is not. Rejected as conflating them.
- **Leaving multipart uncapped, memory budget only** — fixes the disk-spill complaint alone and leaves
  unbounded disk writes, which is the more serious of the two faults. Rejected.
- **Relying on `MaxBytesReader` alone, with no declared-length pre-check** — one code path, but every
  marginally-oversized request pays a connection teardown plus a 500ms linger. Rejected.

## Consequences

- **Breaking**: URL-encoded forms larger than `ServerMaxBodyReadSize` (default 4096) are now rejected;
  they previously got net/http's 10MB allowance. Applications posting large forms must raise the limit.
- **Breaking**: oversized bodies and forms return 413 instead of 500/400 respectively. Clients matching
  on the old statuses need updating.
- Multipart uploads are no longer forced to disk, so a typical upload within the memory budget avoids
  temp-file I/O entirely.
- The parse outcome is cached per call, so the repeated `parseForm` calls behind `FormParam`,
  `FormParams`, `File` and `Files` report one consistent result instead of the later ones observing a
  drained body as an empty form.
- Rejecting an oversized body without a declared `Content-Length` (chunked) still trips
  `MaxBytesReader`, so such a request holds its connection for net/http's 500ms linger. The default
  `ServerShutdownTimeout` of 200ms is shorter than that window, so a shutdown immediately after such a
  rejection will report a timeout.
