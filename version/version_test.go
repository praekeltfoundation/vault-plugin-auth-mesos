package version

import (
	"testing"

	"github.com/praekeltfoundation/vault-plugin-auth-mesos/testutils"
	"github.com/stretchr/testify/suite"
)

// VersionTests is a testify test suite object that we can attach helper
// methods to.
type VersionTests struct{ testutils.TestSuite }

// Test_Version is a standard Go test function that runs our test suite's
// tests.
func Test_Version(t *testing.T) { suite.Run(t, new(VersionTests)) }

// A human-readable version string does not contain a version suffix if there
// is no prerelease string.
func (ts *VersionTests) Test_HumanReadable_no_prerelease() {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123+CHANGES"
	VersionPrerelease = ""

	hr := HumanReadable()
	ts.Contains(hr, "Git Commit: "+GitCommit)
	ts.Contains(hr, "Version: "+Version)
	ts.NotContains(hr, "Version: "+Version+"-")
}

// A human-readable version string contains a version suffix if there is a
// prerelease string.
func (ts *VersionTests) Test_HumanReadable_with_prerelease() {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123"
	VersionPrerelease = "DEV"

	hr := HumanReadable()
	ts.Contains(hr, "Git Commit: "+GitCommit)
	ts.Contains(hr, "Version: "+Version+"-DEV")
}
