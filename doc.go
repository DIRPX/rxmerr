/*
	Copyright 2025 The DIRPX Authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// Package rxmerr provides small convenience helpers on top of go.uber.org/multierr.
//
// # Overview
//
// Go's built-in error type is intentionally minimal: each function typically
// returns at most a single error value. In many real-world scenarios, however,
// a caller needs to:
//
//   - perform a sequence of best-effort operations (for example, closing
//     multiple resources in a defer chain);
//   - collect errors from those operations without short-circuiting on the
//     first failure;
//   - return a single error value to the caller while preserving access to
//     all underlying failures for diagnostics.
//
// The go.uber.org/multierr package addresses this pattern by providing a
// multi-error implementation that aggregates multiple error values into a
// single error. Package rxmerr builds on top of multierr and offers a thin,
// opinionated API that is easier to integrate into application code.
//
// # Design goals
//
// The package is designed around the following principles:
//
//   - Minimal abstraction:
//     rxmerr does not attempt to re-implement or hide multierr; instead, it
//     wraps multierr in a small set of helpers that are easy to read and
//     straightforward to reason about.
//
//   - Explicit ownership and lifecycle:
//     aggregation is expressed through simple patterns and types (such as
//     Collector) so that the lifetime and scope of accumulated errors remain
//     clear in the calling code.
//
//   - Zero surprises:
//     all top-level helpers are direct, documented wrappers around the
//     corresponding multierr functions. Semantics are deliberately aligned
//     with the upstream implementation to reduce the cognitive load for
//     users familiar with multierr.
//
// # Core components
//
// # Collector
//
// The Collector type is a lightweight, stateful helper that incrementally
// accumulates non-nil errors and exposes them as a single aggregated error.
//
// A typical usage pattern is:
//
//	c := rxmerr.NewCollector()
//	c.Append(op1())
//	c.Append(op2())
//	if err := c.Err(); err != nil {
//	    return err
//	}
//
// Collector is intended for sequential use within a single goroutine. It is
// NOT safe for concurrent use without external synchronization. For concurrent
// workflows, callers SHOULD either:
//   - maintain a separate Collector per goroutine and merge their Err()
//     results at the end using Append or Combine, or
//   - use their own concurrency-safe aggregation structure and delegate
//     multi-error creation to rxmerr only at the final step.
//
// The Collector API is intentionally small:
//
//   - Append(err error) adds a non-nil error to the internal aggregate;
//   - AppendFunc(fn func() error) is a convenience wrapper around Append(fn());
//   - Err() error returns the aggregated error (or nil if nothing was added);
//   - Len() int and HasError() bool expose simple inspection helpers;
//   - Reset() clears the accumulated state for reuse;
//   - Errors() []error exposes all underlying errors via multierr.Errors.
//
// # Free functions
//
// Package-level helpers mirror the core multierr primitives while providing a
// stable surface local to this module:
//
//   - Combine(errs ...error) error
//     Merges a variadic list of error values into a single error. Nil
//     arguments are ignored. If all arguments are nil, Combine returns nil.
//     If exactly one argument is non-nil, that error is returned as-is.
//     Otherwise a multi-error compatible with multierr is returned.
//
//   - Append(left, right error) error
//     Combines two error values into a single error, following the same
//     rules as Combine. This is a convenience wrapper for incremental
//     aggregation when a caller maintains a single error variable.
//
//   - Errors(err error) []error
//     Returns the underlying slice of errors represented by a multi-error,
//     or a single-element slice containing err if it is not a multi-error,
//     or nil if err is nil.
//
//   - AppendInto(dst *error, err error)
//     Appends err into the error pointed to by dst, panicking if dst is nil.
//     This enables concise accumulation patterns without temporary variables:
//
//     var err error
//     rxmerr.AppendInto(&err, op1())
//     rxmerr.AppendInto(&err, op2())
//     return err
//
//   - AppendFunc(dst *error, fn func() error)
//     Equivalent to AppendInto(dst, fn()), often used for best-effort
//     cleanup operations such as closing resources:
//
//     var err error
//     rxmerr.AppendFunc(&err, conn.Close)
//     rxmerr.AppendFunc(&err, file.Close)
//     return err
//
// Relationship to go.uber.org/multierr
//
// All aggregation semantics are delegated to go.uber.org/multierr. rxmerr
// does not define its own ErrorGroup type or alternative multi-error
// representation. Instead:
//
//   - Collector uses multierr.Append and multierr.Errors internally;
//   - Combine, Append, Errors, AppendInto, and AppendFunc are thin wrappers
//     around the corresponding multierr functions.
//
// As a consequence:
//
//   - any error returned by rxmerr functions is compatible with the multierr
//     API and can be inspected or handled using multierr directly;
//   - callers SHOULD consult the multierr documentation for detailed
//     guarantees about ordering, flattening, and other low-level behaviors;
//   - changes in multierr semantics may affect the behavior of this package,
//     although rxmerr strives to remain a stable, documented façade.
//
// # Concurrency considerations
//
// None of the exported helpers in this package are inherently concurrency-safe
// when used with shared mutable state:
//
//   - Collector instances MUST NOT be accessed concurrently without external
//     synchronization;
//   - the free functions (such as AppendInto and AppendFunc) are safe as long
//     as the caller ensures that shared destination error variables are not
//     mutated from multiple goroutines at the same time.
//
// When in doubt, restrict the scope of a Collector or error variable to a
// single goroutine, and perform any necessary merging only after all
// concurrent work has completed.
package rxmerr
