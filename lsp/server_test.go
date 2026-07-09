package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encodeMessage wraps a JSON payload in the LSP Content-Length framing.
func encodeMessage(t *testing.T, payload string) string {
	t.Helper()
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)
}

func TestNewServer(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(strings.NewReader(""), &out)
	require.NotNil(t, server)
	require.NotNil(t, server.GetDocuments())
	require.NotNil(t, server.handler)
}

func TestServer_ReadMessage(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
		server := NewServer(strings.NewReader(encodeMessage(t, body)), &bytes.Buffer{})

		msg, err := server.readMessage()
		require.NoError(t, err)
		assert.Equal(t, "initialize", msg.Method)
	})

	t.Run("EOF on empty input", func(t *testing.T) {
		server := NewServer(strings.NewReader(""), &bytes.Buffer{})
		_, err := server.readMessage()
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("missing content-length header errors", func(t *testing.T) {
		server := NewServer(strings.NewReader("\r\n"), &bytes.Buffer{})
		_, err := server.readMessage()
		assert.ErrorContains(t, err, "missing content-length")
	})

	t.Run("invalid content-length value errors", func(t *testing.T) {
		server := NewServer(strings.NewReader("Content-Length: notanumber\r\n\r\n"), &bytes.Buffer{})
		_, err := server.readMessage()
		assert.ErrorContains(t, err, "invalid content length")
	})

	t.Run("truncated body errors", func(t *testing.T) {
		server := NewServer(strings.NewReader("Content-Length: 100\r\n\r\n{}"), &bytes.Buffer{})
		_, err := server.readMessage()
		assert.ErrorContains(t, err, "error reading content")
	})

	t.Run("invalid json body errors", func(t *testing.T) {
		body := `not json`
		server := NewServer(strings.NewReader(encodeMessage(t, body)), &bytes.Buffer{})
		_, err := server.readMessage()
		assert.ErrorContains(t, err, "error parsing message")
	})
}

func TestServer_WriteMessage(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(strings.NewReader(""), &out)

	id := json.RawMessage(`1`)
	err := server.writeMessage(&JSONRPCMessage{JSONRPC: "2.0", ID: &id, Result: json.RawMessage(`{"ok":true}`)})
	require.NoError(t, err)

	written := out.String()
	assert.Contains(t, written, "Content-Length:")
	assert.Contains(t, written, `"result":{"ok":true}`)
}

func TestServer_HandleMessage_RequestWritesResponse(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(strings.NewReader(""), &out)

	id := json.RawMessage(`7`)
	server.handleMessage(&JSONRPCMessage{JSONRPC: "2.0", ID: &id, Method: "initialize"})

	assert.Contains(t, out.String(), `"serverInfo"`)
}

func TestServer_HandleMessage_NotificationWritesNothing(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(strings.NewReader(""), &out)

	server.handleMessage(&JSONRPCMessage{JSONRPC: "2.0", Method: "initialized"})

	assert.Empty(t, out.String())
}

func TestServer_SendNotificationAndPublishDiagnostics(t *testing.T) {
	var out bytes.Buffer
	server := NewServer(strings.NewReader(""), &out)

	require.NoError(t, server.SendNotification("window/logMessage", map[string]string{"message": "hi"}))
	assert.Contains(t, out.String(), "window/logMessage")

	out.Reset()
	require.NoError(t, server.PublishDiagnostics("file:///a.fsl", []Diagnostic{{Message: "boom"}}))
	assert.Contains(t, out.String(), "textDocument/publishDiagnostics")
	assert.Contains(t, out.String(), "boom")
}

func TestServer_Shutdown(t *testing.T) {
	server := NewServer(strings.NewReader(""), &bytes.Buffer{})
	assert.False(t, server.shutdown)
	server.Shutdown()
	assert.True(t, server.shutdown)
}

func TestServer_Run_ProcessesRequestsThenEOF(t *testing.T) {
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	input := encodeMessage(t, initBody)

	var out bytes.Buffer
	server := NewServer(strings.NewReader(input), &out)

	err := server.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "serverInfo")
}

func TestServer_Run_StopsOnShutdownNotification(t *testing.T) {
	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	shutdownBody := `{"jsonrpc":"2.0","id":2,"method":"shutdown"}`
	exitBody := `{"jsonrpc":"2.0","method":"exit"}`

	input := encodeMessage(t, initBody) + encodeMessage(t, shutdownBody) + encodeMessage(t, exitBody)
	var out bytes.Buffer
	server := NewServer(strings.NewReader(input), &out)

	err := server.Run(context.Background())
	require.NoError(t, err)
	assert.True(t, server.shutdown, "exit notification should mark server as shut down")
}

func TestServer_Run_ContextCancellation(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var out bytes.Buffer
	server := NewServer(pr, &out)

	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestServer_Run_ReadErrorPropagates(t *testing.T) {
	server := NewServer(strings.NewReader("Content-Length: notanumber\r\n\r\n"), &bytes.Buffer{})
	err := server.Run(context.Background())
	assert.ErrorContains(t, err, "error reading message")
}

// TestServer_Run_ConcurrentPipeRoundTrip exercises the full concurrency path: a live
// server goroutine reading/writing over one io.Pipe pair per direction, driven by a
// client that frames real requests, reads the framed responses back, and then shuts
// the server down cleanly via the standard shutdown/exit handshake.
func TestServer_Run_ConcurrentPipeRoundTrip(t *testing.T) {
	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()
	defer clientToServerW.Close()
	defer serverToClientR.Close()

	server := NewServer(clientToServerR, serverToClientW)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan error, 1)
	go func() { runDone <- server.Run(ctx) }()

	// A throwaway Server value lets the test reuse the exact framing logic under
	// test (readMessage) to decode responses coming back from the live server.
	client := &Server{reader: bufio.NewReader(serverToClientR)}

	writeFramed := func(payload string) {
		t.Helper()
		_, err := clientToServerW.Write([]byte(encodeMessage(t, payload)))
		require.NoError(t, err)
	}

	// Request 1: initialize.
	writeFramed(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	resp1, err := client.readMessage()
	require.NoError(t, err)
	require.Nil(t, resp1.Error)
	assert.Contains(t, string(resp1.Result), "serverInfo")

	// Request 2: shutdown, per the LSP spec this should succeed with a null result
	// while leaving the connection open until "exit" is received.
	writeFramed(`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`)
	resp2, err := client.readMessage()
	require.NoError(t, err)
	require.Nil(t, resp2.Error)

	// Notification: exit. No response is expected; this should make Run() return.
	writeFramed(`{"jsonrpc":"2.0","method":"exit"}`)

	select {
	case runErr := <-runDone:
		assert.NoError(t, runErr)
	case <-time.After(2 * time.Second):
		t.Fatal("server.Run did not return after exit notification")
	}
}

func TestRunStdio(t *testing.T) {
	oldStdin, oldStdout := os.Stdin, os.Stdout
	t.Cleanup(func() {
		os.Stdin, os.Stdout = oldStdin, oldStdout
	})

	inR, inW, err := os.Pipe()
	require.NoError(t, err)
	outR, outW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdin = inR
	os.Stdout = outW

	// Closing the write end immediately yields EOF, so RunStdio returns nil promptly.
	require.NoError(t, inW.Close())

	done := make(chan error, 1)
	go func() { done <- RunStdio() }()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("RunStdio did not return after stdin EOF")
	}

	require.NoError(t, outW.Close())
	_, _ = io.ReadAll(outR)
}
