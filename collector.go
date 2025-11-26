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

package rxmerr

import "go.uber.org/multierr"

// Collector incrementally accumulates non-nil errors and exposes them as a
// single aggregated error.
//
// This type is intended for simple, sequential use within a single goroutine.
// It wraps multierr.Append/Errors to provide a more ergonomic, stateful API:
//
//	c := rxmerr.NewCollector()
//	c.Append(op1())
//	c.Append(op2())
//	if err := c.Err(); err != nil {
//	    return err
//	}
//
// # Concurrency
//
// Collector is NOT safe for concurrent use. If you need to collect errors from
// multiple goroutines, you MUST add your own synchronization (for example,
// using a mutex) or use a separate Collector per goroutine and merge their
// final errors with multierr.Append at the end.
type Collector struct {
	err   error // aggregated error built via multierr.Append
	count int   // number of non-nil errors that were appended
}

// NewCollector creates a new, empty Collector.
//
// The returned instance contains no errors (Err() returns nil, Len() returns 0)
// and is ready for use. A single Collector MAY be reused across multiple
// logical operations by calling Reset between uses.
func NewCollector() *Collector {
	return &Collector{}
}

// Append adds the provided error to the collector.
//
// If err is nil, Append is a no-op and does not change the internal state.
// If err is non-nil, it is added to the aggregated error using multierr.Append
// and the internal count of non-nil errors is incremented.
//
// Collector does not interpret, wrap, or filter errors by itself; all behavior
// related to aggregation (ordering, flattening, etc.) is delegated to
// go.uber.org/multierr.
func (c *Collector) Append(err error) {
	if err != nil {
		c.err = multierr.Append(c.err, err)
		c.count++
	}
}

// AppendFunc calls fn and appends its returned error to the collector.
//
// This is a convenience helper equivalent to:
//
//	c.Append(fn())
//
// and is especially useful when dealing with Close-like methods:
//
//	c.AppendFunc(conn.Close)
//	c.AppendFunc(file.Close)
//
// If fn returns nil, nothing is added. If fn returns a non-nil error, it is
// accumulated in the same way as via Append. This method does not recover from
// panics in fn: if fn panics, the panic propagates to the caller.
func (c *Collector) AppendFunc(fn func() error) {
	c.Append(fn())
}

// Err returns the aggregated error accumulated so far.
//
// If no non-nil errors were appended, Err returns nil. If one or more
// non-nil errors were appended, Err returns an error value constructed via
// multierr.Append. Callers MAY use multierr.Errors(err) on the returned error
// to inspect all underlying errors if needed.
//
// Err does not reset the collector state; multiple calls return the same
// aggregated error until new errors are appended or Reset is called.
func (c *Collector) Err() error {
	return c.err
}

// Len returns the number of non-nil errors that have been collected so far.
//
// This is a simple counter that increments each time Append is called with a
// non-nil error (or AppendFunc returns a non-nil error). After Reset, Len
// returns 0 until new errors are appended.
func (c *Collector) Len() int {
	return c.count
}

// HasError reports whether at least one non-nil error has been collected.
//
// This is a convenience predicate equivalent to:
//
//	return c.Len() > 0
//
// It is often useful for quick checks when the aggregated error value itself
// is not needed.
func (c *Collector) HasError() bool {
	return c.count > 0
}

// Reset clears all collected errors and prepares the collector for reuse.
//
// After Reset, the collector behaves as if it was newly created:
//   - Err() returns nil;
//   - Len() returns 0;
//   - HasError() returns false.
//
// Any error value previously returned by Err remains valid and independent;
// calling Reset does NOT mutate already returned error instances.
func (c *Collector) Reset() {
	c.err = nil
	c.count = 0
}

// Errors returns all collected non-nil errors as a slice.
//
// If no errors were collected, Errors returns nil. Otherwise it delegates to
// multierr.Errors to extract the underlying error slice from the aggregated
// error stored in the collector.
//
// The returned slice SHOULD be treated as read-only by callers. Its ordering
// and exact behavior are defined by go.uber.org/multierr; callers SHOULD NOT
// rely on a particular concrete implementation beyond what multierr
// documents.
func (c *Collector) Errors() []error {
	if c.err == nil {
		return nil
	}
	return multierr.Errors(c.err)
}
