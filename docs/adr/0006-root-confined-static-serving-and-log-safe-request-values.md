# Root-confined static serving and log-safe request values

## Status

accepted

## Context and decision

Three open CodeQL alerts pointed at the two places where a request decides what the framework
touches: the static file handler (`go/path-injection`) and the access log (`go/clear-text-logging`,
twice). They are treated as one class — *request-controlled values reaching a resource the request
should not get to name* — and fixed together.

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

The access log recorded the request path and `User-Agent` verbatim, and the call ID verbatim from the
inbound `X-Govalin-Id` header. All three are chosen by the client, and all three are unbounded: a
request could decide how many bytes one log line costs, and — depending on the sink — smuggle control
characters that break a record in two or carry terminal escape sequences to whoever reads the log.

`logSafeValue` drops control characters and bounds the value at 256 characters, marking a truncated
value with `…` so it is never mistaken for what the client actually sent. The bound counts characters
rather than bytes, so multi-byte text is not cut mid-rune.

The correlation ID gets a stricter rule, because it is an *identity* and not just a logged value: an
inbound `X-Govalin-Id` is adopted only when it is a bounded (≤128), printable, whitespace-free token.
Anything else is replaced by a generated UUID rather than rejected, so a malformed header never fails
an otherwise valid request.

## Considered options

- **`os.Root` confinement (chosen)** — see above.
- **Keeping the prefix check and adding `filepath.EvalSymlinks`** — closes the symlink hole with no
  new API, but re-introduces the same time-of-check/time-of-use gap the check already had: the path
  is resolved, then opened separately. Rejected.
- **Relying on `http.Dir`/`http.FileServer` alone** — the file server does clean the request path, so
  it was never the leak; the `os.Stat` ahead of it was, as an existence oracle for files outside the
  root. Dropping the stat would lose SPA-mode fallback, which depends on knowing whether the file
  exists. Rejected.
- **Rejecting a hostile `X-Govalin-Id` with 400** — surfaces the bad header loudly, but makes a header
  that most clients never set into a way to fail a valid request. Rejected in favour of falling back
  to a generated ID.
- **Dropping user agent and path from the access log** — removes the tainted values outright, and with
  them the reason anyone reads an access log. Rejected.
- **Leaving sanitization to the log sink** — correct for a JSON handler, which escapes control
  characters, but govalin does not choose its own sink: `slog.Default()` may be a text handler writing
  to a terminal. Rejected as an assumption the framework cannot make.

## Consequences

- **Breaking**: a static mount whose directory is missing or unreadable now answers 500 instead of 404.
- **Breaking**: an `X-Govalin-Id` longer than 128 characters, or containing whitespace, control
  characters or non-ASCII bytes, is no longer echoed as the call ID; a generated UUID is used instead.
  Callers propagating a correlation ID need one that fits the rule — a UUID does.
- Static files reachable only through a symlink out of the static directory are no longer served. A
  deployment that relied on symlinking content in from elsewhere must move or bind-mount it instead.
- Access log values are capped at 256 characters, so a very long path or user agent appears truncated
  with a trailing `…`.
- On-disk and embedded static mounts now run the same code path, so behaviour that used to differ
  between them (status set before the file server ran) is the same for both.
