# Vary on Origin for every response the CORS plugin sees

## Status

accepted

## Context and decision

The plugin sent `Vary: Access-Control-Allow-Origin`. `Vary` names the **request** headers a response
was selected from, and that is a response header — so the value named a header no request carries, a
cache keyed on nothing, and one stored response answered every origin. The response does vary on
`Origin`: the requesting origin is written straight into `Access-Control-Allow-Origin`. The header
is `Vary: Origin`.

It is sent **before the allowed-origin check**, so a rejected origin gets it too. That response is as
wrong to reuse as an accepted one's — stored without `Vary`, the bare answer given to an origin the
config excludes is handed to the next request from an origin it permits, and the browser blocks a
request the server was willing to allow. What the response was selected from is the `Origin` header,
not whether the plugin liked what it read there.

A **preflight** names no further selecting header. `Access-Control-Request-Method` and
`Access-Control-Request-Headers` are request headers, and a preflight answer built out of them would
be selected from them — but this plugin answers from its configuration: the allowed methods and
headers are the same list whatever the preflight asked for. Naming a header the answer does not
depend on splits a cache's storage per requested method-and-header combination and buys nothing back.

## Considered options

- **`Vary: Origin`, sent unconditionally, and nothing else (chosen)** — see above.
- **Adding the two `Access-Control-Request-*` headers on preflights** — correct for a plugin that
  echoes the requested method and headers back, which is what the advice to send them assumes and what
  this one does not do. It becomes the right answer the day the preflight response is computed from
  the preflight request; until then it declares a dependency that isn't there.
- **Sending `Vary` only once the origin is allowed** — the narrow reading of "this response varies",
  and it leaves the rejected answer cacheable under a key that ignores the header it was chosen by.
  The disallowed case is the one where the missing `Vary` is invisible, since neither response carries
  a CORS header to notice the mix-up by.

## Consequences

- Every response from an app with the plugin installed carries `Vary: Origin`, whether or not it has
  anything to do with CORS — the plugin's before handler runs on `*`. A shared cache stores a copy per
  origin, which is the true cost of a response header derived from a request header.
- `call.Header` adds rather than sets, so a handler that declares its own `Vary` sends a second field
  line rather than replacing this one. Both are read, per RFC 9110 §5.3.
- Freshness (ADR 0010) is what makes the reuse window long enough to matter: a response with a
  declared lifetime is one a cache serves again rather than revalidates, so the response that gets
  handed to the wrong origin is handed over for the full lifetime.
