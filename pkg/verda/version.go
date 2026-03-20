package verda

import (
	"runtime/debug"
	"strings"
)

const (
	// fallbackVersion is used when build info is not available (e.g., during development)
	fallbackVersion = "1.3.0"

	// sdkName is the identifier for this SDK
	sdkName = "verdacloud-sdk-go"
)

// SDKVersion returns the version of the SDK.
// It attempts to get the version from Go module build info (which works when
// the SDK is used as a dependency). Falls back to a hardcoded version for
// development/testing scenarios.
func SDKVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		// When used as a dependency, find our module in the deps
		for _, dep := range info.Deps {
			if strings.HasSuffix(dep.Path, sdkName) {
				return strings.TrimPrefix(dep.Version, "v")
			}
		}
		// When running from within the module itself, use Main
		if strings.HasSuffix(info.Main.Path, sdkName) && info.Main.Version != "(devel)" {
			return strings.TrimPrefix(info.Main.Version, "v")
		}
	}
	return fallbackVersion
}

// DefaultUserAgent returns the default User-Agent string for the SDK.
func DefaultUserAgent() string {
	return sdkName + "/" + SDKVersion()
}

// BuildUserAgent constructs the full User-Agent string.
// If customUA is provided, it prepends it to the SDK's default User-Agent.
// If customUA is empty, it returns just the SDK's default User-Agent.
// The custom User-Agent is sanitized to remove control characters and capped at 256 characters.
func BuildUserAgent(customUA string) string {
	defaultUA := DefaultUserAgent()
	if customUA == "" {
		return defaultUA
	}
	sanitized := sanitizeUserAgent(customUA)
	if sanitized == "" {
		return defaultUA
	}
	return sanitized + " " + defaultUA
}

const maxUserAgentLen = 256

// sanitizeUserAgent strips control characters and caps length to prevent
// header injection, log pollution, and downstream parsing issues.
func sanitizeUserAgent(ua string) string {
	var b strings.Builder
	b.Grow(len(ua))
	for _, r := range ua {
		if r >= 0x20 && r != 0x7F {
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if len(result) > maxUserAgentLen {
		result = result[:maxUserAgentLen]
	}
	return result
}
