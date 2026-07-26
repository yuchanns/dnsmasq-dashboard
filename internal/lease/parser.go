package lease

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type Lease struct {
	ExpiresAt *time.Time
	Infinite  bool
	MAC       string
	IP        netip.Addr
	Hostname  string
	ClientID  string
}

func Parse(reader io.Reader) ([]Lease, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var leases []Lease
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 || len(fields) > 5 {
			return nil, fmt.Errorf("line %d: expected 4 or 5 fields, got %d", lineNumber, len(fields))
		}

		expiry, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || expiry < 0 {
			return nil, fmt.Errorf("line %d: invalid expiry %q", lineNumber, fields[0])
		}

		mac, err := net.ParseMAC(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid MAC address %q", lineNumber, fields[1])
		}

		ip, err := netip.ParseAddr(fields[2])
		if err != nil || !ip.Is4() {
			return nil, fmt.Errorf("line %d: invalid IPv4 address %q", lineNumber, fields[2])
		}

		entry := Lease{
			Infinite: expiry == 0,
			MAC:      strings.ToLower(mac.String()),
			IP:       ip,
			Hostname: normalizePlaceholder(fields[3]),
		}
		if expiry != 0 {
			value := time.Unix(expiry, 0).UTC()
			entry.ExpiresAt = &value
		}
		if len(fields) == 5 {
			entry.ClientID = normalizePlaceholder(fields[4])
		}
		leases = append(leases, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read leases: %w", err)
	}

	return leases, nil
}

func normalizePlaceholder(value string) string {
	if value == "*" {
		return ""
	}
	return value
}
