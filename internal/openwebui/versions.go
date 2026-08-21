package openwebui

import (
	"fmt"
	"strconv"
	"strings"
)

// Supported Open-WebUI minors (D5 policy: one current + one prior, pinned in
// docs/supported-versions.md). This window drives the version probe notice.
var supportedMinors = map[int]bool{10: true, 11: true}

const supportedWindow = "0.10–0.11"

// Supported reports whether version falls inside the pinned compatibility
// window (0.10.x, 0.11.x). When it does not, notice states why the client
// proceeds in completions-only mode (D5: never silent failure).
func Supported(version string) (ok bool, notice string) {
	major, minor, err := parseMinor(version)
	if err != nil {
		return false, fmt.Sprintf("unrecognized Open-WebUI version %q", version)
	}
	if major != 0 || !supportedMinors[minor] {
		return false, fmt.Sprintf("unsupported Open-WebUI version %s (supported: %s)", version, supportedWindow)
	}
	return true, ""
}

func parseMinor(version string) (major, minor int, err error) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("not a semver")
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}
