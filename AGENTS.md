# Agent Guidelines

## Formatting

Go code is formatted with **gofumpt**, not gofmt — run `gofumpt -w` on the files you touch. It is
enforced as a golangci-lint formatter, so gofmt-clean code can still fail lint.

## Code comments

Govalin is a public library, so an **exported identifier carries a doc comment** — that is the API
reference, and godoc is where users read it. State the contract the caller gets, in a sentence or
two. A doc comment may run longer when the contract does — what a caller cannot discover by reading
the signature, such as which failures `Stream` returns and which it swallows. Length in a doc
comment is a question about the contract; length inside a function body never is.

Everything else defaults to **no comment**: inside a function, names and structure should carry the
meaning.

Write one only to record a non-obvious *why* a future reader cannot recover from the code itself: a
spec requirement, an external-bug workaround, or a deliberate choice someone would otherwise "fix".
One line.

Never:

- explain why a change was made or what alternative was rejected — that is commit/PR/ADR material
  (if a comment would fit in a PR description or an ADR's Considered Options, it does not belong in
  the source);
- restate what the code does;
- write multi-line narrative or reasoning blocks.

### Examples

```go
// BAD — restates the code:
// Reserve port and buffer incoming connections
listener, err := net.Listen("tcp", addr)

// BAD — a reasoning block re-arguing a decision, with the rejected alternative inline:
// Flush the buffered status on every non-bypass exit path so a handler (or a
// short-circuiting before handler) that sets a status but writes no body still sends
// that status instead of an implicit 200 OK. The bypass check is evaluated when the
// defer runs, not when it is registered, because HTTPServe sets bypassLifecycle inside
// the handler which runs after this point; a snapshot here would wrongly flush over a
// raw response. Registered after the access-log defer so LIFO ordering flushes before
// the log records the status. ...

// GOOD — one line, the non-obvious why the code can't tell you:
// Read when the defer fires, not when it is registered: HTTPServe sets it later (ADR 0002).

// GOOD — cites the decision instead of restating it:
// The call ID is deliberately logged as sent — see ADR 0006.
```

When in doubt, delete the comment: if the next reader could recover the fact from the code, it is
noise. What the code does belongs in the code, what a decision cost belongs in
[docs/adr/](docs/adr/), and the vocabulary belongs in [CONTEXT.md](CONTEXT.md).
