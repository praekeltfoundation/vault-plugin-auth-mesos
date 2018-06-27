package version

// GitCommit references the git commit that was compiled. This will be filled
// in at compile-time.
var GitCommit string

// Version is the main version number that is being run at the moment.
const Version = "0.1.0"

// VersionPrerelease is a pre-release marker for the version. If this is ""
// (empty string) then it means that it is a final release. Otherwise, this is
// a pre-release such as "dev" (in development)
var VersionPrerelease = ""

// HumanReadable emits a human-readable version string.
func HumanReadable() string {
	v := "Version: " + Version
	if VersionPrerelease != "" {
		v += "-" + VersionPrerelease
	}
	v += " Git Commit: " + GitCommit
	return v
}
