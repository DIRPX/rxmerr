# rxmerr

Small, opinionated façade over [`go.uber.org/multierr`](https://pkg.go.dev/go.uber.org/multierr), originally 
built for the DIRPX L7 router but generic enough for any Go service.

`rxmerr` keeps the power and semantics of `multierr` while providing a minimal, ergonomic 
surface for everyday use: simple helpers plus a tiny `Collector` type.

---

## Motivation

Go’s built-in `error` type is deliberately simple: a function usually returns at most a single error value.

In many real-world scenarios you instead need to:

- run several best-effort operations (closing multiple resources, applying several mutations, flushing buffers, etc.);
- collect all failures instead of stopping at the first one;
- return **one** error value while still being able to inspect each 
underlying error later (for logging, metrics, tests, or retries).

[`go.uber.org/multierr`](https://pkg.go.dev/go.uber.org/multierr) solves the aggregation problem.
`rxmerr` builds on it and adds:

- **Minimal API** – no custom error types or interfaces; just functions and a small `Collector` struct;
- **Predictable semantics** – behavior is aligned with `multierr`, so there are no surprises if you already know it;
- **Hot-path friendly** – avoids unnecessary allocations and keeps usage patterns explicit;
- **Drop-in** – everything returned from this package is compatible with `multierr.Errors` and friends.

---

## Installation

```bash
go get github.com/dirpx/rxmerr
```

Then:

```go
import "github.com/dirpx/rxmerr"
```

---

## When to use rxmerr

Use `rxmerr` when:

- you already depend on or are comfortable with `go.uber.org/multierr`;
- you want a **small, explicit** layer over it rather than ad‑hoc `multierr.Append` calls scattered across the codebase;
- you like the `Collector` pattern for sequential operations in one function.

Do **not** use `rxmerr` if:

- you need a fully featured error framework (wrapping, tagging, localization, etc.);
- you want to avoid `multierr` entirely (this package is intentionally a thin façade around it).

---

## API overview

### Collector

`Collector` is a lightweight, stateful helper that incrementally accumulates non‑nil errors and exposes them as a single aggregated error.

Typical usage:

```go
func processRoutes(routes []Route) error {
    c := rxmerr.NewCollector()

    for _, r := range routes {
        c.Append(validateRoute(r))
        c.Append(registerRoute(r))
    }

    if c.HasError() {
        log.Printf("collected %d errors while processing routes", c.Len())
    }

    return c.Err()
}
```

Key properties:

- **Sequential use only** – `Collector` is **not** concurrency‑safe.
  Use it from a single goroutine, or protect it with your own synchronization.
- **Nil-safe** – `Append(nil)` is a no-op.
- **Result shape**:
  - if no non‑nil errors were appended, `Err()` returns `nil`;
  - if exactly one non‑nil error was appended, `Err()` returns that error;
  - otherwise `Err()` returns a multi‑error compatible with `multierr`.
- **Inspection**:
  - `Len()` returns the number of non‑nil errors appended;
  - `HasError()` is equivalent to `Len() > 0`;
  - `Errors()` exposes all underlying errors as a slice (via `multierr.Errors`).
- **Reuse**:
  - `Reset()` clears accumulated state so the same instance can be reused in a new logical operation.

#### Using AppendFunc

`AppendFunc` is a convenience for Close‑style calls:

```go
func closeAll(conn io.Closer, file io.Closer) error {
    c := rxmerr.NewCollector()

    c.AppendFunc(conn.Close)
    c.AppendFunc(file.Close)

    return c.Err()
}
```

Under the hood, `AppendFunc(fn)` is just `Append(fn())`.

---

### Top-level helpers

If you already maintain an `error` variable in a function, free‑standing helpers can be more natural than `Collector`.

#### Combine

```go
err := rxmerr.Combine(err1, err2, err3)
```

Rules:

- `nil` arguments are ignored;
- if all arguments are `nil`, result is `nil`;
- if exactly one argument is non‑nil, that error is returned as‑is;
- otherwise you get a multi‑error that can be unpacked with `rxmerr.Errors`.

Internally this is a thin wrapper over `multierr.Combine`.

#### Append

Incrementally accumulate into a single error:

```go
var err error

err = rxmerr.Append(err, op1())
err = rxmerr.Append(err, op2())
```

Semantics match `Combine` for two arguments and delegate to `multierr.Append`.

#### Errors

Extract all underlying errors, regardless of how the error was constructed:

```go
errs := rxmerr.Errors(err)
for _, e := range errs {
    log.Println("failure:", e)
}
```

Behavior:

- if `err` is `nil`, returns `nil`;
- if `err` is not a multi‑error, returns a slice with a single element (`err`);
- if `err` is a multi‑error, returns all constituents.

Delegates to `multierr.Errors`.

#### AppendInto

Append directly into a pointer to `error`:

```go
var err error

rxmerr.AppendInto(&err, op1())
rxmerr.AppendInto(&err, op2())

return err
```

This pattern is convenient in `defer` blocks and fluent clean‑up code.

Notes:

- if `dst` is `nil`, `AppendInto` panics (same as `multierr.AppendInto`);
- `err == nil` is ignored.

#### AppendFunc

Like `AppendInto`, but calls a function first:

```go
var err error

rxmerr.AppendFunc(&err, conn.Close)
rxmerr.AppendFunc(&err, file.Close)

return err
```

Which is equivalent to:

```go
rxmerr.AppendInto(&err, conn.Close())
rxmerr.AppendInto(&err, file.Close())
```

Delegates to `multierr.AppendFunc`.

---

## Concurrency notes

`rxmerr` itself does not introduce any additional synchronization. The rules are:

- Each `Collector` instance is intended for **single‑goroutine** use.
  If multiple goroutines must share it, protect access with a mutex.
- Top‑level helpers (`AppendInto`, `AppendFunc`, `Combine`, `Append`, `Errors`) are pure functions **except** for writing into the `*error` given to `AppendInto` / `AppendFunc`.
  You must ensure that a given `*error` is not mutated concurrently from multiple goroutines.

A common pattern in concurrent code is:

1. Each goroutine collects its own error (either a `Collector` or a plain `error` with `AppendInto`).
2. The parent goroutine combines the final results with `rxmerr.Append` or `rxmerr.Combine`.

---

## Relationship to go.uber.org/multierr

This package is intentionally thin:

- it does **not** redefine a custom multi‑error type;
- all aggregation semantics (ordering, flattening, wrapping) are fully owned by `go.uber.org/multierr`;
- everything returned from `rxmerr` is safe to inspect with the standard `multierr` API.

If you need to understand the precise behavior of multi‑errors (for example, how errors are flattened or how formatting behaves), refer to the upstream `multierr` documentation. `rxmerr` aims to remain a small, documented façade around it.
