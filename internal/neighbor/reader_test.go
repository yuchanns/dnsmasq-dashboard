package neighbor

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()

	entries, err := Parse([]byte(`[
		{"dst":"10.9.60.122","lladdr":"EE:78:6E:14:AA:6A","state":["REACHABLE"]},
		{"dst":"10.9.60.200","state":["FAILED"]}
	]`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].MAC != "ee:78:6e:14:aa:6a" {
		t.Errorf("MAC = %q", entries[0].MAC)
	}
	if entries[1].States[0] != "FAILED" {
		t.Errorf("state = %q", entries[1].States[0])
	}
}
