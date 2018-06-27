package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_HumanReadable_no_prerelease(t *testing.T) {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123+CHANGES"
	VersionPrerelease = ""

	hr := HumanReadable()
	assert.Contains(t, hr, "Git Commit: "+GitCommit)
	assert.Contains(t, hr, "Version: "+Version)
	assert.NotContains(t, hr, "Version: "+Version+"-")
}

func Test_HumanReadable_with_prerelease(t *testing.T) {
	// Override global variables (usually set at compile time).
	GitCommit = "abc123"
	VersionPrerelease = "DEV"

	hr := HumanReadable()
	assert.Contains(t, hr, "Git Commit: "+GitCommit)
	assert.Contains(t, hr, "Version: "+Version+"-DEV")
}
