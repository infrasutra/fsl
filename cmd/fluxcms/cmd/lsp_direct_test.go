package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunSocketServer_CanceledContextReturnsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() {
		done <- runSocketServer(ctx, "127.0.0.1:0")
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("runSocketServer did not return after context cancellation")
	}
}

func TestRunSocketServer_InvalidAddressErrors(t *testing.T) {
	err := runSocketServer(context.Background(), "not-a-valid-address")
	assert.ErrorContains(t, err, "failed to listen")
}
