# User-supplied cache validators

## Status

accepted

## Context and decision

Govalin answered no conditional request of its own. `Call.ServeContent` inherits `If-None-Match`,
`If-Range` and `If-Modified-Since` handling from `http.ServeContent`, but only against an `ETag` the
handler had already set — and no handler ever did, because there was no way to. Static mounts fell
back to modification time, which an embedded FS does not have: every embedded asset reports the zero
time, so `Last-Modified` was omitted and an embedded bundle was uncacheable outright.

The obvious implementation, and the one every other framework's ETag plugin ships, is to hash the
response body: wrap the writer, buffer what the handler produces, hash it, compare, and either send
it or answer 304. **Govalin does not do this.** It inverts what the feature is for — the handler has
already run the query and marshalled the result before the framework discovers the client needed
none of it, so the only thing saved is transfer. It also fights the response model of ADR 0007, in
which everything writes straight through to the wire and `Stream` exists precisely so that a large
body is never held in memory, and it would put a buffer allocation on every request against the
budget ADR 0008 made a blocking gate.

So the **Validator** is the caller's, out of something it already knows: `call.NotModified(tag)`
sets the `ETag`, compares `If-None-Match` by the weak comparison RFC 9110 requires of that header,
and on a match buffers 304 and returns true. The return is permission to skip the work, not a
correctness obligation — the buffered 304 means a handler that ignores it and writes a body anyway
still sends a correct empty 304, because `net/http` refuses a body the status does not allow. This
is the same instinct as the **Committed response**: the framework records what happened rather than
trusting a caller to declare it.

Static assets get a **Derived validator** instead, on by default and with no flag, since a validator
is advisory and the client decides whether to act on it. A disk mount uses modification time and
size, which `fs.Stat` has already read. An embedded mount has no usable time, so its files are
hashed (FNV-64a — not a security boundary) and the result cached per mount, for the process's
lifetime. The rule states in one line: *a file that knows when it changed is validated by when it
changed; one that does not is validated by what is in it.* Both forms are strong, because
`net/http` requires a strong match for `If-Range` and a weak tag would silently cost every resumed
download its ranges.

## Considered options

- **Caller-supplied validators, derived only for static (chosen)** — see above.
- **Hashing the response body in core** — the familiar implementation, rejected above. It remains
  available as a plugin to anyone who wants it, which is where the **Plugin complexity boundary**
  puts a cost every user would otherwise pay.
- **Size-only tags for embedded assets** — free, and wrong in the case embedded assets are for: a
  same-length edit across a redeploy (`1.0.9` → `1.1.0`) would keep every client on the old copy
  indefinitely.
- **Hashing every static file on every request** — uniform, and a couple of milliseconds of CPU on
  the hottest asset. The cache is what makes hashing viable, not an optimisation on top of it.
- **A registration API (`app.ETag("/api/*", ...)`)** — a second way to say what a `Before` handler
  already says, with its own path matching and lifecycle position, to save the user one line.
- **A `StaticConfig` flag to disable derived validators** — configuration whose only purpose is
  turning off correct behaviour that costs one hash per file. Addable later if someone turns up with
  a reason.
- **Leaving directories entirely to `http.FileServer`** — where they were. A directory holding an
  `index.html` is serving a file, and delegating it meant the mount root, the most requested URL a
  static site has, was the one file with no validator. It now goes through `Call` like any other.
  What stays delegated is what genuinely belongs there: a generated listing, which has no validator
  to derive short of hashing the body, and a directory reached without its trailing slash, which the
  file server redirects.

## Consequences

- Existing static mounts start answering 304 to clients that revalidate. That is the bug fix, but it
  is a behaviour change to responses users are already serving.
- The static allocation budget rises to cover the derived tag and the header write.
- The hash cache assumes a file system reporting no modification time cannot change beneath it. That
  holds for `embed.FS`; a custom mutable `fs.FS` that also reports zero times would be served a stale
  validator, and is documented as such.
- `If-Match` and 412 are not implemented. Optimistic concurrency remains an open feature, and one
  that must not be folded into `NotModified` when it arrives.
- Freshness — `Cache-Control`, `Expires` — is a deliberate follow-up. A validator makes a
  revalidation cheap; only freshness makes it unnecessary. Until then a client still pays a round
  trip per asset to be told 304, which on a fast connection is the larger cost of the two.
