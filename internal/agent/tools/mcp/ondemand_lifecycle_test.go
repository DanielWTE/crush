package mcp

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/config"
	"github.com/stretchr/testify/require"
)

func setupOnDemandLifecycleTest(t *testing.T) {
	t.Helper()
	oldStart := startOnDemandServer
	oldStop := stopOnDemandServer
	onDemandLeases.Lock()
	onDemandLeases.entries = make(map[string]*onDemandEntry)
	onDemandLeases.Unlock()
	t.Cleanup(func() {
		startOnDemandServer = oldStart
		stopOnDemandServer = oldStop
		onDemandLeases.Lock()
		onDemandLeases.entries = make(map[string]*onDemandEntry)
		onDemandLeases.Unlock()
	})
}

func TestAcquireOnDemandSharesConnectionUntilFinalRelease(t *testing.T) {
	setupOnDemandLifecycleTest(t)
	store := config.NewTestStore(&config.Config{MCP: config.MCPs{
		"sistrix": {OnDemand: true},
	}})

	started := make(chan struct{})
	allowStart := make(chan struct{})
	var startCount atomic.Int32
	var stopCount atomic.Int32
	startOnDemandServer = func(context.Context, *config.ConfigStore, string) error {
		if startCount.Add(1) == 1 {
			close(started)
		}
		<-allowStart
		return nil
	}
	stopOnDemandServer = func(*config.ConfigStore, string) error {
		stopCount.Add(1)
		return nil
	}

	type result struct {
		release func()
		err     error
	}
	results := make(chan result, 2)
	go func() {
		release, err := AcquireOnDemand(t.Context(), store, []string{"sistrix"})
		results <- result{release: release, err: err}
	}()
	<-started
	go func() {
		release, err := AcquireOnDemand(t.Context(), store, []string{"sistrix"})
		results <- result{release: release, err: err}
	}()

	require.Eventually(t, func() bool {
		onDemandLeases.Lock()
		defer onDemandLeases.Unlock()
		return onDemandLeases.entries["sistrix"].refs == 2
	}, time.Second, time.Millisecond)
	close(allowStart)

	first := <-results
	second := <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, int32(1), startCount.Load())

	first.release()
	require.Equal(t, int32(0), stopCount.Load())
	second.release()
	require.Equal(t, int32(1), stopCount.Load())
	second.release()
	require.Equal(t, int32(1), stopCount.Load(), "release must be idempotent")
}

func TestAcquireOnDemandReleasesFailedStart(t *testing.T) {
	setupOnDemandLifecycleTest(t)
	store := config.NewTestStore(&config.Config{MCP: config.MCPs{
		"rybbit": {OnDemand: true},
	}})

	wantErr := errors.New("cannot connect")
	var stopCount atomic.Int32
	startOnDemandServer = func(context.Context, *config.ConfigStore, string) error {
		return wantErr
	}
	stopOnDemandServer = func(*config.ConfigStore, string) error {
		stopCount.Add(1)
		return nil
	}

	release, err := AcquireOnDemand(t.Context(), store, []string{"rybbit"})
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, release)
	require.Equal(t, int32(1), stopCount.Load())
}
