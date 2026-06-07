package version

import (
	"regexp"
	"testing"
)

func TestVersionFormat(t *testing.T) {
	// Version should be a valid semver string
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semverRe.MatchString(Version) {
		t.Errorf("Version %q does not match semver format X.Y.Z", Version)
	}
}

func TestVersionNonEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version must not be empty")
	}
}
