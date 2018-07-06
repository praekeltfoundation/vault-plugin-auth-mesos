package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A human-readable version string does not contain a version suffix if there
// is no prerelease string.
func Test_HumanReadable_no_prerelease(t *testing.T) {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123+CHANGES"
	VersionPrerelease = ""

	hr := HumanReadable()
	assert.Contains(t, hr, "Git Commit: "+GitCommit)
	assert.Contains(t, hr, "Version: "+Version)
	assert.NotContains(t, hr, "Version: "+Version+"-")
}

// A human-readable version string contains a version suffix if there is a
// prerelease string.
func Test_HumanReadable_with_prerelease(t *testing.T) {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123"
	VersionPrerelease = "DEV"

	hr := HumanReadable()
	assert.Contains(t, hr, "Git Commit: "+GitCommit)
	assert.Contains(t, hr, "Version: "+Version+"-DEV")
}
