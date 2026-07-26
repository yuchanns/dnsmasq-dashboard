package lease

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"1785016947 2E:C8:06:F2:F3:A5 10.9.60.185 iPhone 01:2e:c8:06:f2:f3:a5",
		"0 40:1a:58:2f:e3:35 10.9.60.233 * *",
	}, "\n")

	leases, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("len(leases) = %d, want 2", len(leases))
	}

	first := leases[0]
	if first.MAC != "2e:c8:06:f2:f3:a5" {
		t.Errorf("MAC = %q", first.MAC)
	}
	if first.Hostname != "iPhone" {
		t.Errorf("Hostname = %q", first.Hostname)
	}
	if first.ExpiresAt == nil || !first.ExpiresAt.Equal(time.Unix(1785016947, 0).UTC()) {
		t.Errorf("ExpiresAt = %v", first.ExpiresAt)
	}

	second := leases[1]
	if !second.Infinite || second.ExpiresAt != nil {
		t.Errorf("infinite lease = %#v", second)
	}
	if second.Hostname != "" || second.ClientID != "" {
		t.Errorf("placeholders were not normalized: %#v", second)
	}
}

func TestParseRejectsMalformedLine(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader("bad line"))
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}
