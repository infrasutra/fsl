package lsp

// LSP types based on the Language Server Protocol specification

// Position in a text document
type Position struct {
	Line      int `json:"line"`      // 0-indexed
	Character int `json:"character"` // 0-indexed
}

// Range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem is an item to transfer a text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a document
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version int `json:"version"`
}

// TextDocumentContentChangeEvent describes content changes
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// Diagnostic represents a diagnostic (error, warning, etc.)
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               interface{}                    `json:"code,omitempty"`
	CodeDescription    *CodeDescription               `json:"codeDescription,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

// DiagnosticSeverity represents the severity of a diagnostic
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// DiagnosticTag represents additional metadata about diagnostics
type DiagnosticTag int

const (
	TagUnnecessary DiagnosticTag = 1
	TagDeprecated  DiagnosticTag = 2
)

// CodeDescription for providing a URI for more information
type CodeDescription struct {
	Href string `json:"href"`
}

// DiagnosticRelatedInformation represents related diagnostic info
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// CompletionItem represents a completion suggestion
type CompletionItem struct {
	Label               string                      `json:"label"`
	LabelDetails        *CompletionItemLabelDetails `json:"labelDetails,omitempty"`
	Kind                CompletionItemKind          `json:"kind,omitempty"`
	Tags                []CompletionItemTag         `json:"tags,omitempty"`
	Detail              string                      `json:"detail,omitempty"`
	Documentation       interface{}                 `json:"documentation,omitempty"`
	Deprecated          bool                        `json:"deprecated,omitempty"`
	Preselect           bool                        `json:"preselect,omitempty"`
	SortText            string                      `json:"sortText,omitempty"`
	FilterText          string                      `json:"filterText,omitempty"`
	InsertText          string                      `json:"insertText,omitempty"`
	InsertTextFormat    InsertTextFormat            `json:"insertTextFormat,omitempty"`
	InsertTextMode      InsertTextMode              `json:"insertTextMode,omitempty"`
	TextEdit            *TextEdit                   `json:"textEdit,omitempty"`
	AdditionalTextEdits []TextEdit                  `json:"additionalTextEdits,omitempty"`
	CommitCharacters    []string                    `json:"commitCharacters,omitempty"`
	Command             *Command                    `json:"command,omitempty"`
	Data                interface{}                 `json:"data,omitempty"`
}

// CompletionItemLabelDetails provides extra detail for a completion item
type CompletionItemLabelDetails struct {
	Detail      string `json:"detail,omitempty"`
	Description string `json:"description,omitempty"`
}

// CompletionItemTag represents additional info about a completion item
type CompletionItemTag int

const (
	CompletionTagDeprecated CompletionItemTag = 1
)

// CompletionItemKind represents the kind of a completion item
type CompletionItemKind int

const (
	CompletionKindText          CompletionItemKind = 1
	CompletionKindMethod        CompletionItemKind = 2
	CompletionKindFunction      CompletionItemKind = 3
	CompletionKindConstructor   CompletionItemKind = 4
	CompletionKindField         CompletionItemKind = 5
	CompletionKindVariable      CompletionItemKind = 6
	CompletionKindClass         CompletionItemKind = 7
	CompletionKindInterface     CompletionItemKind = 8
	CompletionKindModule        CompletionItemKind = 9
	CompletionKindProperty      CompletionItemKind = 10
	CompletionKindUnit          CompletionItemKind = 11
	CompletionKindValue         CompletionItemKind = 12
	CompletionKindEnum          CompletionItemKind = 13
	CompletionKindKeyword       CompletionItemKind = 14
	CompletionKindSnippet       CompletionItemKind = 15
	CompletionKindColor         CompletionItemKind = 16
	CompletionKindFile          CompletionItemKind = 17
	CompletionKindReference     CompletionItemKind = 18
	CompletionKindFolder        CompletionItemKind = 19
	CompletionKindEnumMember    CompletionItemKind = 20
	CompletionKindConstant      CompletionItemKind = 21
	CompletionKindStruct        CompletionItemKind = 22
	CompletionKindEvent         CompletionItemKind = 23
	CompletionKindOperator      CompletionItemKind = 24
	CompletionKindTypeParameter CompletionItemKind = 25
)

// InsertTextFormat represents the format of insert text
type InsertTextFormat int

const (
	InsertTextFormatPlainText InsertTextFormat = 1
	InsertTextFormatSnippet   InsertTextFormat = 2
)

// InsertTextMode represents how to insert the text
type InsertTextMode int

const (
	InsertTextModeAsIs              InsertTextMode = 1
	InsertTextModeAdjustIndentation InsertTextMode = 2
)

// TextEdit represents a text edit
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Command represents a command
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}

// CompletionList represents a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// Hover represents hover information
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent represents a marked up string value
type MarkupContent struct {
	Kind  MarkupKind `json:"kind"`
	Value string     `json:"value"`
}

// MarkupKind represents the type of markup
type MarkupKind string

const (
	MarkupKindPlainText MarkupKind = "plaintext"
	MarkupKindMarkdown  MarkupKind = "markdown"
)

// SymbolKind represents the kind of a symbol
type SymbolKind int

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

// DocumentSymbol represents a symbol in a document
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Tags           []SymbolTag      `json:"tags,omitempty"`
	Deprecated     bool             `json:"deprecated,omitempty"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolTag represents additional info about a symbol
type SymbolTag int

const (
	SymbolTagDeprecated SymbolTag = 1
)

// SymbolInformation represents information about a symbol
type SymbolInformation struct {
	Name          string      `json:"name"`
	Kind          SymbolKind  `json:"kind"`
	Tags          []SymbolTag `json:"tags,omitempty"`
	Deprecated    bool        `json:"deprecated,omitempty"`
	Location      Location    `json:"location"`
	ContainerName string      `json:"containerName,omitempty"`
}

// InitializeParams represents initialization parameters
type InitializeParams struct {
	ProcessID             *int               `json:"processId"`
	ClientInfo            *ClientInfo        `json:"clientInfo,omitempty"`
	Locale                string             `json:"locale,omitempty"`
	RootPath              *string            `json:"rootPath,omitempty"`
	RootURI               *string            `json:"rootUri"`
	InitializationOptions interface{}        `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 string             `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
	General      *GeneralClientCapabilities      `json:"general,omitempty"`
	Experimental interface{}                     `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities represents workspace capabilities
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool                                      `json:"applyEdit,omitempty"`
	WorkspaceEdit          *WorkspaceEditClientCapabilities          `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration *DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  *DidChangeWatchedFilesClientCapabilities  `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 *WorkspaceSymbolClientCapabilities        `json:"symbol,omitempty"`
	Configuration          bool                                      `json:"configuration,omitempty"`
	WorkspaceFolders       bool                                      `json:"workspaceFolders,omitempty"`
}

// WorkspaceEditClientCapabilities represents workspace edit capabilities
type WorkspaceEditClientCapabilities struct {
	DocumentChanges    bool     `json:"documentChanges,omitempty"`
	ResourceOperations []string `json:"resourceOperations,omitempty"`
	FailureHandling    string   `json:"failureHandling,omitempty"`
}

// DidChangeConfigurationClientCapabilities represents configuration change capabilities
type DidChangeConfigurationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DidChangeWatchedFilesClientCapabilities represents file watch capabilities
type DidChangeWatchedFilesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WorkspaceSymbolClientCapabilities represents workspace symbol capabilities
type WorkspaceSymbolClientCapabilities struct {
	DynamicRegistration bool                          `json:"dynamicRegistration,omitempty"`
	SymbolKind          *SymbolKindClientCapabilities `json:"symbolKind,omitempty"`
}

// SymbolKindClientCapabilities represents symbol kind capabilities
type SymbolKindClientCapabilities struct {
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

// TextDocumentClientCapabilities represents text document capabilities
type TextDocumentClientCapabilities struct {
	Synchronization    *TextDocumentSyncClientCapabilities   `json:"synchronization,omitempty"`
	Completion         *CompletionClientCapabilities         `json:"completion,omitempty"`
	Hover              *HoverClientCapabilities              `json:"hover,omitempty"`
	SignatureHelp      *SignatureHelpClientCapabilities      `json:"signatureHelp,omitempty"`
	References         *ReferenceClientCapabilities          `json:"references,omitempty"`
	DocumentHighlight  *DocumentHighlightClientCapabilities  `json:"documentHighlight,omitempty"`
	DocumentSymbol     *DocumentSymbolClientCapabilities     `json:"documentSymbol,omitempty"`
	Formatting         *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`
	Definition         *DefinitionClientCapabilities         `json:"definition,omitempty"`
	CodeAction         *CodeActionClientCapabilities         `json:"codeAction,omitempty"`
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
}

// TextDocumentSyncClientCapabilities represents sync capabilities
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// CompletionClientCapabilities represents completion capabilities
type CompletionClientCapabilities struct {
	DynamicRegistration bool                                  `json:"dynamicRegistration,omitempty"`
	CompletionItem      *CompletionItemClientCapabilities     `json:"completionItem,omitempty"`
	CompletionItemKind  *CompletionItemKindClientCapabilities `json:"completionItemKind,omitempty"`
	ContextSupport      bool                                  `json:"contextSupport,omitempty"`
}

// CompletionItemClientCapabilities represents completion item capabilities
type CompletionItemClientCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
	DeprecatedSupport       bool     `json:"deprecatedSupport,omitempty"`
	PreselectSupport        bool     `json:"preselectSupport,omitempty"`
}

// CompletionItemKindClientCapabilities represents completion item kind capabilities
type CompletionItemKindClientCapabilities struct {
	ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
}

// HoverClientCapabilities represents hover capabilities
type HoverClientCapabilities struct {
	DynamicRegistration bool         `json:"dynamicRegistration,omitempty"`
	ContentFormat       []MarkupKind `json:"contentFormat,omitempty"`
}

// SignatureHelpClientCapabilities represents signature help capabilities
type SignatureHelpClientCapabilities struct {
	DynamicRegistration  bool                                    `json:"dynamicRegistration,omitempty"`
	SignatureInformation *SignatureInformationClientCapabilities `json:"signatureInformation,omitempty"`
}

// SignatureInformationClientCapabilities represents signature info capabilities
type SignatureInformationClientCapabilities struct {
	DocumentationFormat  []MarkupKind                            `json:"documentationFormat,omitempty"`
	ParameterInformation *ParameterInformationClientCapabilities `json:"parameterInformation,omitempty"`
}

// ParameterInformationClientCapabilities represents parameter info capabilities
type ParameterInformationClientCapabilities struct {
	LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
}

// ReferenceClientCapabilities represents reference capabilities
type ReferenceClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentHighlightClientCapabilities represents highlight capabilities
type DocumentHighlightClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapabilities represents document symbol capabilities
type DocumentSymbolClientCapabilities struct {
	DynamicRegistration               bool                          `json:"dynamicRegistration,omitempty"`
	SymbolKind                        *SymbolKindClientCapabilities `json:"symbolKind,omitempty"`
	HierarchicalDocumentSymbolSupport bool                          `json:"hierarchicalDocumentSymbolSupport,omitempty"`
}

// DocumentFormattingClientCapabilities represents formatting capabilities
type DocumentFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DefinitionClientCapabilities represents definition capabilities
type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// CodeActionClientCapabilities represents code action capabilities
type CodeActionClientCapabilities struct {
	DynamicRegistration      bool                      `json:"dynamicRegistration,omitempty"`
	CodeActionLiteralSupport *CodeActionLiteralSupport `json:"codeActionLiteralSupport,omitempty"`
}

// CodeActionLiteralSupport represents code action literal support
type CodeActionLiteralSupport struct {
	CodeActionKind *CodeActionKindValueSet `json:"codeActionKind,omitempty"`
}

// CodeActionKindValueSet represents supported code action kinds
type CodeActionKindValueSet struct {
	ValueSet []string `json:"valueSet,omitempty"`
}

// PublishDiagnosticsClientCapabilities represents publish diagnostics capabilities
type PublishDiagnosticsClientCapabilities struct {
	RelatedInformation     bool                  `json:"relatedInformation,omitempty"`
	TagSupport             *DiagnosticTagSupport `json:"tagSupport,omitempty"`
	VersionSupport         bool                  `json:"versionSupport,omitempty"`
	CodeDescriptionSupport bool                  `json:"codeDescriptionSupport,omitempty"`
	DataSupport            bool                  `json:"dataSupport,omitempty"`
}

// DiagnosticTagSupport represents diagnostic tag support
type DiagnosticTagSupport struct {
	ValueSet []DiagnosticTag `json:"valueSet,omitempty"`
}

// WindowClientCapabilities represents window capabilities
type WindowClientCapabilities struct {
	WorkDoneProgress bool                                  `json:"workDoneProgress,omitempty"`
	ShowMessage      *ShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`
	ShowDocument     *ShowDocumentClientCapabilities       `json:"showDocument,omitempty"`
}

// ShowMessageRequestClientCapabilities represents show message capabilities
type ShowMessageRequestClientCapabilities struct {
	MessageActionItem *MessageActionItemClientCapabilities `json:"messageActionItem,omitempty"`
}

// MessageActionItemClientCapabilities represents message action item capabilities
type MessageActionItemClientCapabilities struct {
	AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
}

// ShowDocumentClientCapabilities represents show document capabilities
type ShowDocumentClientCapabilities struct {
	Support bool `json:"support,omitempty"`
}

// GeneralClientCapabilities represents general capabilities
type GeneralClientCapabilities struct {
	StaleRequestSupport *StaleRequestSupportCapabilities      `json:"staleRequestSupport,omitempty"`
	RegularExpressions  *RegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`
	Markdown            *MarkdownClientCapabilities           `json:"markdown,omitempty"`
}

// StaleRequestSupportCapabilities represents stale request support
type StaleRequestSupportCapabilities struct {
	Cancel                 bool     `json:"cancel,omitempty"`
	RetryOnContentModified []string `json:"retryOnContentModified,omitempty"`
}

// RegularExpressionsClientCapabilities represents regex capabilities
type RegularExpressionsClientCapabilities struct {
	Engine  string `json:"engine,omitempty"`
	Version string `json:"version,omitempty"`
}

// MarkdownClientCapabilities represents markdown capabilities
type MarkdownClientCapabilities struct {
	Parser      string   `json:"parser,omitempty"`
	Version     string   `json:"version,omitempty"`
	AllowedTags []string `json:"allowedTags,omitempty"`
}

// WorkspaceFolder represents a workspace folder
type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// InitializeResult represents the result of initialization
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerCapabilities represents what the server can do
type ServerCapabilities struct {
	TextDocumentSync           interface{}                  `json:"textDocumentSync,omitempty"`
	CompletionProvider         *CompletionOptions           `json:"completionProvider,omitempty"`
	HoverProvider              interface{}                  `json:"hoverProvider,omitempty"`
	SignatureHelpProvider      *SignatureHelpOptions        `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider         interface{}                  `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider     interface{}                  `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     interface{}                  `json:"implementationProvider,omitempty"`
	ReferencesProvider         interface{}                  `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider  interface{}                  `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider     interface{}                  `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider    interface{}                  `json:"workspaceSymbolProvider,omitempty"`
	CodeActionProvider         interface{}                  `json:"codeActionProvider,omitempty"`
	DocumentFormattingProvider interface{}                  `json:"documentFormattingProvider,omitempty"`
	RenameProvider             interface{}                  `json:"renameProvider,omitempty"`
	FoldingRangeProvider       interface{}                  `json:"foldingRangeProvider,omitempty"`
	Workspace                  *ServerWorkspaceCapabilities `json:"workspace,omitempty"`
}

// ServerWorkspaceCapabilities represents workspace capabilities
type ServerWorkspaceCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
}

// WorkspaceFoldersServerCapabilities represents workspace folder capabilities
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool        `json:"supported,omitempty"`
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"`
}

// CompletionOptions represents completion server options
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	WorkDoneProgress  bool     `json:"workDoneProgress,omitempty"`
}

// SignatureHelpOptions represents signature help options
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// TextDocumentSyncOptions represents text document sync options
type TextDocumentSyncOptions struct {
	OpenClose bool                 `json:"openClose,omitempty"`
	Change    TextDocumentSyncKind `json:"change,omitempty"`
	WillSave  bool                 `json:"willSave,omitempty"`
	Save      *SaveOptions         `json:"save,omitempty"`
}

// TextDocumentSyncKind represents how documents are synced
type TextDocumentSyncKind int

const (
	SyncKindNone        TextDocumentSyncKind = 0
	SyncKindFull        TextDocumentSyncKind = 1
	SyncKindIncremental TextDocumentSyncKind = 2
)

// SaveOptions represents save options
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// PublishDiagnosticsParams represents publish diagnostics parameters
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// DidOpenTextDocumentParams represents did open parameters
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams represents did change parameters
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams represents did close parameters
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams represents did save parameters
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// TextDocumentPositionParams represents a position in a document
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams represents completion request parameters
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext represents the context of a completion request
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionTriggerKind represents how completion was triggered
type CompletionTriggerKind int

const (
	CompletionTriggerInvoked                         CompletionTriggerKind = 1
	CompletionTriggerTriggerCharacter                CompletionTriggerKind = 2
	CompletionTriggerTriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// DocumentSymbolParams represents document symbol request parameters
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// ReferenceParams represents reference request parameters
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext represents reference context
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}
