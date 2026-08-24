package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/charmbracelet/crush/internal/config"
)

// OnDemandAuthRequiredError reports that a cold MCP profile needs its
// one-time OAuth flow before the requested turn can be retried.
type OnDemandAuthRequiredError struct {
	Name string
}

func (e *OnDemandAuthRequiredError) Error() string {
	return fmt.Sprintf("on-demand MCP %q requires authentication; complete the OAuth dialog and retry the prompt", e.Name)
}

type onDemandEntry struct {
	refs  int
	ready chan struct{}
	err   error
}

var onDemandLeases = struct {
	sync.Mutex
	entries map[string]*onDemandEntry
}{entries: make(map[string]*onDemandEntry)}

var (
	startOnDemandServer = initializeOnDemandServer
	stopOnDemandServer  = disableOnDemandServer
)

// AcquireOnDemand starts the named cold MCP profiles and returns an
// idempotent release function. Concurrent turns share one connection per
// profile; the final release tears it down.
func AcquireOnDemand(ctx context.Context, cfg *config.ConfigStore, names []string) (func(), error) {
	names = slices.Compact(slices.Sorted(slices.Values(names)))
	if len(names) == 0 {
		return func() {}, nil
	}

	for _, name := range names {
		m, exists := cfg.Config().MCP[name]
		if !exists {
			return nil, fmt.Errorf("on-demand MCP %q is not configured", name)
		}
		if m.Disabled {
			return nil, fmt.Errorf("on-demand MCP %q is disabled", name)
		}
		if !m.OnDemand {
			return nil, fmt.Errorf("MCP %q is not configured for on-demand use", name)
		}
	}

	acquired := make([]string, 0, len(names))
	for _, name := range names {
		acquired = append(acquired, name)
		if err := acquireOnDemandServer(ctx, cfg, name); err != nil {
			releaseOnDemandServers(cfg, acquired)
			return nil, err
		}
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			releaseOnDemandServers(cfg, acquired)
		})
	}, nil
}

func acquireOnDemandServer(ctx context.Context, cfg *config.ConfigStore, name string) error {
	onDemandLeases.Lock()
	entry, exists := onDemandLeases.entries[name]
	if exists {
		entry.refs++
		ready := entry.ready
		onDemandLeases.Unlock()
		select {
		case <-ready:
			return entry.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	entry = &onDemandEntry{refs: 1, ready: make(chan struct{})}
	onDemandLeases.entries[name] = entry
	onDemandLeases.Unlock()

	err := startOnDemandServer(ctx, cfg, name)
	onDemandLeases.Lock()
	entry.err = err
	close(entry.ready)
	onDemandLeases.Unlock()
	return err
}

func releaseOnDemandServers(cfg *config.ConfigStore, names []string) {
	for i := len(names) - 1; i >= 0; i-- {
		releaseOnDemandServer(cfg, names[i])
	}
}

func releaseOnDemandServer(cfg *config.ConfigStore, name string) {
	onDemandLeases.Lock()
	entry, exists := onDemandLeases.entries[name]
	if !exists {
		onDemandLeases.Unlock()
		return
	}
	entry.refs--
	if entry.refs > 0 {
		onDemandLeases.Unlock()
		return
	}
	delete(onDemandLeases.entries, name)
	onDemandLeases.Unlock()

	if err := stopOnDemandServer(cfg, name); err != nil {
		slog.Warn("Failed to stop on-demand MCP server", "name", name, "error", err)
	}
}

func initializeOnDemandServer(ctx context.Context, cfg *config.ConfigStore, name string) error {
	m := cfg.Config().MCP[name]
	if info, exists := states.Get(name); exists && info.State == StateConnected && mcpConfigEqual(info.Config, m) {
		return nil
	}

	teardown(name)
	if err := initClient(ctx, cfg, name, m, currentGen(name), cfg.Resolver()); err != nil {
		return fmt.Errorf("failed to start on-demand MCP %q: %w", name, err)
	}
	info, exists := states.Get(name)
	if !exists {
		return fmt.Errorf("on-demand MCP %q did not publish a state", name)
	}
	switch info.State {
	case StateConnected:
		return nil
	case StateNeedsAuth:
		return &OnDemandAuthRequiredError{Name: name}
	case StateError:
		return fmt.Errorf("failed to start on-demand MCP %q: %w", name, info.Error)
	default:
		return fmt.Errorf("on-demand MCP %q did not connect (state: %s)", name, info.State)
	}
}

func disableOnDemandServer(cfg *config.ConfigStore, name string) error {
	if info, exists := states.Get(name); exists && info.State == StateNeedsAuth {
		// Keep NeedsAuth visible long enough for the TUI to offer its OAuth
		// dialog. AuthenticateMCP returns this profile to cold state after the
		// token has been persisted.
		return nil
	}
	return DisableSingle(cfg, name)
}

func onDemandInUse(name string) bool {
	onDemandLeases.Lock()
	defer onDemandLeases.Unlock()
	entry, exists := onDemandLeases.entries[name]
	return exists && entry.refs > 0
}

func onDemandActiveSnapshot() map[string]bool {
	onDemandLeases.Lock()
	defer onDemandLeases.Unlock()
	active := make(map[string]bool, len(onDemandLeases.entries))
	for name, entry := range onDemandLeases.entries {
		active[name] = entry.refs > 0
	}
	return active
}
