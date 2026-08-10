# Opt-in freshness headers

## Status

accepted

## Context and decision

ADR 0009 made a revalidation cheap. It did not make it unnecessary: the client still asks. Without a
freshness header a browser has to guess how long a response stays current, and the heuristic it
guesses with is a fraction of the age since `Last-Modified` — which an embedded asset does not have,
because `embed.FS` reports the zero time and `http.ServeContent` omits the header. So an embedded
bundle is revalidated on **every** page load: twenty assets, twenty round trips, twenty 304s. On a
fast connection that latency is the larger of the two costs, and a validator does not touch it.

**Freshness** is the other half. `call.CacheFor(10 * time.Minute)` writes `Cache-Control: max-age=600`
and the client reuses its own copy without asking at all. `CachePrivateFor` adds `private` for a
response behind a login, and `NoStore` refuses storage outright — a **Cache scope** the caller states
rather than one read off the lifetime. All three write the one header with `Set`, so
the last one called is the one that answers and a default declared for a group in a `Before` handler
is overridable by the single route that must not be stored.

A **SPA mount's shell is the exception**, and the one place govalin declares freshness without being
asked: `index.html` is served `no-cache` whatever the mount says, at the mount root, at its own URL
and at every client-side route the fallback answers. A lifetime on the shell is not the usual trade
of staleness against round trips — the shell names the fingerprinted bundles, so a client reusing an
old one asks for assets the deploy replaced, and a mount whose fallback serves it everywhere makes a
stale copy the whole application. `no-cache` rather than `no-store` because the **Derived validator**
is already there: the client keeps its copy and is told to keep it, so being never stale costs a 304.
Outside SPA mode there is no shell to make an exception of, so `index.html` is a file like any other.

Static mounts otherwise declare **nothing by default**, which is the opposite of the **Derived validator** and
deliberately so. A validator is advisory: a client that ignores it loses a round trip. A lifetime is
binding: the asset is served stale for its full duration out of a cache the server cannot reach, and
no deploy takes that back. Nothing govalin knows about a directory says how long its files stay
current, so the mount says it in one line — `config.CacheFor(time.Hour)` — covering every file it
serves.

## Considered options

- **`Cache-Control` alone, and no default lifetime for static (chosen)** — see above.
- **Sending `Expires` alongside `max-age`** — the header this feature was filed under, and dead
  weight. RFC 9111 §5.3 has a recipient ignore `Expires` whenever `max-age` is present, so it is read
  only by a cache that speaks no HTTP/1.1 at all, and reaching that cache costs a clock read and an
  IMF-fixdate format on every response that declares a lifetime.
- **A default lifetime for static mounts** — the value that makes an embedded bundle fast is the same
  value that serves a stale shell after a deploy, and no value is right without knowing what is in
  the directory. A framework guessing here is guessing with a promise it cannot withdraw.
- **Excluding `index.html` from every mount's lifetime** — a plain mount has no shell: `index.html`
  is a file like any other there, and one that silently meant "every file except one" would be harder
  to reason about than one that means what it says. Rejected outside SPA mode, where the exclusion is
  instead the whole point (below).
- **`public` on `CacheFor`** — the bare `max-age` form deliberately leaves a response to an
  authenticated request unstorable by shared caches (RFC 9111 §3.5). `public` is what overrides that,
  and it is not something a method named for a duration should do without being asked.
- **A freshness argument on `NotModified`** — one call for both halves, and wrong: the halves are
  independent. A response can be fresh without a validator and validated without a lifetime, and
  pairing them in the signature would make the common static case pass a value it does not have.
- **`immutable`** — safe only when the URL is content-addressed (`app.a3f2c1.js`). Govalin does not
  fingerprint assets, so it belongs to an asset-pipeline discussion, not this one.
- **`Vary`** — adjacent, and a different question: which stored response may answer a request, rather
  than how long any of them lasts. Its own issue.

## Consequences

- Freshness composes with validation without either knowing about the other. A 304 carries the
  `Cache-Control` the 200 would have, so a revalidation renews the stored copy instead of leaving the
  client asking on every reuse — the status flush drops only the headers describing a representation
  (ADR 0009), and a lifetime is not one of them.
- A mount that declares a lifetime serves stale files for its duration, and there is no
  server-side way to withdraw one. That is the cost the opt-in buys deliberately.
- A generated directory listing gets no lifetime, for the same reason it gets no validator: it is
  answered by `http.FileServer` and never passes through `Call`.
- A lifetime is a header like any other, so one declared before the handler ran lands on whatever
  that route ends up answering — a 404 or a 500 included, and an explicit lifetime is what makes an
  error response storable. Clearing it on an error status was rejected: `no-store` on a 500 is the
  same header and is worth keeping, so the framework cannot tell the two apart without inspecting a
  value the caller chose. Documented as a reason to declare a lifetime where the answer is known.
- A SPA mount's fallback is stored under whichever URL asked for it, which is why the shell is
  `no-cache` and not merely short-lived: a lifetime there would mask a route added at that URL later
  for its full duration, on a response served at every URL the app routes.
- A mount cannot declare a lifetime of zero: the zero value is what "no lifetime" is, and telling the
  two apart would cost a field for a case nobody has asked for. Said from a `Before` handler it still
  reaches the response, since a mount overrides only a lifetime it was configured with.
- `s-maxage`, `must-revalidate` and `stale-while-revalidate` are not offered. Each is a real answer to
  a question nobody has asked here yet, and `call.Header` remains available to anyone who needs one.
- Nothing sends `Expires`, so a cache that reads only that header sees what it saw before this change.
