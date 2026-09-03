package config

import (
	"os"
	"testing"
)

func TestIsApprovedOriginWithPublicURL(t *testing.T) {
	// simulate setting PUBLIC_URL in .env
	os.Setenv("PUBLIC_URL", "https://bpl.starventures.org")
	defer os.Unsetenv("PUBLIC_URL")

	// staging domain should be approved via PUBLIC_URL
	if !IsApprovedOrigin("https://bpl.starventures.org") {
		t.Errorf("Expected bpl.starventures.org to be approved via PUBLIC_URL")
	}

	// staging domain with port should also be approved
	if !IsApprovedOrigin("https://bpl.starventures.org:8080") {
		t.Errorf("Expected bpl.starventures.org:8080 to be approved via PUBLIC_URL")
	}

	// static production domain should still be approved
	if !IsApprovedOrigin("https://bpl-poe.com") {
		t.Errorf("Expected bpl-poe.com to be approved")
	}

	// random domains should be rejected (xdd)
	if IsApprovedOrigin("https://spooky-evil-liberator.com") {
		t.Errorf("Expected spooky-evil-liberator.com to be rejected")
	}
}
