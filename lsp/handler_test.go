package lsp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler() (*Handler, *Server) {
	server := &Server{documents: NewDocumentStore(), writer: &bytes.Buffer{}}
	return NewHandler(server), server
}

func TestHandler_Handle_Notification_NoResponse(t *testing.T) {
	handler, _ := newTestHandler()

	msg := &JSONRPCMessage{JSONRPC: "2.0", Method: "initialized"}
	resp := handler.Handle(msg)
	assert.Nil(t, resp)
	assert.True(t, handler.initialized)
}

func TestHandler_Handle_Request_Success(t *testing.T) {
	handler, _ := newTestHandler()

	id := json.RawMessage(`42`)
	msg := &JSONRPCMessage{JSONRPC: "2.0", ID: &id, Method: "initialize"}
	resp := handler.Handle(msg)
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Contains(t, string(resp.Result), "serverInfo")
}

func TestHandler_Handle_Request_Error(t *testing.T) {
	handler, _ := newTestHandler()

	id := json.RawMessage(`1`)
	// Malformed params trigger a json.Unmarshal error inside handleHover.
	msg := &JSONRPCMessage{JSONRPC: "2.0", ID: &id, Method: "textDocument/hover", Params: json.RawMessage(`not json`)}
	resp := handler.Handle(msg)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32603, resp.Error.Code)
}

func TestHandler_HandleRequest_UnknownMethod(t *testing.T) {
	handler, _ := newTestHandler()
	result, err := handler.handleRequest("workspace/unknownMethod", nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestHandler_HandleShutdown(t *testing.T) {
	handler, _ := newTestHandler()
	result, err := handler.handleShutdown()
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestHandler_LifecycleNotifications(t *testing.T) {
	handler, server := newTestHandler()

	openParams, _ := json.Marshal(DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: "file:///a.fsl", Text: "type A { x: String! }", Version: 1},
	})
	handler.handleNotification("textDocument/didOpen", openParams)
	require.NotNil(t, server.GetDocuments().Get("file:///a.fsl"))

	changeParams, _ := json.Marshal(DidChangeTextDocumentParams{
		TextDocument:   VersionedTextDocumentIdentifier{TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///a.fsl"}, Version: 2},
		ContentChanges: []TextDocumentContentChangeEvent{{Text: "type A { y: Int }"}},
	})
	handler.handleNotification("textDocument/didChange", changeParams)
	assert.Equal(t, "type A { y: Int }", server.GetDocuments().Get("file:///a.fsl").Content)

	saveParams, _ := json.Marshal(DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
	})
	handler.handleNotification("textDocument/didSave", saveParams)

	closeParams, _ := json.Marshal(DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
	})
	handler.handleNotification("textDocument/didClose", closeParams)
	assert.Nil(t, server.GetDocuments().Get("file:///a.fsl"))
}

func TestHandler_LifecycleNotifications_MalformedParamsIgnored(t *testing.T) {
	var out bytes.Buffer
	server := &Server{documents: NewDocumentStore(), writer: &out}
	handler := NewHandler(server)

	// Seed a real document so didChange/didClose have existing state to (not) mutate.
	server.GetDocuments().Open("file:///a.fsl", "type A { x: String! }", 1)

	handler.handleNotification("textDocument/didOpen", json.RawMessage(`not json`))
	assert.Len(t, server.GetDocuments().All(), 1, "malformed didOpen must not create a document")

	handler.handleNotification("textDocument/didChange", json.RawMessage(`not json`))
	doc := server.GetDocuments().Get("file:///a.fsl")
	require.NotNil(t, doc)
	assert.Equal(t, "type A { x: String! }", doc.Content, "malformed didChange must not mutate the document")
	assert.Equal(t, 1, doc.Version, "malformed didChange must not bump the version")

	handler.handleNotification("textDocument/didClose", json.RawMessage(`not json`))
	assert.NotNil(t, server.GetDocuments().Get("file:///a.fsl"), "malformed didClose must not remove the document")

	handler.handleNotification("textDocument/didSave", json.RawMessage(`not json`))
	assert.Empty(t, out.Bytes(), "malformed didSave must not trigger a publishDiagnostics notification")
}

func TestHandler_HandleDidChange_NoContentChangesIsNoop(t *testing.T) {
	handler, server := newTestHandler()
	server.GetDocuments().Open("file:///a.fsl", "type A { x: String! }", 1)

	params, _ := json.Marshal(DidChangeTextDocumentParams{
		TextDocument:   VersionedTextDocumentIdentifier{TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///a.fsl"}, Version: 2},
		ContentChanges: nil,
	})
	handler.handleNotification("textDocument/didChange", params)
	assert.Equal(t, "type A { x: String! }", server.GetDocuments().Get("file:///a.fsl").Content)
}

func TestHandler_HandleCompletion(t *testing.T) {
	handler, server := newTestHandler()

	t.Run("missing document returns empty list", func(t *testing.T) {
		params, _ := json.Marshal(CompletionParams{
			TextDocumentPositionParams: TextDocumentPositionParams{
				TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"},
			},
		})
		result, err := handler.handleCompletion(params)
		require.NoError(t, err)
		list, ok := result.(*CompletionList)
		require.True(t, ok)
		assert.Empty(t, list.Items)
	})

	t.Run("open document returns completions", func(t *testing.T) {
		server.GetDocuments().Open("file:///a.fsl", "type Post {\n  title: \n}", 1)
		params, _ := json.Marshal(CompletionParams{
			TextDocumentPositionParams: TextDocumentPositionParams{
				TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
				Position:     Position{Line: 1, Character: 9},
			},
		})
		result, err := handler.handleCompletion(params)
		require.NoError(t, err)
		list, ok := result.(*CompletionList)
		require.True(t, ok)
		assert.NotEmpty(t, list.Items)
	})

	t.Run("invalid params errors", func(t *testing.T) {
		_, err := handler.handleCompletion(json.RawMessage(`not json`))
		assert.Error(t, err)
	})
}

func TestHandler_HandleHover(t *testing.T) {
	handler, server := newTestHandler()
	server.GetDocuments().Open("file:///a.fsl", "type Post { title: String! }", 1)

	t.Run("missing document returns nil", func(t *testing.T) {
		params, _ := json.Marshal(TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}})
		result, err := handler.handleHover(params)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("open document returns hover", func(t *testing.T) {
		// Character 20 lands inside "String", a builtin type with known documentation.
		params, _ := json.Marshal(TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
			Position:     Position{Line: 0, Character: 20},
		})
		result, err := handler.handleHover(params)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandler_HandleDefinition(t *testing.T) {
	handler, server := newTestHandler()
	server.GetDocuments().Open("file:///a.fsl", "type Post {\n  author: Post\n}", 1)

	t.Run("missing document returns nil", func(t *testing.T) {
		params, _ := json.Marshal(TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}})
		result, err := handler.handleDefinition(params)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("open document resolves definition", func(t *testing.T) {
		params, _ := json.Marshal(TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
			Position:     Position{Line: 0, Character: 6},
		})
		result, err := handler.handleDefinition(params)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandler_HandleDocumentSymbol(t *testing.T) {
	handler, server := newTestHandler()

	t.Run("missing document returns empty slice", func(t *testing.T) {
		params, _ := json.Marshal(DocumentSymbolParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}})
		result, err := handler.handleDocumentSymbol(params)
		require.NoError(t, err)
		symbols, ok := result.([]DocumentSymbol)
		require.True(t, ok)
		assert.Empty(t, symbols)
	})

	t.Run("open document returns symbols", func(t *testing.T) {
		server.GetDocuments().Open("file:///a.fsl", "type Post { title: String! }", 1)
		params, _ := json.Marshal(DocumentSymbolParams{TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"}})
		result, err := handler.handleDocumentSymbol(params)
		require.NoError(t, err)
		symbols, ok := result.([]DocumentSymbol)
		require.True(t, ok)
		assert.NotEmpty(t, symbols)
	})
}

func TestHandler_HandleReferences(t *testing.T) {
	handler, server := newTestHandler()

	t.Run("missing document returns empty slice", func(t *testing.T) {
		params, _ := json.Marshal(ReferenceParams{
			TextDocumentPositionParams: TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}},
		})
		result, err := handler.handleReferences(params)
		require.NoError(t, err)
		locs, ok := result.([]Location)
		require.True(t, ok)
		assert.Empty(t, locs)
	})

	t.Run("open document returns references", func(t *testing.T) {
		server.GetDocuments().Open("file:///a.fsl", "type Post {\n  author: Post\n}", 1)
		params, _ := json.Marshal(ReferenceParams{
			TextDocumentPositionParams: TextDocumentPositionParams{
				TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
				Position:     Position{Line: 0, Character: 6},
			},
			Context: ReferenceContext{IncludeDeclaration: true},
		})
		result, err := handler.handleReferences(params)
		require.NoError(t, err)
		locs, ok := result.([]Location)
		require.True(t, ok)
		assert.NotEmpty(t, locs)
	})
}

func TestHandler_HandlePrepareRename(t *testing.T) {
	handler, server := newTestHandler()
	server.GetDocuments().Open("file:///a.fsl", "type Post { title: String! }", 1)

	t.Run("missing document returns nil", func(t *testing.T) {
		params, _ := json.Marshal(TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}})
		result, err := handler.handlePrepareRename(params)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("open document allows renaming type", func(t *testing.T) {
		params, _ := json.Marshal(TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
			Position:     Position{Line: 0, Character: 6},
		})
		result, err := handler.handlePrepareRename(params)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandler_HandleRename(t *testing.T) {
	handler, server := newTestHandler()
	server.GetDocuments().Open("file:///a.fsl", "type Post { title: String! }", 1)

	t.Run("missing document returns nil", func(t *testing.T) {
		params, _ := json.Marshal(RenameParams{
			TextDocumentPositionParams: TextDocumentPositionParams{TextDocument: TextDocumentIdentifier{URI: "file:///missing.fsl"}},
			NewName:                    "Article",
		})
		result, err := handler.handleRename(params)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("open document renames type", func(t *testing.T) {
		params, _ := json.Marshal(RenameParams{
			TextDocumentPositionParams: TextDocumentPositionParams{
				TextDocument: TextDocumentIdentifier{URI: "file:///a.fsl"},
				Position:     Position{Line: 0, Character: 6},
			},
			NewName: "Article",
		})
		result, err := handler.handleRename(params)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestHandler_PublishDiagnostics_MissingDocumentNoop(t *testing.T) {
	var out bytes.Buffer
	server := &Server{documents: NewDocumentStore(), writer: &out}
	handler := NewHandler(server)

	handler.publishDiagnostics("file:///missing.fsl")

	assert.Empty(t, out.Bytes(), "publishDiagnostics must not write a notification for an unknown document")
}
