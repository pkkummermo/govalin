# Root-confined static serving and log-safe request values

## Status

accepted

## Context and decision

Three open CodeQL alerts pointed at the two places where a request decides what the framework
touches: the static file handler (`go/path-injection`) and the access log (`go/clear-text-logging`,
twice). Two describe a real weakness and are fixed here; the third does not, and is recorded below as
a false positive rather than designed around.

### Static serving is confined by the OS, not by string comparison

`handle` resolved the request path against the static directory and then compared the cleaned
absolute result against the cleaned absolute root. That rejects `../` traversal, but path arithmetic
only describes names, not what they resolve to: a symlink inside the static directory is contained as
a *path* while pointing anywhere on the filesystem, so it passed the check and was then stat'd and
served.

Static serving now opens the configured directory as an `os.Root` and does every read through it. The
kernel refuses any name that leaves the root, including through a symlink, and the containment no
longer depends on getting string comparison right on every platform. The two branches also collapse:
an `os.Root.FS()` is an `fs.FS`, so the on-disk and embedded cases share one stat and one file server
instead of each having their own.

Two behaviours follow from that and are deliberate:

- A name that fails to resolve is answered **404 directly**, rather than being handed on to
  `http.FileServer`. A name that escapes the root is not a *missing* file to the file server, which
  would answer 500 and thereby confirm that something is there.
- A mount whose directory cannot be opened is a **500** with an actionable log line, not a 404 for
  every request. A static mount pointing at a directory that is not there is a misconfiguration, and
  reporting it as "no such file" hides it.

### Values a client controls are bounded and scrubbed before they are logged

The access log recorded the request path and `User-Agent` verbatim. Both are chosen by the client and
neither is capped by anything else, so a request decided how many bytes its log line cost — up to
`MaxHeaderBytes` (1MB by default) for the user agent, and as much again for the path.

`logSafeValue` bounds the value at 256 characters, marking a truncated value with `…` so it is never
mistaken for what the client actually sent. The bound counts characters rather than bytes, so
multi-byte text is not cut mid-rune.

It also drops control characters, but as defence in depth rather than as the fix for a live hole.
Measured against the real stack:

- Control bytes in a *header* value never arrive: net/http answers **400** to a request carrying
  `ESC`, `NUL` or `DEL` in a header, before any handler or log call runs. Only tab, and `\r\n `
  continuation folded to a space, get through.
- Control bytes in a *path* do arrive, because percent-encoding survives the request-line check and
  `url.Parse` decodes it — `/x%00`, `/x%1b[31m` and `/x%0a` all reach the handler.
- `slog`'s own text and JSON handlers then escape them: a newline is written `\n`, an escape byte
  `\x1b` (text) or `\u001b` (JSON). Neither can break a record or reach a terminal raw.

So scrubbing guards only the case of a handler that does not escape. It is kept because it is nearly
free and govalin does not choose the handler it logs through, not because the standard ones leak.

### The logged call ID is not a finding

The third alert flags the call ID on the same log call, because it originates in the inbound
`X-Govalin-Id` header. That is the feature, not a leak: the header exists so a caller can propagate
its own correlation ID, and appearing in every record for the request is the entire point of having
one. Nothing sensitive is disclosed — the client reads back an identifier it chose itself.

Nor is there a payload to smuggle through it. A header value carrying `ESC`, `NUL` or `DEL` is
answered 400 by net/http before govalin sees it; what does get through (tab, folded continuation
lines) is escaped by `slog`. The only thing an inbound ID can still do is be long, which is a
log-volume question and not what the alert is about.

The ID is therefore logged as sent, and the alert is dismissed as a false positive rather than
answered with code.

## Considered options

- **`os.Root` confinement (chosen)** — see above.
- **Keeping the prefix check and adding `filepath.EvalSymlinks`** — closes the symlink hole with no
  new API, but re-introduces the same time-of-check/time-of-use gap the check already had: the path
  is resolved, then opened separately. Rejected.
- **Relying on `http.Dir`/`http.FileServer` alone** — the file server does clean the request path, so
  it was never the leak; the `os.Stat` ahead of it was, as an existence oracle for files outside the
  root. Dropping the stat would lose SPA-mode fallback, which depends on knowing whether the file
  exists. Rejected.
- **Constraining the inbound `X-Govalin-Id` to a bounded printable token, falling back to a generated
  UUID** — would silence the third alert, at the cost of the propagation the header exists for: a
  caller's own correlation ID no longer survives if it does not match govalin's idea of a token.
  Rejected; the alert is the thing that is wrong, not the behaviour.
- **Dropping user agent and path from the access log** — removes the tainted values outright, and with
  them the reason anyone reads an access log. Rejected.
- **Leaving control characters to the log sink** — accepted, in effect: both standard `slog` handlers
  escape them, so the scrub in `logSafeValue` is redundant with them and is kept only for a custom
  handler that does not. It is the *length* bound that no sink provides.

## Consequences

- **Breaking**: a static mount whose directory is missing or unreadable now answers 500 instead of 404.
- An inbound `X-Govalin-Id` is still adopted verbatim as the call ID and still logged verbatim, so a
  client can put an arbitrarily long value in every log record for its request. That is accepted:
  bounding it would break correlation-ID propagation, which is what the header is for.
- Static files reachable only through a symlink out of the static directory are no longer served. A
  deployment that relied on symlinking content in from elsewhere must move or bind-mount it instead.
- Access log values are capped at 256 characters, so a very long path or user agent appears truncated
  with a trailing `…`.
- On-disk and embedded static mounts now run the same code path, so behaviour that used to differ
  between them (status set before the file server ran) is the same for both.
