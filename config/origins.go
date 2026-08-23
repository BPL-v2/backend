package config

import (
	"net/url"
	"strings"
)

// staticApprovedOrigins are frontend origins that are always allowed, in
// addition to localhost/127.0.0.1 (any port) and whatever is configured via
// the APPROVED_TEST_DOMAINS environment variable.
var staticApprovedOrigins = []string{
	"https://bpl-poe.com",
	"https://bpl-2.netlify.app",
	"https://v2202503259898322516.goodsrv.de",
	"https://bpl-v2.github.io/",
}

// approvedTestHostnames returns the extra hostnames (no scheme/port) that
// were whitelisted via the APPROVED_TEST_DOMAINS env var, e.g.
// "APPROVED_TEST_DOMAINS=staging.bpl-poe.com,my-test-domain.com".
func approvedTestHostnames() []string {
	raw := getEnvWithDefault("APPROVED_TEST_DOMAINS", "")
	if raw == "" {
		return nil
	}
	var hostnames []string
	for _, host := range strings.Split(raw, ",") {
		host = strings.TrimSpace(host)
		if host != "" {
			hostnames = append(hostnames, host)
		}
	}
	return hostnames
}

// ApprovedFrontendOrigins returns the static list of fully-qualified origins
// that are allowed to talk to this backend (used for CORS).
func ApprovedFrontendOrigins() []string {
	return staticApprovedOrigins
}

// IsApprovedOrigin reports whether the given origin (scheme://host[:port])
// is allowed to interact with this backend: either one of the static
// production origins, localhost/127.0.0.1 on any port, or a hostname
// configured via APPROVED_TEST_DOMAINS (any port).
func IsApprovedOrigin(origin string) bool {
	for _, approved := range staticApprovedOrigins {
		if origin == approved {
			return true
		}
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	hostname := parsed.Hostname()
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return true
	}
	for _, approved := range approvedTestHostnames() {
		if hostname == approved {
			return true
		}
	}
	return false
}
