package testutils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// CleanupTests is a suite for testing cleanup handlers.
type CleanupTests struct {
	TestSuite

	cleanupMsgs []string
	ccbt1       int
	ccbt2       int
}

// The assertion for this test is in Test_TestUtils below.
func (ts *CleanupTests) Test_cleanups_run_in_reverse_order() {
	ts.AddCleanup(func() { ts.cleanupMsgs = append(ts.cleanupMsgs, "first") })
	ts.AddCleanup(func() { ts.cleanupMsgs = append(ts.cleanupMsgs, "second") })
	ts.Nil(ts.cleanupMsgs)
}

// The assertion for this test is in Test_TestUtils below.
func (ts *CleanupTests) Test_cleanups_cleared_between_tests1() {
	ts.Equal(ts.ccbt1, 0)
	ts.AddCleanup(func() { ts.ccbt1++ })
}

// The assertion for this test is in Test_TestUtils below.
func (ts *CleanupTests) Test_cleanups_cleared_between_tests2() {
	ts.Equal(ts.ccbt2, 0)
	ts.AddCleanup(func() { ts.ccbt2++ })
}

// This would typically just run the test suite, but in this case we want to
// make some assertions about the way the test suite ran.
func Test_Cleanup(t *testing.T) {
	ts := new(CleanupTests)
	suite.Run(t, ts)

	// Assertion for Test_cleanups_run_in_reverse_order.
	assert.Equal(t, ts.cleanupMsgs, []string{"second", "first"})
	// Assertions for Test_cleanups_cleared_between_tests
	ts.Equal(ts.ccbt1, 1)
	ts.Equal(ts.ccbt2, 1)
}

// WithoutErrorTests is a suite for testing WithoutError.
type WithoutErrorTests struct {
	TestSuite
}

// WithoutError ignores any nil error values.
func (ts *WithoutErrorTests) Test_WithoutError_without_errors() {
	f1 := func() (int, error) { return 1, nil }
	ts.Equal(ts.WithoutError(f1()).(int), 1)

	f2 := func() (string, error) { return "one", nil }
	ts.Equal(ts.WithoutError(f2()).(string), "one")
}

// WithoutError fails the test immediately for non-nil error values.
func (ts *WithoutErrorTests) Test_WithoutError_with_error() {
	ts.T().Skip("This test is intended to fail. Testing test helpers in Go is hard.")

	fe := func() (int, error) { return 0, errors.New("I meant to do that") }
	ts.WithoutError(fe())
}

// Run WithoutErrorTests.
func Test_WithoutError(t *testing.T) {
	suite.Run(t, new(WithoutErrorTests))
}
