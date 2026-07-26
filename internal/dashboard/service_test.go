package dashboard

import (
	"net/netip"
	"testing"
	"time"

	"github.com/yuchanns/dnsmasq-dashboard/internal/config"
	"github.com/yuchanns/dnsmasq-dashboard/internal/lease"
	"github.com/yuchanns/dnsmasq-dashboard/internal/neighbor"
)

func TestBuildSnapshot(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_000, 0).UTC()
	expiry := now.Add(time.Hour)
	cfg := config.Config{
		NetworkName: "Test LAN",
		Interface:   "eth0",
		PoolStart:   netip.MustParseAddr("10.0.0.100"),
		PoolEnd:     netip.MustParseAddr("10.0.0.109"),
	}
	leases := []lease.Lease{
		{
			ExpiresAt: &expiry,
			MAC:       "00:11:22:33:44:55",
			IP:        netip.MustParseAddr("10.0.0.101"),
			Hostname:  "phone",
		},
		{
			Infinite: true,
			MAC:      "00:11:22:33:44:66",
			IP:       netip.MustParseAddr("10.0.0.102"),
		},
	}
	neighbors := []neighbor.Entry{
		{IP: "10.0.0.101", MAC: "00:11:22:33:44:55", States: []string{"REACHABLE"}},
		{IP: "10.0.0.102", MAC: "00:11:22:33:44:99", States: []string{"STALE"}},
	}

	snapshot := buildSnapshot(cfg, leases, neighbors, now, nil)
	if snapshot.Summary.Capacity != 10 || snapshot.Summary.Leased != 2 || snapshot.Summary.Available != 8 {
		t.Errorf("summary = %#v", snapshot.Summary)
	}
	if snapshot.Summary.Online != 1 || snapshot.Summary.Conflicts != 1 {
		t.Errorf("status counts = %#v", snapshot.Summary)
	}
	if snapshot.Leases[1].Status != StatusConflict {
		t.Errorf("Status = %q", snapshot.Leases[1].Status)
	}
}

func TestNeighborStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		states []string
		want   Status
	}{
		{[]string{"REACHABLE"}, StatusOnline},
		{[]string{"STALE"}, StatusRecent},
		{[]string{"FAILED"}, StatusOffline},
		{nil, StatusOffline},
	}
	for _, test := range tests {
		if got := neighborStatus(test.states); got != test.want {
			t.Errorf("neighborStatus(%v) = %q, want %q", test.states, got, test.want)
		}
	}
}

func TestRevisionIgnoresRefreshTime(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		NetworkName: "Test LAN",
		Interface:   "eth0",
		PoolStart:   netip.MustParseAddr("10.0.0.100"),
		PoolEnd:     netip.MustParseAddr("10.0.0.109"),
	}
	leases := []lease.Lease{
		{
			Infinite: true,
			MAC:      "00:11:22:33:44:55",
			IP:       netip.MustParseAddr("10.0.0.101"),
		},
	}

	first := buildSnapshot(cfg, leases, nil, time.Unix(1_700_000_000, 0).UTC(), nil)
	second := buildSnapshot(cfg, leases, nil, time.Unix(1_700_000_010, 0).UTC(), nil)
	if first.Revision != second.Revision {
		t.Fatalf("revision changed without a data change: %q != %q", first.Revision, second.Revision)
	}
}
