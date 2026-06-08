# Transform-aware string validation pipeline

## Status

accepted

## Context and decision

The curryable `StringValidator` accumulated rules as `func(string, string) error` and `Get()`
returned the original captured value unchanged — the pipeline could assert on a value but never
change it. To support normalizing transforms (the first being `Trim()`, which removes surrounding
whitespace), we changed the internal rule type to `func(string, string) (string, error)` so each
step can return a possibly-modified value alongside an error. `Get()` threads the value through the
chain in order and returns the final transformed result. Public method signatures are unchanged;
only the unexported `rules` field type changed.

This makes the chain position-sensitive: a transform only affects checks placed after it
(`Trim().MinLength(3)` measures the trimmed value; reversing them does not). Transform steps never
return an error.

## Considered options

- **Transform-aware pipeline (chosen)** — rules return `(string, error)`. Keeps the lazy/curried
  contract, makes ordering meaningful and honest, generalizes to future transforms.
- **Eager mutation** — `Trim()` mutates the captured value immediately when called. Rejected: breaks
  the lazy contract and makes a step's written position meaningless.
- **Separate transforms slice** — keep rules error-only, add a parallel transform list applied first.
  Rejected: loses interleaved ordering between transforms and validations.

## Consequences

- `IntValidator` and `BodyValidator` keep the older error-only shape; the sibling validators now have
  different internal pipeline shapes. This is deliberate (YAGNI — no numeric/body transform exists yet)
  and invisible to users since the field is unexported.
- Body-field trimming is deferred: it would require writing the trimmed value back into the caller's
  struct via reflection, which is a separate semantic decision.
