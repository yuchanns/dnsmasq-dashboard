package neighbor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Entry struct {
	IP     string
	MAC    string
	States []string
}

type Reader interface {
	Read(context.Context) ([]Entry, error)
}

type CommandReader struct {
	Command   string
	Interface string
}

func (reader CommandReader) Read(ctx context.Context) ([]Entry, error) {
	output, err := exec.CommandContext(ctx, reader.Command, "-j", "neighbor", "show", "dev", reader.Interface).Output()
	if err != nil {
		return nil, fmt.Errorf("read neighbor table: %w", err)
	}
	return Parse(output)
}

type FileReader struct {
	Path string
}

func (reader FileReader) Read(_ context.Context) ([]Entry, error) {
	data, err := os.ReadFile(reader.Path)
	if err != nil {
		return nil, fmt.Errorf("read neighbor fixture: %w", err)
	}
	return Parse(data)
}

type jsonEntry struct {
	Destination string   `json:"dst"`
	MAC         string   `json:"lladdr"`
	State       []string `json:"state"`
}

func Parse(data []byte) ([]Entry, error) {
	var raw []jsonEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode neighbor table: %w", err)
	}

	entries := make([]Entry, 0, len(raw))
	for _, item := range raw {
		if item.Destination == "" {
			continue
		}
		states := make([]string, 0, len(item.State))
		for _, state := range item.State {
			states = append(states, strings.ToUpper(state))
		}
		entries = append(entries, Entry{
			IP:     item.Destination,
			MAC:    strings.ToLower(item.MAC),
			States: states,
		})
	}
	return entries, nil
}
