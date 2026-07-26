package dashboard

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yuchanns/dnsmasq-dashboard/internal/config"
	"github.com/yuchanns/dnsmasq-dashboard/internal/lease"
	"github.com/yuchanns/dnsmasq-dashboard/internal/neighbor"
)

type Status string

const (
	StatusOnline   Status = "online"
	StatusRecent   Status = "recent"
	StatusOffline  Status = "offline"
	StatusConflict Status = "conflict"
)

type LeaseView struct {
	IP            string     `json:"ip"`
	MAC           string     `json:"mac"`
	Hostname      string     `json:"hostname"`
	ClientID      string     `json:"clientId"`
	ExpiresAt     *time.Time `json:"expiresAt"`
	Infinite      bool       `json:"infinite"`
	Status        Status     `json:"status"`
	NeighborState string     `json:"neighborState"`
	ObservedMAC   string     `json:"observedMac,omitempty"`
}

type Summary struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	PoolStart string `json:"poolStart"`
	PoolEnd   string `json:"poolEnd"`
	Capacity  int    `json:"capacity"`
	Leased    int    `json:"leased"`
	Available int    `json:"available"`
	Online    int    `json:"online"`
	Recent    int    `json:"recent"`
	Conflicts int    `json:"conflicts"`
}

type Snapshot struct {
	Revision    string      `json:"revision"`
	GeneratedAt time.Time   `json:"generatedAt"`
	Healthy     bool        `json:"healthy"`
	Summary     Summary     `json:"summary"`
	Leases      []LeaseView `json:"leases"`
	Warnings    []string    `json:"warnings"`
}

type Service struct {
	config    config.Config
	neighbors neighbor.Reader
	logger    *slog.Logger

	mu       sync.RWMutex
	snapshot Snapshot
}

func NewService(cfg config.Config, neighborReader neighbor.Reader, logger *slog.Logger) *Service {
	return &Service{
		config:    cfg,
		neighbors: neighborReader,
		logger:    logger,
		snapshot: Snapshot{
			GeneratedAt: time.Now().UTC(),
			Summary:     baseSummary(cfg),
			Leases:      []LeaseView{},
			Warnings:    []string{"Waiting for the first refresh"},
		},
	}
}

func (service *Service) Run(ctx context.Context) {
	service.refresh(ctx)
	ticker := time.NewTicker(service.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			service.refresh(ctx)
		}
	}
}

func (service *Service) Current() Snapshot {
	service.mu.RLock()
	defer service.mu.RUnlock()

	copy := service.snapshot
	copy.Leases = slices.Clone(service.snapshot.Leases)
	copy.Warnings = slices.Clone(service.snapshot.Warnings)
	return copy
}

func (service *Service) refresh(ctx context.Context) {
	now := time.Now().UTC()
	leases, leaseErr := readLeases(service.config.LeaseFile)
	neighbors, neighborErr := service.neighbors.Read(ctx)

	warnings := make([]string, 0, 2)
	if leaseErr != nil {
		warnings = append(warnings, leaseErr.Error())
	}
	if neighborErr != nil {
		warnings = append(warnings, neighborErr.Error())
	}

	service.mu.RLock()
	previous := service.snapshot
	service.mu.RUnlock()

	if leaseErr != nil {
		leases = modelLeases(previous.Leases)
	}
	if neighborErr != nil {
		neighbors = nil
	}

	next := buildSnapshot(service.config, leases, neighbors, now, warnings)
	if next.Revision == previous.Revision {
		next.Revision = previous.Revision
	}

	service.mu.Lock()
	service.snapshot = next
	service.mu.Unlock()

	if len(warnings) > 0 {
		service.logger.Warn("dashboard refresh completed with warnings", "warnings", warnings)
	}
}

func readLeases(path string) ([]lease.Lease, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read lease file: %w", err)
	}
	defer file.Close()

	leases, err := lease.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("parse lease file: %w", err)
	}
	return leases, nil
}

func buildSnapshot(
	cfg config.Config,
	leases []lease.Lease,
	neighbors []neighbor.Entry,
	now time.Time,
	warnings []string,
) Snapshot {
	neighborsByIP := make(map[string]neighbor.Entry, len(neighbors))
	for _, entry := range neighbors {
		neighborsByIP[entry.IP] = entry
	}

	views := make([]LeaseView, 0, len(leases))
	summary := baseSummary(cfg)
	for _, item := range leases {
		view := LeaseView{
			IP:        item.IP.String(),
			MAC:       item.MAC,
			Hostname:  item.Hostname,
			ClientID:  item.ClientID,
			ExpiresAt: item.ExpiresAt,
			Infinite:  item.Infinite,
			Status:    StatusOffline,
		}
		if observed, ok := neighborsByIP[view.IP]; ok {
			view.NeighborState = strings.Join(observed.States, ",")
			view.ObservedMAC = observed.MAC
			view.Status = neighborStatus(observed.States)
			if observed.MAC != "" && !strings.EqualFold(observed.MAC, item.MAC) {
				view.Status = StatusConflict
			}
		}

		switch view.Status {
		case StatusOnline:
			summary.Online++
		case StatusRecent:
			summary.Recent++
		case StatusConflict:
			summary.Conflicts++
		}
		views = append(views, view)
	}

	slices.SortFunc(views, func(left, right LeaseView) int {
		leftIP := netip.MustParseAddr(left.IP)
		rightIP := netip.MustParseAddr(right.IP)
		return leftIP.Compare(rightIP)
	})

	summary.Leased = len(views)
	summary.Available = max(0, summary.Capacity-summary.Leased)

	snapshot := Snapshot{
		GeneratedAt: now,
		Healthy:     len(warnings) == 0,
		Summary:     summary,
		Leases:      views,
		Warnings:    warnings,
	}
	snapshot.Revision = revision(snapshot)
	return snapshot
}

func baseSummary(cfg config.Config) Summary {
	return Summary{
		Name:      cfg.NetworkName,
		Interface: cfg.Interface,
		PoolStart: cfg.PoolStart.String(),
		PoolEnd:   cfg.PoolEnd.String(),
		Capacity:  ipv4Distance(cfg.PoolStart, cfg.PoolEnd) + 1,
	}
}

func ipv4Distance(start, end netip.Addr) int {
	startValue := binary.BigEndian.Uint32(start.AsSlice())
	endValue := binary.BigEndian.Uint32(end.AsSlice())
	return int(endValue - startValue)
}

func neighborStatus(states []string) Status {
	for _, state := range states {
		switch strings.ToUpper(state) {
		case "REACHABLE", "DELAY", "PROBE", "PERMANENT", "NOARP":
			return StatusOnline
		case "STALE":
			return StatusRecent
		}
	}
	return StatusOffline
}

func revision(snapshot Snapshot) string {
	payload := struct {
		Healthy  bool
		Summary  Summary
		Leases   []LeaseView
		Warnings []string
	}{
		Healthy:  snapshot.Healthy,
		Summary:  snapshot.Summary,
		Leases:   snapshot.Leases,
		Warnings: snapshot.Warnings,
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func modelLeases(views []LeaseView) []lease.Lease {
	result := make([]lease.Lease, 0, len(views))
	for _, view := range views {
		ip, err := netip.ParseAddr(view.IP)
		if err != nil {
			continue
		}
		result = append(result, lease.Lease{
			ExpiresAt: view.ExpiresAt,
			Infinite:  view.Infinite,
			MAC:       view.MAC,
			IP:        ip,
			Hostname:  view.Hostname,
			ClientID:  view.ClientID,
		})
	}
	return result
}
