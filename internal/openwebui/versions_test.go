package openwebui

import (
	"strings"
	"testing"
)

func TestSupported(t *testing.T) {
	tests := []struct {
		version    string
		wantOK     bool
		wantNotice string // substring; "" means no assertion on the notice
	}{
		{"0.11.0", true, ""},
		{"0.10.2", true, ""},
		{"0.10.0", true, ""},
		{"0.12.0", false, "unsupported Open-WebUI version 0.12.0"},
		{"0.9.1", false, "unsupported"},
		{"1.0.0", false, "unsupported"},
		{"0.11", true, ""},
		{"garbage", false, "unrecognized"},
		{"", false, "unrecognized"},
		{" 0.11.0 ", true, ""}, // tolerant of surrounding whitespace
	}
	for _, tt := range tests {
		ok, notice := Supported(tt.version)
		if ok != tt.wantOK {
			t.Errorf("Supported(%q) ok = %v, want %v", tt.version, ok, tt.wantOK)
		}
		if tt.wantNotice != "" && !strings.Contains(notice, tt.wantNotice) {
			t.Errorf("Supported(%q) notice = %q, want it to contain %q", tt.version, notice, tt.wantNotice)
		}
	}
}

func TestParseMinor(t *testing.T) {
	if major, minor, err := parseMinor("0.10.2"); err != nil || major != 0 || minor != 10 {
		t.Errorf("parseMinor(0.10.2) = %d.%d, %v", major, minor, err)
	}
	if _, _, err := parseMinor("banana"); err == nil {
		t.Error("parseMinor(banana) should fail")
	}
}
