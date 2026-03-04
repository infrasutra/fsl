package lsp

import (
	"encoding/json"
)

var Version = "dev"

// Handler handles LSP requests and notifications
type Handler struct {
	server      *Server
	initialized bool
}

// NewHandler creates a new LSP handler
func NewHandler(server *Server) *Handler {
	return &Handler{
		server: server,
	}
}

// Handle processes an incoming message
func (h *Handler) Handle(msg *JSONRPCMessage) *JSONRPCMessage {
	// Handle notifications (no ID)
	if msg.ID == nil {
		h.handleNotification(msg.Method, msg.Params)
		return nil
	}

	// Handle requests
	result, err := h.handleRequest(msg.Method, msg.Params)
	if err != nil {
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	resultJSON, _ := json.Marshal(result)
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  resultJSON,
	}
}

func (h *Handler) handleRequest(method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "initialize":
		return h.handleInitialize(params)
	case "shutdown":
		return h.handleShutdown()
	case "textDocument/completion":
		return h.handleCompletion(params)
	case "textDocument/hover":
		return h.handleHover(params)
	case "textDocument/definition":
		return h.handleDefinition(params)
	case "textDocument/documentSymbol":
		return h.handleDocumentSymbol(params)
	case "textDocument/references":
		return h.handleReferences(params)
	default:
		return nil, nil
	}
}

func (h *Handler) handleNotification(method string, params json.RawMessage) {
	switch method {
	case "initialized":
		h.initialized = true
	case "exit":
		h.server.Shutdown()
	case "textDocument/didOpen":
		h.handleDidOpen(params)
	case "textDocument/didChange":
		h.handleDidChange(params)
	case "textDocument/didClose":
		h.handleDidClose(params)
	case "textDocument/didSave":
		h.handleDidSave(params)
	}
}

func (h *Handler) handleInitialize(params json.RawMessage) (*InitializeResult, error) {
	return &InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: &TextDocumentSyncOptions{
				OpenClose: true,
				Change:    SyncKindFull,
				Save: &SaveOptions{
					IncludeText: true,
				},
			},
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"@", ":", " "},
				ResolveProvider:   false,
			},
			HoverProvider:          true,
			DefinitionProvider:     true,
			DocumentSymbolProvider: true,
			ReferencesProvider:     true,
		},
		ServerInfo: &ServerInfo{
			Name:    "fsl-lsp",
			Version: Version,
		},
	}, nil
}

func (h *Handler) handleShutdown() (interface{}, error) {
	return nil, nil
}

func (h *Handler) handleDidOpen(params json.RawMessage) {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	h.server.GetDocuments().Open(p.TextDocument.URI, p.TextDocument.Text, p.TextDocument.Version)
	h.publishDiagnostics(p.TextDocument.URI)
}

func (h *Handler) handleDidChange(params json.RawMessage) {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	// Full sync - use the last content change
	if len(p.ContentChanges) > 0 {
		lastChange := p.ContentChanges[len(p.ContentChanges)-1]
		h.server.GetDocuments().Update(p.TextDocument.URI, lastChange.Text, p.TextDocument.Version)
	}

	h.publishDiagnostics(p.TextDocument.URI)
}

func (h *Handler) handleDidClose(params json.RawMessage) {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	h.server.GetDocuments().Close(p.TextDocument.URI)
}

func (h *Handler) handleDidSave(params json.RawMessage) {
	var p DidSaveTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	// Re-publish diagnostics on save
	h.publishDiagnostics(p.TextDocument.URI)
}

func (h *Handler) handleCompletion(params json.RawMessage) (interface{}, error) {
	var p CompletionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return &CompletionList{Items: []CompletionItem{}}, nil
	}

	return GetCompletions(doc, p.Position), nil
}

func (h *Handler) handleHover(params json.RawMessage) (interface{}, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return GetHover(doc, p.Position), nil
}

func (h *Handler) handleDefinition(params json.RawMessage) (interface{}, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return GetDefinition(doc, p.Position), nil
}

func (h *Handler) handleDocumentSymbol(params json.RawMessage) (interface{}, error) {
	var p DocumentSymbolParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return []DocumentSymbol{}, nil
	}

	return GetDocumentSymbols(doc), nil
}

func (h *Handler) handleReferences(params json.RawMessage) (interface{}, error) {
	var p ReferenceParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := h.server.GetDocuments().Get(p.TextDocument.URI)
	if doc == nil {
		return []Location{}, nil
	}

	return GetReferences(doc, p.Position, p.Context.IncludeDeclaration), nil
}

func (h *Handler) publishDiagnostics(uri string) {
	doc := h.server.GetDocuments().Get(uri)
	if doc == nil {
		return
	}

	diagnostics := GetDiagnostics(doc)
	h.server.PublishDiagnostics(uri, diagnostics)
}
