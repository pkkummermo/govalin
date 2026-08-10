# Selecting headers accumulate rather than replace

## Status

accepted

## Context and decision

ADR 0009 said whether a stored response may be reused and ADR 0010 said for how long. Neither says
**which** stored response answers a request. A response produced from a request header — a body
negotiated from `Accept`, a language from `Accept-Language`, anything read off a cookie or
`Authorization` — is stored under its URL alone, and a shared cache hands the copy it happens to hold
to the next client whose headers said something else. Freshness makes that worse rather than better:
it is precisely what stops the next client asking.

`call.VaryOn(names...)` declares the **Selecting headers** a response was chosen from. It **adds** to
what is already declared, which is deliberately the opposite of how `Cache-Control` was made
last-call-wins. A lifetime is a policy about one response, and the last caller is the one that knows
what is being answered. A selecting header is a fact about how the response was produced, and each
layer holds a fact the others cannot see: the CORS plugin knows it read `Origin`, the handler under
it knows it read `Accept-Language`, and a declaration that replaced would drop one of them silently.
What that costs is not a policy nobody wanted but a response handed to the wrong client.

The names are merged into **one field line**, canonicalized, with a name already there declared once.
RFC 9110 §5.3 reads several field lines as the one list, so a second line would be equivalent on the
wire — but merging is what lets a repeated claim be told from a new one, and a plugin running on `*`
under a handler that varies on the same header is the case that produces one.

Nothing else in govalin declares a selecting header. A static mount serves one representation per
URL: it negotiates nothing, and the conditional and range headers it does read select a status rather
than a representation, which caches handle themselves. Sessions read the session cookie on every
request when they are enabled, but reading one is not what makes a response depend on it — the
handler decides that, and `VaryOn` is now what it says so with. A response that mints a session
carries `Set-Cookie`, and what keeps that out of a shared cache is a **Cache scope**, not a cache
key.

## Considered options

- **An additive `VaryOn`, merged into a single field line (chosen)** — see above.
- **Last-call-wins, for symmetry with `Cache-Control`** — the symmetry is in the header shape, not in
  what the header says. Overriding a lifetime is how a route says "not this one"; overriding a
  selecting header is how a route unsays something that happened.
- **Leaving it to `call.Header`, which already appends** — the shape works and the deduplication does
  not: the plugin's `Origin` and the handler's `Origin` are two field lines that no layer can see the
  other in, and neither knows to stay quiet.
- **A `VaryOnAll` for `Vary: *`** — reads as "varies on everything" and means "never reuse this",
  which `NoStore` already says without leaving the response stored under a key nothing matches. A `*`
  passed to `VaryOn` is left alone rather than filtered: it means exactly what RFC 9110 says it does,
  and stripping a caller's value would be a worse surprise than honouring it.
- **`Vary: Accept-Encoding` on static mounts by default** — govalin neither negotiates content nor
  compresses it, so the responses do not depend on the header. Naming one the answer does not depend
  on splits a shared cache's storage per encoding and buys nothing back, which is ADR 0011's argument
  against the preflight headers, made for the same reason.

## Consequences

- A shared cache stores one copy per distinct value of every declared header. That is the true price
  of a selecting header, and it is why declaring one the response does not depend on is not free.
- `Vary` is a response header like any other, so one declared in a `Before` handler lands on whatever
  that route ends up answering, a 404 or a 500 included — the same property freshness has (ADR 0010).
- A 304 carries it: the **Status flush** drops only the headers describing a representation, and a
  cache updating its stored copy from a revalidation needs the key that copy is stored under.
- It weighs less against `CachePrivateFor` than against `CacheFor`. A private cache holds one client's
  copies, so `Authorization` or a session cookie there selects between responses that client would be
  given anyway; in a shared cache the same header is the difference between one client's response and
  another's. `CachePrivateFor` is not a substitute for the declaration — an intermediary that ignores
  `private` is exactly the one that also stored the response — but it is the reason a private response
  rarely needs one.
- ADR 0011's consequence that a handler's own `Vary` arrives as a second field line no longer holds:
  the plugin declares through `VaryOn`, so the handler's names join the plugin's in one list.
- `call.Header(headers.Vary, ...)` still works, and `VaryOn` folds what it wrote into its own list
  rather than sending a duplicate beside it.
