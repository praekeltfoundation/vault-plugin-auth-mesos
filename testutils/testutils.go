// Package testutils contains some common test util that can be shared across
// modules.
package testutils

import (
	"github.com/stretchr/testify/suite"
)

// TestSuite is a testify test suite object that supports per-test cleanup
// functions and some other helpers.
type TestSuite struct {
	suite.Suite
	cleanups []func()
}

// AddCleanup schedules the given cleanup function to be run after the test.
// Think of it like `defer`, except it applies to the whole test rather than
// the specific function it appears in.
//
// Cleanup functions are run in the reverse of the order in which they were
// added. This makes it safe for later cleanup functions to use entities that
// are cleaned up by earlier cleanup functions.
func (ts *TestSuite) AddCleanup(f func()) {
	ts.cleanups = append([]func(){f}, ts.cleanups...)
}

// SetupTest clears all our TestSuite state at the start of each test, because
// the same object is shared across all tests. This needs to be called from any
// "child" suites that implement their own SetupTest.
func (ts *TestSuite) SetupTest() {
	ts.cleanups = []func(){}
}

// TearDownTest calls the registered cleanup functions. This needs to be called
// from any "child" suites that implement their own TearDownTest.
func (ts *TestSuite) TearDownTest() {
	for _, f := range ts.cleanups {
		f()
	}
}

// WithoutError accepts a (result, error) pair, immediately fails the test if
// there is an error, and returns just the result if there is no error. It
// accepts and returns the result value as an `interface{}`, so it may need to
// be cast back to whatever type it should be afterwards.
func (ts *TestSuite) WithoutError(result interface{}, err error) interface{} {
	ts.T().Helper()
	ts.Require().NoError(err)
	return result
}
