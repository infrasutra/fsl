package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Server implements an LSP server for FSL
type Server struct {
	reader    *bufio.Reader
	writer    io.Writer
	documents *DocumentStore
	handler   *Handler

	mu       sync.Mutex
	shutdown bool
}

// NewServer creates a new LSP server
func NewServer(reader io.Reader, writer io.Writer) *Server {
	s := &Server{
		reader:    bufio.NewReader(reader),
		writer:    writer,
		documents: NewDocumentStore(),
	}
	s.handler = NewHandler(s)
	return s
}

// Run starts the LSP server main loop
func (s *Server) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading message: %w", err)
		}

		s.handleMessage(msg)

		s.mu.Lock()
		if s.shutdown {
			s.mu.Unlock()
			return nil
		}
		s.mu.Unlock()
	}
}

// JSONRPCMessage represents a JSON-RPC 2.0 message
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) readMessage() (*JSONRPCMessage, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)

		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid content length: %w", err)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing content-length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, content)
	if err != nil {
		return nil, fmt.Errorf("error reading content: %w", err)
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, fmt.Errorf("error parsing message: %w", err)
	}

	return &msg, nil
}

func (s *Server) writeMessage(msg *JSONRPCMessage) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := s.writer.Write(content); err != nil {
		return err
	}
	return nil
}

func (s *Server) handleMessage(msg *JSONRPCMessage) {
	response := s.handler.Handle(msg)
	if response != nil {
		s.writeMessage(response)
	}
}

// SendNotification sends a notification to the client
func (s *Server) SendNotification(method string, params interface{}) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	return s.writeMessage(msg)
}

// PublishDiagnostics sends diagnostics for a document
func (s *Server) PublishDiagnostics(uri string, diagnostics []Diagnostic) error {
	return s.SendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// Shutdown marks the server for shutdown
func (s *Server) Shutdown() {
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()
}

// GetDocuments returns the document store
func (s *Server) GetDocuments() *DocumentStore {
	return s.documents
}

// RunStdio runs the server on stdin/stdout
func RunStdio() error {
	server := NewServer(os.Stdin, os.Stdout)
	return server.Run(context.Background())
}
