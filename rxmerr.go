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

// Package rxmerr provides a lightweight facade over uber-go/multierr
// for aggregating multiple errors in the DIRPX L7 router.
//
// This package wraps multierr with a simpler API tailored for common
// error collection patterns while maintaining zero-allocation fast paths.
package rxmerr

import "go.uber.org/multierr"

// Combine merges multiple errors into a single error value.
//
// Nil arguments are ignored. If all provided errors are nil, Combine returns
// nil. If exactly one non-nil error is provided, that error is returned as-is.
// If more than one non-nil error is provided, the result is a multi-error
// compatible with go.uber.org/multierr; callers may use Errors to inspect its
// individual components.
//
// This is a thin convenience wrapper around multierr.Combine.
func Combine(errs ...error) error {
	return multierr.Combine(errs...)
}

// Append combines two error values into a single error.
//
// If both left and right are nil, Append returns nil. If exactly one of them
// is non-nil, that non-nil error is returned as-is. If both are non-nil, the
// result is a multi-error that aggregates both and is compatible with
// go.uber.org/multierr. Callers may use Errors to retrieve all underlying
// errors later.
//
// This is a thin convenience wrapper around multierr.Append.
func Append(left, right error) error {
	return multierr.Append(left, right)
}

// Errors returns all underlying errors contained in err.
//
// If err is nil, Errors returns nil. If err is not a multi-error produced by
// this package or go.uber.org/multierr, Errors returns a slice containing
// exactly one element: err itself. If err is a multi-error, Errors returns
// the full set of its constituent errors.
//
// The returned slice MUST be treated as read-only by callers. Its exact
// ordering and other details are defined by go.uber.org/multierr.
//
// This is a thin convenience wrapper around multierr.Errors.
func Errors(err error) []error {
	return multierr.Errors(err)
}

// AppendInto appends an error into the destination pointed to by dst.
//
// The behavior is equivalent to:
//
//	*dst = Append(*dst, err)
//
// If dst is nil, AppendInto panics. This pattern is useful when collecting
// errors in a single variable without introducing explicit temporary values:
//
//	var err error
//	AppendInto(&err, operation1())
//	AppendInto(&err, operation2())
//	return err
//
// Nil err values are ignored in the same way as by Append.
//
// This is a thin convenience wrapper around multierr.AppendInto.
func AppendInto(dst *error, err error) {
	multierr.AppendInto(dst, err)
}

// AppendFunc calls fn and appends its returned error into dst.
//
// This is a convenience helper equivalent to:
//
//	AppendInto(dst, fn())
//
// and is especially useful for closing resources or running best-effort
// cleanup operations:
//
//	var err error
//	AppendFunc(&err, conn.Close)
//	AppendFunc(&err, file.Close)
//	return err
//
// If fn returns nil, nothing is added. If fn returns a non-nil error, it is
// appended using AppendInto. This function does not recover from panics in fn;
// any panic propagates to the caller.
//
// This is a thin convenience wrapper around multierr.AppendFunc.
func AppendFunc(dst *error, fn func() error) {
	multierr.AppendFunc(dst, fn)
}
