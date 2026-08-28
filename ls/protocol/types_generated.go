// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated from the Language Server Protocol 3.18 metamodel.
// LSP version: 3.18.0
// Source commit: 25005c80d9ec5e366c51108a4981ef264fe058e7
// Metamodel SHA-256: 50b5f057b4c9098cb90b34cecb39d24dcdc20e39f848538adceabbe78f85774c
// DO NOT EDIT.

package protocol

// Optional represents an optional non-null field.
type Optional[T any] struct {
	value *T
}

// IsZero reports whether the optional field is unset.
func (o Optional[T]) IsZero() bool {
	return o.value == nil
}

// NewOptional creates an Optional holding v.
func NewOptional[T any](v T) Optional[T] {
	return Optional[T]{value: &v}
}

// IsSet reports whether the optional field was present.
func (o Optional[T]) IsSet() bool {
	return o.value != nil
}

// Value returns the value and true if set; otherwise the zero value and false.
func (o Optional[T]) Value() (T, bool) {
	if o.value == nil {
		var zero T
		return zero, false
	}
	return *o.value, true
}

// Or returns the value if set, otherwise def.
func (o Optional[T]) Or(def T) T {
	if o.value == nil {
		return def
	}
	return *o.value
}

// Nullable represents a required nullable field.
type Nullable[T any] struct {
	value *T
	null  bool
}

// IsZero reports whether the nullable field is unset (neither value nor explicit null).
func (n Nullable[T]) IsZero() bool {
	return n.value == nil && !n.null
}

// NewNullable creates a Nullable holding v.
func NewNullable[T any](v T) Nullable[T] {
	return Nullable[T]{value: &v}
}

// NullNullable returns an explicit JSON null.
func NullNullable[T any]() Nullable[T] {
	return Nullable[T]{null: true}
}

// IsNull reports whether the value is explicit null.
func (n Nullable[T]) IsNull() bool {
	return n.null
}

// IsSet reports whether the nullable holds a concrete value (not null).
func (n Nullable[T]) IsSet() bool {
	return n.value != nil
}

// Value returns the concrete value and true if set.
func (n Nullable[T]) Value() (T, bool) {
	if n.value == nil {
		var zero T
		return zero, false
	}
	return *n.value, true
}

// OptionalNullable represents an optional nullable field.
type OptionalNullable[T any] struct {
	value *T
	null  bool
}

// IsZero reports whether the optional nullable field is unset.
func (o OptionalNullable[T]) IsZero() bool {
	return o.value == nil && !o.null
}

// NewOptionalNullable creates an OptionalNullable holding v.
func NewOptionalNullable[T any](v T) OptionalNullable[T] {
	return OptionalNullable[T]{value: &v}
}

// NullOptionalNullable returns an explicit JSON null optional.
func NullOptionalNullable[T any]() OptionalNullable[T] {
	return OptionalNullable[T]{null: true}
}

// IsSet reports whether the field was present (value or explicit null).
func (o OptionalNullable[T]) IsSet() bool {
	return o.value != nil || o.null
}

// IsNull reports whether the present value is explicit null.
func (o OptionalNullable[T]) IsNull() bool {
	return o.null
}

// Value returns the concrete value and true if set to a value.
func (o OptionalNullable[T]) Value() (T, bool) {
	if o.value == nil {
		var zero T
		return zero, false
	}
	return *o.value, true
}

// DocumentURI is the LSP DocumentUri base type.
type DocumentURI = string

// An identifier to refer to a change annotation stored with a workspace edit.
type ChangeAnnotationIdentifier = string

// Information about where a symbol is declared.
// Provides additional metadata over normal Location location declarations, including the range of
// the declaring symbol.
// Servers should prefer returning `DeclarationLink` over `Declaration` if supported
// by the client.
type DeclarationLink = LocationLink

// Information about where a symbol is defined.
// Provides additional metadata over normal Location location definitions, including the range of
// the defining symbol
type DefinitionLink = LocationLink

// A document selector is the combination of one or many document filters.
// @sample `let sel:DocumentSelector = [{ language: 'typescript' , { language: 'json', pattern: '**∕tsconfig.json' ]`;
// The use of a string as a document filter is deprecated @since 3.16.0.
// @since 3.16.0.
type DocumentSelector = []DocumentFilter

// LSP arrays.
// @since 3.17.0
// @since 3.17.0
type LSPArray = []LSPAny

// LSP object definition.
// @since 3.17.0
// @since 3.17.0
type LSPObject = map[string]any

// The glob pattern to watch relative to the base path. Glob patterns can have the following syntax:
// - `*` to match zero or more characters in a path segment
// - `?` to match on one character in a path segment
// - `**` to match any number of path segments, including none
// - `{` to group conditions (e.g. `**​/*.{ts,js` matches all TypeScript and JavaScript files)
// - `[]` to declare a range of characters to match in a path segment (e.g., `example.[0-9]` to match on `example.0`, `example.1`, …)
// - `[!...]` to negate a range of characters to match in a path segment (e.g., `example.[!0-9]` to match on `example.a`, `example.b`, but not `example.0`)
// @since 3.17.0
// @since 3.17.0
type Pattern = string

type RegularExpressionEngineKind = string

// Defines how values from a set of defaults and an individual item will be
// merged.
// @since 3.18.0
// @since 3.18.0
type ApplyKind uint32

// The value from the individual item (if provided and not `null`) will be
// used instead of the default.
const ApplyKindReplace ApplyKind = 1

// The value from the item will be merged with the default.
// The specific rules for mergeing values are defined against each field
// that supports merging.
const ApplyKindMerge ApplyKind = 2

// A set of predefined code action kinds
type CodeActionKind string

// Empty kind.
const CodeActionKindEmpty CodeActionKind = ""

// Base kind for quickfix actions: 'quickfix'
const CodeActionKindQuickFix CodeActionKind = "quickfix"

// Base kind for refactoring actions: 'refactor'
const CodeActionKindRefactor CodeActionKind = "refactor"

// Base kind for refactoring extraction actions: 'refactor.extract'
// Example extract actions:
// - Extract method
// - Extract function
// - Extract variable
// - Extract interface from class
// - ...
const CodeActionKindRefactorExtract CodeActionKind = "refactor.extract"

// Base kind for refactoring inline actions: 'refactor.inline'
// Example inline actions:
// - Inline function
// - Inline variable
// - Inline constant
// - ...
const CodeActionKindRefactorInline CodeActionKind = "refactor.inline"

// Base kind for refactoring move actions: `refactor.move`
// Example move actions:
// - Move a function to a new file
// - Move a property between classes
// - Move method to base class
// - ...
// @since 3.18.0
// @since 3.18.0
const CodeActionKindRefactorMove CodeActionKind = "refactor.move"

// Base kind for refactoring rewrite actions: 'refactor.rewrite'
// Example rewrite actions:
// - Convert JavaScript function to class
// - Add or remove parameter
// - Encapsulate field
// - Make method static
// - Move method to base class
// - ...
const CodeActionKindRefactorRewrite CodeActionKind = "refactor.rewrite"

// Base kind for source actions: `source`
// Source code actions apply to the entire file.
const CodeActionKindSource CodeActionKind = "source"

// Base kind for an organize imports source action: `source.organizeImports`
const CodeActionKindSourceOrganizeImports CodeActionKind = "source.organizeImports"

// Base kind for auto-fix source actions: `source.fixAll`.
// Fix all actions automatically fix errors that have a clear fix that do not require user input.
// They should not suppress errors or perform unsafe fixes such as generating new types or classes.
// @since 3.15.0
// @since 3.15.0
const CodeActionKindSourceFixAll CodeActionKind = "source.fixAll"

// Base kind for all code actions applying to the entire notebook's scope. CodeActionKinds using
// this should always begin with `notebook.`
// @since 3.18.0
// @since 3.18.0
const CodeActionKindNotebook CodeActionKind = "notebook"

// Code action tags are extra annotations that tweak the behavior of a code action.
// @since 3.18.0
// @since 3.18.0
type CodeActionTag uint32

// Marks the code action as LLM-generated.
const CodeActionTagLLMGenerated CodeActionTag = 1

// The reason why code actions were requested.
// @since 3.17.0
// @since 3.17.0
type CodeActionTriggerKind uint32

// Code actions were explicitly requested by the user or by an extension.
const CodeActionTriggerKindInvoked CodeActionTriggerKind = 1

// Code actions were requested automatically.
// This typically happens when current selection in a file changes, but can
// also be triggered when file content changes.
const CodeActionTriggerKindAutomatic CodeActionTriggerKind = 2

// The kind of a completion entry.
type CompletionItemKind uint32

const CompletionItemKindText CompletionItemKind = 1
const CompletionItemKindMethod CompletionItemKind = 2
const CompletionItemKindFunction CompletionItemKind = 3
const CompletionItemKindConstructor CompletionItemKind = 4
const CompletionItemKindField CompletionItemKind = 5
const CompletionItemKindVariable CompletionItemKind = 6
const CompletionItemKindClass CompletionItemKind = 7
const CompletionItemKindInterface CompletionItemKind = 8
const CompletionItemKindModule CompletionItemKind = 9
const CompletionItemKindProperty CompletionItemKind = 10
const CompletionItemKindUnit CompletionItemKind = 11
const CompletionItemKindValue CompletionItemKind = 12
const CompletionItemKindEnum CompletionItemKind = 13
const CompletionItemKindKeyword CompletionItemKind = 14
const CompletionItemKindSnippet CompletionItemKind = 15
const CompletionItemKindColor CompletionItemKind = 16
const CompletionItemKindFile CompletionItemKind = 17
const CompletionItemKindReference CompletionItemKind = 18
const CompletionItemKindFolder CompletionItemKind = 19
const CompletionItemKindEnumMember CompletionItemKind = 20
const CompletionItemKindConstant CompletionItemKind = 21
const CompletionItemKindStruct CompletionItemKind = 22
const CompletionItemKindEvent CompletionItemKind = 23
const CompletionItemKindOperator CompletionItemKind = 24
const CompletionItemKindTypeParameter CompletionItemKind = 25

// Completion item tags are extra annotations that tweak the rendering of a completion
// item.
// @since 3.15.0
// @since 3.15.0
type CompletionItemTag uint32

// Render a completion as obsolete, usually using a strike-out.
const CompletionItemTagDeprecated CompletionItemTag = 1

// How a completion was triggered
type CompletionTriggerKind uint32

// Completion was triggered by typing an identifier (24x7 code
// complete), manual invocation (e.g Ctrl+Space) or via API.
const CompletionTriggerKindInvoked CompletionTriggerKind = 1

// Completion was triggered by a trigger character specified by
// the `triggerCharacters` properties of the `CompletionRegistrationOptions`.
const CompletionTriggerKindTriggerCharacter CompletionTriggerKind = 2

// Completion was re-triggered as current completion list is incomplete
const CompletionTriggerKindTriggerForIncompleteCompletions CompletionTriggerKind = 3

// The diagnostic's severity.
type DiagnosticSeverity uint32

// Reports an error.
const DiagnosticSeverityError DiagnosticSeverity = 1

// Reports a warning.
const DiagnosticSeverityWarning DiagnosticSeverity = 2

// Reports an information.
const DiagnosticSeverityInformation DiagnosticSeverity = 3

// Reports a hint.
const DiagnosticSeverityHint DiagnosticSeverity = 4

// The diagnostic tags.
// @since 3.15.0
// @since 3.15.0
type DiagnosticTag uint32

// Unused or unnecessary code.
// Clients are allowed to render diagnostics with this tag faded out instead of having
// an error squiggle.
const DiagnosticTagUnnecessary DiagnosticTag = 1

// Deprecated or obsolete code.
// Clients are allowed to rendered diagnostics with this tag strike through.
const DiagnosticTagDeprecated DiagnosticTag = 2

// The document diagnostic report kinds.
// @since 3.17.0
// @since 3.17.0
type DocumentDiagnosticReportKind string

// A diagnostic report with a full
// set of problems.
const DocumentDiagnosticReportKindFull DocumentDiagnosticReportKind = "full"

// A report indicating that the last
// returned report is still accurate.
const DocumentDiagnosticReportKindUnchanged DocumentDiagnosticReportKind = "unchanged"

// A document highlight kind.
type DocumentHighlightKind uint32

// A textual occurrence.
const DocumentHighlightKindText DocumentHighlightKind = 1

// Read-access of a symbol, like reading a variable.
const DocumentHighlightKindRead DocumentHighlightKind = 2

// Write-access of a symbol, like writing to a variable.
const DocumentHighlightKindWrite DocumentHighlightKind = 3

// Predefined error codes.
type ErrorCodes int32

const ErrorCodesParseError ErrorCodes = -32700
const ErrorCodesInvalidRequest ErrorCodes = -32600
const ErrorCodesMethodNotFound ErrorCodes = -32601
const ErrorCodesInvalidParams ErrorCodes = -32602
const ErrorCodesInternalError ErrorCodes = -32603

// Error code indicating that a server received a notification or
// request before the server has received the `initialize` request.
const ErrorCodesServerNotInitialized ErrorCodes = -32002
const ErrorCodesUnknownErrorCode ErrorCodes = -32001

type FailureHandlingKind string

// Applying the workspace change is simply aborted if one of the changes provided
// fails. All operations executed before the failing operation stay executed.
const FailureHandlingKindAbort FailureHandlingKind = "abort"

// All operations are executed transactional. That means they either all
// succeed or no changes at all are applied to the workspace.
const FailureHandlingKindTransactional FailureHandlingKind = "transactional"

// If the workspace edit contains only textual file changes they are executed transactional.
// If resource changes (create, rename or delete file) are part of the change the failure
// handling strategy is abort.
const FailureHandlingKindTextOnlyTransactional FailureHandlingKind = "textOnlyTransactional"

// The client tries to undo the operations already executed. But there is no
// guarantee that this is succeeding.
const FailureHandlingKindUndo FailureHandlingKind = "undo"

// The file event type
type FileChangeType uint32

// The file got created.
const FileChangeTypeCreated FileChangeType = 1

// The file got changed.
const FileChangeTypeChanged FileChangeType = 2

// The file got deleted.
const FileChangeTypeDeleted FileChangeType = 3

// A pattern kind describing if a glob pattern matches a file a folder or
// both.
// @since 3.16.0
// @since 3.16.0
type FileOperationPatternKind string

// The pattern matches a file only.
const FileOperationPatternKindFile FileOperationPatternKind = "file"

// The pattern matches a folder only.
const FileOperationPatternKindFolder FileOperationPatternKind = "folder"

// A set of predefined range kinds.
type FoldingRangeKind string

// Folding range for a comment
const FoldingRangeKindComment FoldingRangeKind = "comment"

// Folding range for an import or include
const FoldingRangeKindImports FoldingRangeKind = "imports"

// Folding range for a region (e.g. `#region`)
const FoldingRangeKindRegion FoldingRangeKind = "region"

// Inlay hint kinds.
// @since 3.17.0
// @since 3.17.0
type InlayHintKind uint32

// An inlay hint that for a type annotation.
const InlayHintKindType InlayHintKind = 1

// An inlay hint that is for a parameter.
const InlayHintKindParameter InlayHintKind = 2

// Describes how an InlineCompletionItemProvider inline completion provider was triggered.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionTriggerKind uint32

// Completion was triggered explicitly by a user gesture.
const InlineCompletionTriggerKindInvoked InlineCompletionTriggerKind = 1

// Completion was triggered automatically while editing.
const InlineCompletionTriggerKindAutomatic InlineCompletionTriggerKind = 2

// Defines whether the insert text in a completion item should be interpreted as
// plain text or a snippet.
type InsertTextFormat uint32

// The primary text to be inserted is treated as a plain string.
const InsertTextFormatPlainText InsertTextFormat = 1

// The primary text to be inserted is treated as a snippet.
// A snippet can define tab stops and placeholders with `$1`, `$2`
// and `${3:foo`. `$0` defines the final tab stop, it defaults to
// the end of the snippet. Placeholders with equal identifiers are linked,
// that is typing in one will update others too.
// See also: https://microsoft.github.io/language-server-protocol/specifications/specification-current/#snippet_syntax
const InsertTextFormatSnippet InsertTextFormat = 2

// How whitespace and indentation is handled during completion
// item insertion.
// @since 3.16.0
// @since 3.16.0
type InsertTextMode uint32

// The insertion or replace strings is taken as it is. If the
// value is multi line the lines below the cursor will be
// inserted using the indentation defined in the string value.
// The client will not apply any kind of adjustments to the
// string.
const InsertTextModeAsIs InsertTextMode = 1

// The editor adjusts leading whitespace of new lines so that
// they match the indentation up to the cursor of the line for
// which the item is accepted.
// Consider a line like this: <2tabs><cursor><3tabs>foo. Accepting a
// multi line completion item is indented using 2 tabs and all
// following lines inserted will be indented using 2 tabs as well.
const InsertTextModeAdjustIndentation InsertTextMode = 2

type LSPErrorCodes int32

// A request failed but it was syntactically correct, e.g the
// method name was known and the parameters were valid. The error
// message should contain human readable information about why
// the request failed.
// @since 3.17.0
// @since 3.17.0
const LSPErrorCodesRequestFailed LSPErrorCodes = -32803

// The server cancelled the request. This error code should
// only be used for requests that explicitly support being
// server cancellable.
// @since 3.17.0
// @since 3.17.0
const LSPErrorCodesServerCancelled LSPErrorCodes = -32802

// The server detected that the content of a document got
// modified outside normal conditions. A server should
// NOT send this error code if it detects a content change
// in it unprocessed messages. The result even computed
// on an older state might still be useful for the client.
// If a client decides that a result is not of any use anymore
// the client should cancel the request.
const LSPErrorCodesContentModified LSPErrorCodes = -32801

// The client has canceled a request and a server has detected
// the cancel.
const LSPErrorCodesRequestCancelled LSPErrorCodes = -32800

// Predefined Language kinds
// @since 3.18.0
// @since 3.18.0
type LanguageKind string

const LanguageKindABAP LanguageKind = "abap"
const LanguageKindWindowsBat LanguageKind = "bat"
const LanguageKindBibTeX LanguageKind = "bibtex"
const LanguageKindClojure LanguageKind = "clojure"
const LanguageKindCoffeescript LanguageKind = "coffeescript"
const LanguageKindC LanguageKind = "c"
const LanguageKindCPP LanguageKind = "cpp"
const LanguageKindCSharp LanguageKind = "csharp"
const LanguageKindCSS LanguageKind = "css"

// @since 3.18.0
// @since 3.18.0
const LanguageKindD LanguageKind = "d"

// @since 3.18.0
// @since 3.18.0
const LanguageKindDelphi LanguageKind = "pascal"
const LanguageKindDiff LanguageKind = "diff"
const LanguageKindDart LanguageKind = "dart"
const LanguageKindDockerfile LanguageKind = "dockerfile"
const LanguageKindElixir LanguageKind = "elixir"
const LanguageKindErlang LanguageKind = "erlang"
const LanguageKindFSharp LanguageKind = "fsharp"
const LanguageKindGitCommit LanguageKind = "git-commit"
const LanguageKindGitRebase LanguageKind = "git-rebase"
const LanguageKindGo LanguageKind = "go"
const LanguageKindGroovy LanguageKind = "groovy"
const LanguageKindHandlebars LanguageKind = "handlebars"
const LanguageKindHaskell LanguageKind = "haskell"
const LanguageKindHTML LanguageKind = "html"
const LanguageKindIni LanguageKind = "ini"
const LanguageKindJava LanguageKind = "java"
const LanguageKindJavaScript LanguageKind = "javascript"
const LanguageKindJavaScriptReact LanguageKind = "javascriptreact"
const LanguageKindJSON LanguageKind = "json"
const LanguageKindLaTeX LanguageKind = "latex"
const LanguageKindLess LanguageKind = "less"
const LanguageKindLua LanguageKind = "lua"
const LanguageKindMakefile LanguageKind = "makefile"
const LanguageKindMarkdown LanguageKind = "markdown"
const LanguageKindObjectiveC LanguageKind = "objective-c"
const LanguageKindObjectiveCPP LanguageKind = "objective-cpp"

// @since 3.18.0
// @since 3.18.0
const LanguageKindPascal LanguageKind = "pascal"
const LanguageKindPerl LanguageKind = "perl"
const LanguageKindPerl6 LanguageKind = "perl6"
const LanguageKindPHP LanguageKind = "php"
const LanguageKindPlaintext LanguageKind = "plaintext"
const LanguageKindPowershell LanguageKind = "powershell"
const LanguageKindPug LanguageKind = "jade"
const LanguageKindPython LanguageKind = "python"
const LanguageKindR LanguageKind = "r"
const LanguageKindRazor LanguageKind = "razor"
const LanguageKindRuby LanguageKind = "ruby"
const LanguageKindRust LanguageKind = "rust"
const LanguageKindSCSS LanguageKind = "scss"
const LanguageKindSASS LanguageKind = "sass"
const LanguageKindScala LanguageKind = "scala"
const LanguageKindShaderLab LanguageKind = "shaderlab"
const LanguageKindShellScript LanguageKind = "shellscript"
const LanguageKindSQL LanguageKind = "sql"
const LanguageKindSwift LanguageKind = "swift"
const LanguageKindTypeScript LanguageKind = "typescript"
const LanguageKindTypeScriptReact LanguageKind = "typescriptreact"
const LanguageKindTeX LanguageKind = "tex"
const LanguageKindVisualBasic LanguageKind = "vb"
const LanguageKindXML LanguageKind = "xml"
const LanguageKindXSL LanguageKind = "xsl"
const LanguageKindYAML LanguageKind = "yaml"

// Describes the content type that a client supports in various
// result literals like `Hover`, `ParameterInfo` or `CompletionItem`.
// Please note that `MarkupKinds` must not start with a `$`. This kinds
// are reserved for internal usage.
type MarkupKind string

// Plain text is supported as a content format
const MarkupKindPlainText MarkupKind = "plaintext"

// Markdown is supported as a content format
const MarkupKindMarkdown MarkupKind = "markdown"

// The message type
type MessageType uint32

// An error message.
const MessageTypeError MessageType = 1

// A warning message.
const MessageTypeWarning MessageType = 2

// An information message.
const MessageTypeInfo MessageType = 3

// A log message.
const MessageTypeLog MessageType = 4

// A debug message.
// @since 3.18.0
// @since 3.18.0
const MessageTypeDebug MessageType = 5

// The moniker kind.
// @since 3.16.0
// @since 3.16.0
type MonikerKind string

// The moniker represent a symbol that is imported into a project
const MonikerKindImport MonikerKind = "import"

// The moniker represents a symbol that is exported from a project
const MonikerKindExport MonikerKind = "export"

// The moniker represents a symbol that is local to a project (e.g. a local
// variable of a function, a class not visible outside the project, ...)
const MonikerKindLocal MonikerKind = "local"

// A notebook cell kind.
// @since 3.17.0
// @since 3.17.0
type NotebookCellKind uint32

// A markup-cell is formatted source that is used for display.
const NotebookCellKindMarkup NotebookCellKind = 1

// A code-cell is source code.
const NotebookCellKindCode NotebookCellKind = 2

// A set of predefined position encoding kinds.
// @since 3.17.0
// @since 3.17.0
type PositionEncodingKind string

// Character offsets count UTF-8 code units (e.g. bytes).
const PositionEncodingKindUTF8 PositionEncodingKind = "utf-8"

// Character offsets count UTF-16 code units.
// This is the default and must always be supported
// by servers
const PositionEncodingKindUTF16 PositionEncodingKind = "utf-16"

// Character offsets count UTF-32 code units.
// Implementation note: these are the same as Unicode codepoints,
// so this `PositionEncodingKind` may also be used for an
// encoding-agnostic representation of character offsets.
const PositionEncodingKindUTF32 PositionEncodingKind = "utf-32"

type PrepareSupportDefaultBehavior uint32

// The client's default behavior is to select the identifier
// according the to language's syntax rule.
const PrepareSupportDefaultBehaviorIdentifier PrepareSupportDefaultBehavior = 1

type ResourceOperationKind string

// Supports creating new files and folders.
const ResourceOperationKindCreate ResourceOperationKind = "create"

// Supports renaming existing files and folders.
const ResourceOperationKindRename ResourceOperationKind = "rename"

// Supports deleting existing files and folders.
const ResourceOperationKindDelete ResourceOperationKind = "delete"

// A set of predefined token modifiers. This set is not fixed
// an clients can specify additional token types via the
// corresponding client capabilities.
// @since 3.16.0
// @since 3.16.0
type SemanticTokenModifiers string

const SemanticTokenModifiersDeclaration SemanticTokenModifiers = "declaration"
const SemanticTokenModifiersDefinition SemanticTokenModifiers = "definition"
const SemanticTokenModifiersReadonly SemanticTokenModifiers = "readonly"
const SemanticTokenModifiersStatic SemanticTokenModifiers = "static"
const SemanticTokenModifiersDeprecated SemanticTokenModifiers = "deprecated"
const SemanticTokenModifiersAbstract SemanticTokenModifiers = "abstract"
const SemanticTokenModifiersAsync SemanticTokenModifiers = "async"
const SemanticTokenModifiersModification SemanticTokenModifiers = "modification"
const SemanticTokenModifiersDocumentation SemanticTokenModifiers = "documentation"
const SemanticTokenModifiersDefaultLibrary SemanticTokenModifiers = "defaultLibrary"

// A set of predefined token types. This set is not fixed
// an clients can specify additional token types via the
// corresponding client capabilities.
// @since 3.16.0
// @since 3.16.0
type SemanticTokenTypes string

const SemanticTokenTypesNamespace SemanticTokenTypes = "namespace"

// Represents a generic type. Acts as a fallback for types which can't be mapped to
// a specific type like class or enum.
const SemanticTokenTypesType SemanticTokenTypes = "type"
const SemanticTokenTypesClass SemanticTokenTypes = "class"
const SemanticTokenTypesEnum SemanticTokenTypes = "enum"
const SemanticTokenTypesInterface SemanticTokenTypes = "interface"
const SemanticTokenTypesStruct SemanticTokenTypes = "struct"
const SemanticTokenTypesTypeParameter SemanticTokenTypes = "typeParameter"
const SemanticTokenTypesParameter SemanticTokenTypes = "parameter"
const SemanticTokenTypesVariable SemanticTokenTypes = "variable"
const SemanticTokenTypesProperty SemanticTokenTypes = "property"
const SemanticTokenTypesEnumMember SemanticTokenTypes = "enumMember"
const SemanticTokenTypesEvent SemanticTokenTypes = "event"
const SemanticTokenTypesFunction SemanticTokenTypes = "function"
const SemanticTokenTypesMethod SemanticTokenTypes = "method"
const SemanticTokenTypesMacro SemanticTokenTypes = "macro"
const SemanticTokenTypesKeyword SemanticTokenTypes = "keyword"
const SemanticTokenTypesModifier SemanticTokenTypes = "modifier"
const SemanticTokenTypesComment SemanticTokenTypes = "comment"
const SemanticTokenTypesString SemanticTokenTypes = "string"
const SemanticTokenTypesNumber SemanticTokenTypes = "number"
const SemanticTokenTypesRegexp SemanticTokenTypes = "regexp"
const SemanticTokenTypesOperator SemanticTokenTypes = "operator"

// @since 3.17.0
// @since 3.17.0
const SemanticTokenTypesDecorator SemanticTokenTypes = "decorator"

// @since 3.18.0
// @since 3.18.0
const SemanticTokenTypesLabel SemanticTokenTypes = "label"

// How a signature help was triggered.
// @since 3.15.0
// @since 3.15.0
type SignatureHelpTriggerKind uint32

// Signature help was invoked manually by the user or by a command.
const SignatureHelpTriggerKindInvoked SignatureHelpTriggerKind = 1

// Signature help was triggered by a trigger character.
const SignatureHelpTriggerKindTriggerCharacter SignatureHelpTriggerKind = 2

// Signature help was triggered by the cursor moving or by the document content changing.
const SignatureHelpTriggerKindContentChange SignatureHelpTriggerKind = 3

// A symbol kind.
type SymbolKind uint32

const SymbolKindFile SymbolKind = 1
const SymbolKindModule SymbolKind = 2
const SymbolKindNamespace SymbolKind = 3
const SymbolKindPackage SymbolKind = 4
const SymbolKindClass SymbolKind = 5
const SymbolKindMethod SymbolKind = 6
const SymbolKindProperty SymbolKind = 7
const SymbolKindField SymbolKind = 8
const SymbolKindConstructor SymbolKind = 9
const SymbolKindEnum SymbolKind = 10
const SymbolKindInterface SymbolKind = 11
const SymbolKindFunction SymbolKind = 12
const SymbolKindVariable SymbolKind = 13
const SymbolKindConstant SymbolKind = 14
const SymbolKindString SymbolKind = 15
const SymbolKindNumber SymbolKind = 16
const SymbolKindBoolean SymbolKind = 17
const SymbolKindArray SymbolKind = 18
const SymbolKindObject SymbolKind = 19
const SymbolKindKey SymbolKind = 20
const SymbolKindNull SymbolKind = 21
const SymbolKindEnumMember SymbolKind = 22
const SymbolKindStruct SymbolKind = 23
const SymbolKindEvent SymbolKind = 24
const SymbolKindOperator SymbolKind = 25
const SymbolKindTypeParameter SymbolKind = 26

// Symbol tags are extra annotations that tweak the rendering of a symbol.
// @since 3.16
// @since 3.16
type SymbolTag uint32

// Render a symbol as obsolete, usually using a strike-out.
const SymbolTagDeprecated SymbolTag = 1

// Represents reasons why a text document is saved.
type TextDocumentSaveReason uint32

// Manually triggered, e.g. by the user pressing save, by starting debugging,
// or by an API call.
const TextDocumentSaveReasonManual TextDocumentSaveReason = 1

// Automatic after a delay.
const TextDocumentSaveReasonAfterDelay TextDocumentSaveReason = 2

// When the editor lost focus.
const TextDocumentSaveReasonFocusOut TextDocumentSaveReason = 3

// Defines how the host (editor) should sync
// document changes to the language server.
type TextDocumentSyncKind uint32

// Documents should not be synced at all.
const TextDocumentSyncKindNone TextDocumentSyncKind = 0

// Documents are synced by always sending the full content
// of the document.
const TextDocumentSyncKindFull TextDocumentSyncKind = 1

// Documents are synced by sending the full content on open.
// After that only incremental updates to the document are
// send.
const TextDocumentSyncKindIncremental TextDocumentSyncKind = 2

type TokenFormat string

const TokenFormatRelative TokenFormat = "relative"

type TraceValue string

// Turn tracing off.
const TraceValueOff TraceValue = "off"

// Trace messages only.
const TraceValueMessages TraceValue = "messages"

// Verbose message tracing.
const TraceValueVerbose TraceValue = "verbose"

// Moniker uniqueness level to define scope of the moniker.
// @since 3.16.0
// @since 3.16.0
type UniquenessLevel string

// The moniker is only unique inside a document
const UniquenessLevelDocument UniquenessLevel = "document"

// The moniker is unique inside a project for which a dump got created
const UniquenessLevelProject UniquenessLevel = "project"

// The moniker is unique inside the group to which a project belongs
const UniquenessLevelGroup UniquenessLevel = "group"

// The moniker is unique inside the moniker scheme.
const UniquenessLevelScheme UniquenessLevel = "scheme"

// The moniker is globally unique
const UniquenessLevelGlobal UniquenessLevel = "global"

type WatchKind uint32

// Interested in create events.
const WatchKindCreate WatchKind = 1

// Interested in change events
const WatchKindChange WatchKind = 2

// Interested in delete events
const WatchKindDelete WatchKind = 4

// Definition is a generated union type.
type Definition struct {
	value any
	tag   int
}

// NewDefinitionLocation constructs the Location variant of Definition.
func NewDefinitionLocation(v Location) Definition {
	return Definition{value: v, tag: 0}
}

// NewDefinitionArray1 constructs the Location[] variant of Definition.
func NewDefinitionArray1(v []Location) Definition {
	return Definition{value: v, tag: 1}
}

// LSPAny is a generated union type.
type LSPAny struct {
	value any
	tag   int
}

// NewLSPAnyLSPObject constructs the LSPObject variant of LSPAny.
func NewLSPAnyLSPObject(v LSPObject) LSPAny {
	return LSPAny{value: v, tag: 0}
}

// NewLSPAnyLSPArray constructs the LSPArray variant of LSPAny.
func NewLSPAnyLSPArray(v LSPArray) LSPAny {
	return LSPAny{value: v, tag: 1}
}

// NewLSPAnyString constructs the string variant of LSPAny.
func NewLSPAnyString(v string) LSPAny {
	return LSPAny{value: v, tag: 2}
}

// NewLSPAnyInteger constructs the integer variant of LSPAny.
func NewLSPAnyInteger(v int32) LSPAny {
	return LSPAny{value: v, tag: 3}
}

// NewLSPAnyUinteger constructs the uinteger variant of LSPAny.
func NewLSPAnyUinteger(v uint32) LSPAny {
	return LSPAny{value: v, tag: 4}
}

// NewLSPAnyDecimal constructs the decimal variant of LSPAny.
func NewLSPAnyDecimal(v float64) LSPAny {
	return LSPAny{value: v, tag: 5}
}

// NewLSPAnyBoolean constructs the boolean variant of LSPAny.
func NewLSPAnyBoolean(v bool) LSPAny {
	return LSPAny{value: v, tag: 6}
}

// Declaration is a generated union type.
type Declaration struct {
	value any
	tag   int
}

// NewDeclarationLocation constructs the Location variant of Declaration.
func NewDeclarationLocation(v Location) Declaration {
	return Declaration{value: v, tag: 0}
}

// NewDeclarationArray1 constructs the Location[] variant of Declaration.
func NewDeclarationArray1(v []Location) Declaration {
	return Declaration{value: v, tag: 1}
}

// InlineValue is a generated union type.
type InlineValue struct {
	value any
	tag   int
}

// NewInlineValueInlineValueText constructs the InlineValueText variant of InlineValue.
func NewInlineValueInlineValueText(v InlineValueText) InlineValue {
	return InlineValue{value: v, tag: 0}
}

// NewInlineValueInlineValueVariableLookup constructs the InlineValueVariableLookup variant of InlineValue.
func NewInlineValueInlineValueVariableLookup(v InlineValueVariableLookup) InlineValue {
	return InlineValue{value: v, tag: 1}
}

// NewInlineValueInlineValueEvaluatableExpression constructs the InlineValueEvaluatableExpression variant of InlineValue.
func NewInlineValueInlineValueEvaluatableExpression(v InlineValueEvaluatableExpression) InlineValue {
	return InlineValue{value: v, tag: 2}
}

// DocumentDiagnosticReport is a generated union type.
type DocumentDiagnosticReport struct {
	value any
	tag   int
}

// NewDocumentDiagnosticReportRelatedFullDocumentDiagnosticReport constructs the RelatedFullDocumentDiagnosticReport variant of DocumentDiagnosticReport.
func NewDocumentDiagnosticReportRelatedFullDocumentDiagnosticReport(v RelatedFullDocumentDiagnosticReport) DocumentDiagnosticReport {
	return DocumentDiagnosticReport{value: v, tag: 0}
}

// NewDocumentDiagnosticReportRelatedUnchangedDocumentDiagnosticReport constructs the RelatedUnchangedDocumentDiagnosticReport variant of DocumentDiagnosticReport.
func NewDocumentDiagnosticReportRelatedUnchangedDocumentDiagnosticReport(v RelatedUnchangedDocumentDiagnosticReport) DocumentDiagnosticReport {
	return DocumentDiagnosticReport{value: v, tag: 1}
}

// DocumentDiagnosticReportProgress is a generated union type.
type DocumentDiagnosticReportProgress struct {
	value any
	tag   int
}

// NewDocumentDiagnosticReportProgressDocumentDiagnosticReport constructs the DocumentDiagnosticReport variant of DocumentDiagnosticReportProgress.
func NewDocumentDiagnosticReportProgressDocumentDiagnosticReport(v DocumentDiagnosticReport) DocumentDiagnosticReportProgress {
	return DocumentDiagnosticReportProgress{value: v, tag: 0}
}

// NewDocumentDiagnosticReportProgressDocumentDiagnosticReportPartialResult constructs the DocumentDiagnosticReportPartialResult variant of DocumentDiagnosticReportProgress.
func NewDocumentDiagnosticReportProgressDocumentDiagnosticReportPartialResult(v DocumentDiagnosticReportPartialResult) DocumentDiagnosticReportProgress {
	return DocumentDiagnosticReportProgress{value: v, tag: 1}
}

// PrepareRenameResult is a generated union type.
type PrepareRenameResult struct {
	value any
	tag   int
}

// NewPrepareRenameResultRange constructs the Range variant of PrepareRenameResult.
func NewPrepareRenameResultRange(v Range) PrepareRenameResult {
	return PrepareRenameResult{value: v, tag: 0}
}

// NewPrepareRenameResultPrepareRenamePlaceholder constructs the PrepareRenamePlaceholder variant of PrepareRenameResult.
func NewPrepareRenameResultPrepareRenamePlaceholder(v PrepareRenamePlaceholder) PrepareRenameResult {
	return PrepareRenameResult{value: v, tag: 1}
}

// NewPrepareRenameResultPrepareRenameDefaultBehavior constructs the PrepareRenameDefaultBehavior variant of PrepareRenameResult.
func NewPrepareRenameResultPrepareRenameDefaultBehavior(v PrepareRenameDefaultBehavior) PrepareRenameResult {
	return PrepareRenameResult{value: v, tag: 2}
}

// ProgressToken is a generated union type.
type ProgressToken struct {
	value any
	tag   int
}

// NewProgressTokenInteger constructs the integer variant of ProgressToken.
func NewProgressTokenInteger(v int32) ProgressToken {
	return ProgressToken{value: v, tag: 0}
}

// NewProgressTokenString constructs the string variant of ProgressToken.
func NewProgressTokenString(v string) ProgressToken {
	return ProgressToken{value: v, tag: 1}
}

// WorkspaceDocumentDiagnosticReport is a generated union type.
type WorkspaceDocumentDiagnosticReport struct {
	value any
	tag   int
}

// NewWorkspaceDocumentDiagnosticReportWorkspaceFullDocumentDiagnosticReport constructs the WorkspaceFullDocumentDiagnosticReport variant of WorkspaceDocumentDiagnosticReport.
func NewWorkspaceDocumentDiagnosticReportWorkspaceFullDocumentDiagnosticReport(v WorkspaceFullDocumentDiagnosticReport) WorkspaceDocumentDiagnosticReport {
	return WorkspaceDocumentDiagnosticReport{value: v, tag: 0}
}

// NewWorkspaceDocumentDiagnosticReportWorkspaceUnchangedDocumentDiagnosticReport constructs the WorkspaceUnchangedDocumentDiagnosticReport variant of WorkspaceDocumentDiagnosticReport.
func NewWorkspaceDocumentDiagnosticReportWorkspaceUnchangedDocumentDiagnosticReport(v WorkspaceUnchangedDocumentDiagnosticReport) WorkspaceDocumentDiagnosticReport {
	return WorkspaceDocumentDiagnosticReport{value: v, tag: 1}
}

// TextDocumentContentChangeEvent is a generated union type.
type TextDocumentContentChangeEvent struct {
	value any
	tag   int
}

// NewTextDocumentContentChangeEventTextDocumentContentChangePartial constructs the TextDocumentContentChangePartial variant of TextDocumentContentChangeEvent.
func NewTextDocumentContentChangeEventTextDocumentContentChangePartial(v TextDocumentContentChangePartial) TextDocumentContentChangeEvent {
	return TextDocumentContentChangeEvent{value: v, tag: 0}
}

// NewTextDocumentContentChangeEventTextDocumentContentChangeWholeDocument constructs the TextDocumentContentChangeWholeDocument variant of TextDocumentContentChangeEvent.
func NewTextDocumentContentChangeEventTextDocumentContentChangeWholeDocument(v TextDocumentContentChangeWholeDocument) TextDocumentContentChangeEvent {
	return TextDocumentContentChangeEvent{value: v, tag: 1}
}

// MarkedString is a generated union type.
type MarkedString struct {
	value any
	tag   int
}

// NewMarkedStringString constructs the string variant of MarkedString.
func NewMarkedStringString(v string) MarkedString {
	return MarkedString{value: v, tag: 0}
}

// NewMarkedStringMarkedStringWithLanguage constructs the MarkedStringWithLanguage variant of MarkedString.
func NewMarkedStringMarkedStringWithLanguage(v MarkedStringWithLanguage) MarkedString {
	return MarkedString{value: v, tag: 1}
}

// DocumentFilter is a generated union type.
type DocumentFilter struct {
	value any
	tag   int
}

// NewDocumentFilterTextDocumentFilter constructs the TextDocumentFilter variant of DocumentFilter.
func NewDocumentFilterTextDocumentFilter(v TextDocumentFilter) DocumentFilter {
	return DocumentFilter{value: v, tag: 0}
}

// NewDocumentFilterNotebookCellTextDocumentFilter constructs the NotebookCellTextDocumentFilter variant of DocumentFilter.
func NewDocumentFilterNotebookCellTextDocumentFilter(v NotebookCellTextDocumentFilter) DocumentFilter {
	return DocumentFilter{value: v, tag: 1}
}

// GlobPattern is a generated union type.
type GlobPattern struct {
	value any
	tag   int
}

// NewGlobPatternPattern constructs the Pattern variant of GlobPattern.
func NewGlobPatternPattern(v Pattern) GlobPattern {
	return GlobPattern{value: v, tag: 0}
}

// NewGlobPatternRelativePattern constructs the RelativePattern variant of GlobPattern.
func NewGlobPatternRelativePattern(v RelativePattern) GlobPattern {
	return GlobPattern{value: v, tag: 1}
}

// TextDocumentFilter is a generated union type.
type TextDocumentFilter struct {
	value any
	tag   int
}

// NewTextDocumentFilterTextDocumentFilterLanguage constructs the TextDocumentFilterLanguage variant of TextDocumentFilter.
func NewTextDocumentFilterTextDocumentFilterLanguage(v TextDocumentFilterLanguage) TextDocumentFilter {
	return TextDocumentFilter{value: v, tag: 0}
}

// NewTextDocumentFilterTextDocumentFilterScheme constructs the TextDocumentFilterScheme variant of TextDocumentFilter.
func NewTextDocumentFilterTextDocumentFilterScheme(v TextDocumentFilterScheme) TextDocumentFilter {
	return TextDocumentFilter{value: v, tag: 1}
}

// NewTextDocumentFilterTextDocumentFilterPattern constructs the TextDocumentFilterPattern variant of TextDocumentFilter.
func NewTextDocumentFilterTextDocumentFilterPattern(v TextDocumentFilterPattern) TextDocumentFilter {
	return TextDocumentFilter{value: v, tag: 2}
}

// NotebookDocumentFilter is a generated union type.
type NotebookDocumentFilter struct {
	value any
	tag   int
}

// NewNotebookDocumentFilterNotebookDocumentFilterNotebookType constructs the NotebookDocumentFilterNotebookType variant of NotebookDocumentFilter.
func NewNotebookDocumentFilterNotebookDocumentFilterNotebookType(v NotebookDocumentFilterNotebookType) NotebookDocumentFilter {
	return NotebookDocumentFilter{value: v, tag: 0}
}

// NewNotebookDocumentFilterNotebookDocumentFilterScheme constructs the NotebookDocumentFilterScheme variant of NotebookDocumentFilter.
func NewNotebookDocumentFilterNotebookDocumentFilterScheme(v NotebookDocumentFilterScheme) NotebookDocumentFilter {
	return NotebookDocumentFilter{value: v, tag: 1}
}

// NewNotebookDocumentFilterNotebookDocumentFilterPattern constructs the NotebookDocumentFilterPattern variant of NotebookDocumentFilter.
func NewNotebookDocumentFilterNotebookDocumentFilterPattern(v NotebookDocumentFilterPattern) NotebookDocumentFilter {
	return NotebookDocumentFilter{value: v, tag: 2}
}

// OrCancelParamsId is a generated union type.
type OrCancelParamsId struct {
	value any
	tag   int
}

// NewOrCancelParamsIdInteger constructs the integer variant of OrCancelParamsId.
func NewOrCancelParamsIdInteger(v int32) OrCancelParamsId {
	return OrCancelParamsId{value: v, tag: 0}
}

// NewOrCancelParamsIdString constructs the string variant of OrCancelParamsId.
func NewOrCancelParamsIdString(v string) OrCancelParamsId {
	return OrCancelParamsId{value: v, tag: 1}
}

// OrClientSemanticTokensRequestOptionsRange is a generated union type.
type OrClientSemanticTokensRequestOptionsRange struct {
	value any
	tag   int
}

// NewOrClientSemanticTokensRequestOptionsRangeBoolean constructs the boolean variant of OrClientSemanticTokensRequestOptionsRange.
func NewOrClientSemanticTokensRequestOptionsRangeBoolean(v bool) OrClientSemanticTokensRequestOptionsRange {
	return OrClientSemanticTokensRequestOptionsRange{value: v, tag: 0}
}

// NewOrClientSemanticTokensRequestOptionsRangeLiteral1 constructs the literal variant of OrClientSemanticTokensRequestOptionsRange.
func NewOrClientSemanticTokensRequestOptionsRangeLiteral1(v LitClientSemanticTokensRequestOptionsRangeItem1) OrClientSemanticTokensRequestOptionsRange {
	return OrClientSemanticTokensRequestOptionsRange{value: v, tag: 1}
}

// LitClientSemanticTokensRequestOptionsRangeItem1 is a generated anonymous literal type.
type LitClientSemanticTokensRequestOptionsRangeItem1 struct {
}

// OrClientSemanticTokensRequestOptionsFull is a generated union type.
type OrClientSemanticTokensRequestOptionsFull struct {
	value any
	tag   int
}

// NewOrClientSemanticTokensRequestOptionsFullBoolean constructs the boolean variant of OrClientSemanticTokensRequestOptionsFull.
func NewOrClientSemanticTokensRequestOptionsFullBoolean(v bool) OrClientSemanticTokensRequestOptionsFull {
	return OrClientSemanticTokensRequestOptionsFull{value: v, tag: 0}
}

// NewOrClientSemanticTokensRequestOptionsFullClientSemanticTokensRequestFullDelta constructs the ClientSemanticTokensRequestFullDelta variant of OrClientSemanticTokensRequestOptionsFull.
func NewOrClientSemanticTokensRequestOptionsFullClientSemanticTokensRequestFullDelta(v ClientSemanticTokensRequestFullDelta) OrClientSemanticTokensRequestOptionsFull {
	return OrClientSemanticTokensRequestOptionsFull{value: v, tag: 1}
}

// OrCompletionItemDocumentation is a generated union type.
type OrCompletionItemDocumentation struct {
	value any
	tag   int
}

// NewOrCompletionItemDocumentationString constructs the string variant of OrCompletionItemDocumentation.
func NewOrCompletionItemDocumentationString(v string) OrCompletionItemDocumentation {
	return OrCompletionItemDocumentation{value: v, tag: 0}
}

// NewOrCompletionItemDocumentationMarkupContent constructs the MarkupContent variant of OrCompletionItemDocumentation.
func NewOrCompletionItemDocumentationMarkupContent(v MarkupContent) OrCompletionItemDocumentation {
	return OrCompletionItemDocumentation{value: v, tag: 1}
}

// OrCompletionItemTextEdit is a generated union type.
type OrCompletionItemTextEdit struct {
	value any
	tag   int
}

// NewOrCompletionItemTextEditTextEdit constructs the TextEdit variant of OrCompletionItemTextEdit.
func NewOrCompletionItemTextEditTextEdit(v TextEdit) OrCompletionItemTextEdit {
	return OrCompletionItemTextEdit{value: v, tag: 0}
}

// NewOrCompletionItemTextEditInsertReplaceEdit constructs the InsertReplaceEdit variant of OrCompletionItemTextEdit.
func NewOrCompletionItemTextEditInsertReplaceEdit(v InsertReplaceEdit) OrCompletionItemTextEdit {
	return OrCompletionItemTextEdit{value: v, tag: 1}
}

// OrCompletionItemDefaultsEditRange is a generated union type.
type OrCompletionItemDefaultsEditRange struct {
	value any
	tag   int
}

// NewOrCompletionItemDefaultsEditRangeRange constructs the Range variant of OrCompletionItemDefaultsEditRange.
func NewOrCompletionItemDefaultsEditRangeRange(v Range) OrCompletionItemDefaultsEditRange {
	return OrCompletionItemDefaultsEditRange{value: v, tag: 0}
}

// NewOrCompletionItemDefaultsEditRangeEditRangeWithInsertReplace constructs the EditRangeWithInsertReplace variant of OrCompletionItemDefaultsEditRange.
func NewOrCompletionItemDefaultsEditRangeEditRangeWithInsertReplace(v EditRangeWithInsertReplace) OrCompletionItemDefaultsEditRange {
	return OrCompletionItemDefaultsEditRange{value: v, tag: 1}
}

// OrDiagnosticCode is a generated union type.
type OrDiagnosticCode struct {
	value any
	tag   int
}

// NewOrDiagnosticCodeInteger constructs the integer variant of OrDiagnosticCode.
func NewOrDiagnosticCodeInteger(v int32) OrDiagnosticCode {
	return OrDiagnosticCode{value: v, tag: 0}
}

// NewOrDiagnosticCodeString constructs the string variant of OrDiagnosticCode.
func NewOrDiagnosticCodeString(v string) OrDiagnosticCode {
	return OrDiagnosticCode{value: v, tag: 1}
}

// OrDiagnosticMessage is a generated union type.
type OrDiagnosticMessage struct {
	value any
	tag   int
}

// NewOrDiagnosticMessageString constructs the string variant of OrDiagnosticMessage.
func NewOrDiagnosticMessageString(v string) OrDiagnosticMessage {
	return OrDiagnosticMessage{value: v, tag: 0}
}

// NewOrDiagnosticMessageMarkupContent constructs the MarkupContent variant of OrDiagnosticMessage.
func NewOrDiagnosticMessageMarkupContent(v MarkupContent) OrDiagnosticMessage {
	return OrDiagnosticMessage{value: v, tag: 1}
}

// OrDidChangeConfigurationRegistrationOptionsSection is a generated union type.
type OrDidChangeConfigurationRegistrationOptionsSection struct {
	value any
	tag   int
}

// NewOrDidChangeConfigurationRegistrationOptionsSectionString constructs the string variant of OrDidChangeConfigurationRegistrationOptionsSection.
func NewOrDidChangeConfigurationRegistrationOptionsSectionString(v string) OrDidChangeConfigurationRegistrationOptionsSection {
	return OrDidChangeConfigurationRegistrationOptionsSection{value: v, tag: 0}
}

// NewOrDidChangeConfigurationRegistrationOptionsSectionArray1 constructs the string[] variant of OrDidChangeConfigurationRegistrationOptionsSection.
func NewOrDidChangeConfigurationRegistrationOptionsSectionArray1(v []string) OrDidChangeConfigurationRegistrationOptionsSection {
	return OrDidChangeConfigurationRegistrationOptionsSection{value: v, tag: 1}
}

// OrHoverContents is a generated union type.
type OrHoverContents struct {
	value any
	tag   int
}

// NewOrHoverContentsMarkupContent constructs the MarkupContent variant of OrHoverContents.
func NewOrHoverContentsMarkupContent(v MarkupContent) OrHoverContents {
	return OrHoverContents{value: v, tag: 0}
}

// NewOrHoverContentsMarkedString constructs the MarkedString variant of OrHoverContents.
func NewOrHoverContentsMarkedString(v MarkedString) OrHoverContents {
	return OrHoverContents{value: v, tag: 1}
}

// NewOrHoverContentsArray2 constructs the MarkedString[] variant of OrHoverContents.
func NewOrHoverContentsArray2(v []MarkedString) OrHoverContents {
	return OrHoverContents{value: v, tag: 2}
}

// OrInlayHintLabel is a generated union type.
type OrInlayHintLabel struct {
	value any
	tag   int
}

// NewOrInlayHintLabelString constructs the string variant of OrInlayHintLabel.
func NewOrInlayHintLabelString(v string) OrInlayHintLabel {
	return OrInlayHintLabel{value: v, tag: 0}
}

// NewOrInlayHintLabelArray1 constructs the InlayHintLabelPart[] variant of OrInlayHintLabel.
func NewOrInlayHintLabelArray1(v []InlayHintLabelPart) OrInlayHintLabel {
	return OrInlayHintLabel{value: v, tag: 1}
}

// OrInlayHintTooltip is a generated union type.
type OrInlayHintTooltip struct {
	value any
	tag   int
}

// NewOrInlayHintTooltipString constructs the string variant of OrInlayHintTooltip.
func NewOrInlayHintTooltipString(v string) OrInlayHintTooltip {
	return OrInlayHintTooltip{value: v, tag: 0}
}

// NewOrInlayHintTooltipMarkupContent constructs the MarkupContent variant of OrInlayHintTooltip.
func NewOrInlayHintTooltipMarkupContent(v MarkupContent) OrInlayHintTooltip {
	return OrInlayHintTooltip{value: v, tag: 1}
}

// OrInlayHintLabelPartTooltip is a generated union type.
type OrInlayHintLabelPartTooltip struct {
	value any
	tag   int
}

// NewOrInlayHintLabelPartTooltipString constructs the string variant of OrInlayHintLabelPartTooltip.
func NewOrInlayHintLabelPartTooltipString(v string) OrInlayHintLabelPartTooltip {
	return OrInlayHintLabelPartTooltip{value: v, tag: 0}
}

// NewOrInlayHintLabelPartTooltipMarkupContent constructs the MarkupContent variant of OrInlayHintLabelPartTooltip.
func NewOrInlayHintLabelPartTooltipMarkupContent(v MarkupContent) OrInlayHintLabelPartTooltip {
	return OrInlayHintLabelPartTooltip{value: v, tag: 1}
}

// OrInlineCompletionItemInsertText is a generated union type.
type OrInlineCompletionItemInsertText struct {
	value any
	tag   int
}

// NewOrInlineCompletionItemInsertTextString constructs the string variant of OrInlineCompletionItemInsertText.
func NewOrInlineCompletionItemInsertTextString(v string) OrInlineCompletionItemInsertText {
	return OrInlineCompletionItemInsertText{value: v, tag: 0}
}

// NewOrInlineCompletionItemInsertTextStringValue constructs the StringValue variant of OrInlineCompletionItemInsertText.
func NewOrInlineCompletionItemInsertTextStringValue(v StringValue) OrInlineCompletionItemInsertText {
	return OrInlineCompletionItemInsertText{value: v, tag: 1}
}

// OrNotebookCellTextDocumentFilterNotebook is a generated union type.
type OrNotebookCellTextDocumentFilterNotebook struct {
	value any
	tag   int
}

// NewOrNotebookCellTextDocumentFilterNotebookString constructs the string variant of OrNotebookCellTextDocumentFilterNotebook.
func NewOrNotebookCellTextDocumentFilterNotebookString(v string) OrNotebookCellTextDocumentFilterNotebook {
	return OrNotebookCellTextDocumentFilterNotebook{value: v, tag: 0}
}

// NewOrNotebookCellTextDocumentFilterNotebookNotebookDocumentFilter constructs the NotebookDocumentFilter variant of OrNotebookCellTextDocumentFilterNotebook.
func NewOrNotebookCellTextDocumentFilterNotebookNotebookDocumentFilter(v NotebookDocumentFilter) OrNotebookCellTextDocumentFilterNotebook {
	return OrNotebookCellTextDocumentFilterNotebook{value: v, tag: 1}
}

// OrNotebookDocumentFilterWithCellsNotebook is a generated union type.
type OrNotebookDocumentFilterWithCellsNotebook struct {
	value any
	tag   int
}

// NewOrNotebookDocumentFilterWithCellsNotebookString constructs the string variant of OrNotebookDocumentFilterWithCellsNotebook.
func NewOrNotebookDocumentFilterWithCellsNotebookString(v string) OrNotebookDocumentFilterWithCellsNotebook {
	return OrNotebookDocumentFilterWithCellsNotebook{value: v, tag: 0}
}

// NewOrNotebookDocumentFilterWithCellsNotebookNotebookDocumentFilter constructs the NotebookDocumentFilter variant of OrNotebookDocumentFilterWithCellsNotebook.
func NewOrNotebookDocumentFilterWithCellsNotebookNotebookDocumentFilter(v NotebookDocumentFilter) OrNotebookDocumentFilterWithCellsNotebook {
	return OrNotebookDocumentFilterWithCellsNotebook{value: v, tag: 1}
}

// OrNotebookDocumentFilterWithNotebookNotebook is a generated union type.
type OrNotebookDocumentFilterWithNotebookNotebook struct {
	value any
	tag   int
}

// NewOrNotebookDocumentFilterWithNotebookNotebookString constructs the string variant of OrNotebookDocumentFilterWithNotebookNotebook.
func NewOrNotebookDocumentFilterWithNotebookNotebookString(v string) OrNotebookDocumentFilterWithNotebookNotebook {
	return OrNotebookDocumentFilterWithNotebookNotebook{value: v, tag: 0}
}

// NewOrNotebookDocumentFilterWithNotebookNotebookNotebookDocumentFilter constructs the NotebookDocumentFilter variant of OrNotebookDocumentFilterWithNotebookNotebook.
func NewOrNotebookDocumentFilterWithNotebookNotebookNotebookDocumentFilter(v NotebookDocumentFilter) OrNotebookDocumentFilterWithNotebookNotebook {
	return OrNotebookDocumentFilterWithNotebookNotebook{value: v, tag: 1}
}

// OrNotebookDocumentSyncOptionsNotebookSelectorElem is a generated union type.
type OrNotebookDocumentSyncOptionsNotebookSelectorElem struct {
	value any
	tag   int
}

// NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithNotebook constructs the NotebookDocumentFilterWithNotebook variant of OrNotebookDocumentSyncOptionsNotebookSelectorElem.
func NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithNotebook(v NotebookDocumentFilterWithNotebook) OrNotebookDocumentSyncOptionsNotebookSelectorElem {
	return OrNotebookDocumentSyncOptionsNotebookSelectorElem{value: v, tag: 0}
}

// NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithCells constructs the NotebookDocumentFilterWithCells variant of OrNotebookDocumentSyncOptionsNotebookSelectorElem.
func NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithCells(v NotebookDocumentFilterWithCells) OrNotebookDocumentSyncOptionsNotebookSelectorElem {
	return OrNotebookDocumentSyncOptionsNotebookSelectorElem{value: v, tag: 1}
}

// OrOptionalVersionedTextDocumentIdentifierVersion is a generated union type.
type OrOptionalVersionedTextDocumentIdentifierVersion struct {
	value any
	tag   int
}

// NewOrOptionalVersionedTextDocumentIdentifierVersionInteger constructs the integer variant of OrOptionalVersionedTextDocumentIdentifierVersion.
func NewOrOptionalVersionedTextDocumentIdentifierVersionInteger(v int32) OrOptionalVersionedTextDocumentIdentifierVersion {
	return OrOptionalVersionedTextDocumentIdentifierVersion{value: v, tag: 0}
}

// OrParameterInformationLabel is a generated union type.
type OrParameterInformationLabel struct {
	value any
	tag   int
}

// NewOrParameterInformationLabelString constructs the string variant of OrParameterInformationLabel.
func NewOrParameterInformationLabelString(v string) OrParameterInformationLabel {
	return OrParameterInformationLabel{value: v, tag: 0}
}

// NewOrParameterInformationLabelVariant1 constructs the [uinteger, uinteger] variant of OrParameterInformationLabel.
func NewOrParameterInformationLabelVariant1(v TupleParameterInformationLabelItem1) OrParameterInformationLabel {
	return OrParameterInformationLabel{value: v, tag: 1}
}

// TupleParameterInformationLabelItem1 is a generated tuple type.
type TupleParameterInformationLabelItem1 struct {
	Item0 uint32
	Item1 uint32
}

// OrParameterInformationDocumentation is a generated union type.
type OrParameterInformationDocumentation struct {
	value any
	tag   int
}

// NewOrParameterInformationDocumentationString constructs the string variant of OrParameterInformationDocumentation.
func NewOrParameterInformationDocumentationString(v string) OrParameterInformationDocumentation {
	return OrParameterInformationDocumentation{value: v, tag: 0}
}

// NewOrParameterInformationDocumentationMarkupContent constructs the MarkupContent variant of OrParameterInformationDocumentation.
func NewOrParameterInformationDocumentationMarkupContent(v MarkupContent) OrParameterInformationDocumentation {
	return OrParameterInformationDocumentation{value: v, tag: 1}
}

// OrRelativePatternBaseUri is a generated union type.
type OrRelativePatternBaseUri struct {
	value any
	tag   int
}

// NewOrRelativePatternBaseUriWorkspaceFolder constructs the WorkspaceFolder variant of OrRelativePatternBaseUri.
func NewOrRelativePatternBaseUriWorkspaceFolder(v WorkspaceFolder) OrRelativePatternBaseUri {
	return OrRelativePatternBaseUri{value: v, tag: 0}
}

// NewOrRelativePatternBaseUriURI constructs the URI variant of OrRelativePatternBaseUri.
func NewOrRelativePatternBaseUriURI(v string) OrRelativePatternBaseUri {
	return OrRelativePatternBaseUri{value: v, tag: 1}
}

// OrSemanticTokensOptionsRange is a generated union type.
type OrSemanticTokensOptionsRange struct {
	value any
	tag   int
}

// NewOrSemanticTokensOptionsRangeBoolean constructs the boolean variant of OrSemanticTokensOptionsRange.
func NewOrSemanticTokensOptionsRangeBoolean(v bool) OrSemanticTokensOptionsRange {
	return OrSemanticTokensOptionsRange{value: v, tag: 0}
}

// NewOrSemanticTokensOptionsRangeLiteral1 constructs the literal variant of OrSemanticTokensOptionsRange.
func NewOrSemanticTokensOptionsRangeLiteral1(v LitSemanticTokensOptionsRangeItem1) OrSemanticTokensOptionsRange {
	return OrSemanticTokensOptionsRange{value: v, tag: 1}
}

// LitSemanticTokensOptionsRangeItem1 is a generated anonymous literal type.
type LitSemanticTokensOptionsRangeItem1 struct {
}

// OrSemanticTokensOptionsFull is a generated union type.
type OrSemanticTokensOptionsFull struct {
	value any
	tag   int
}

// NewOrSemanticTokensOptionsFullBoolean constructs the boolean variant of OrSemanticTokensOptionsFull.
func NewOrSemanticTokensOptionsFullBoolean(v bool) OrSemanticTokensOptionsFull {
	return OrSemanticTokensOptionsFull{value: v, tag: 0}
}

// NewOrSemanticTokensOptionsFullSemanticTokensFullDelta constructs the SemanticTokensFullDelta variant of OrSemanticTokensOptionsFull.
func NewOrSemanticTokensOptionsFullSemanticTokensFullDelta(v SemanticTokensFullDelta) OrSemanticTokensOptionsFull {
	return OrSemanticTokensOptionsFull{value: v, tag: 1}
}

// OrServerCapabilitiesTextDocumentSync is a generated union type.
type OrServerCapabilitiesTextDocumentSync struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncOptions constructs the TextDocumentSyncOptions variant of OrServerCapabilitiesTextDocumentSync.
func NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncOptions(v TextDocumentSyncOptions) OrServerCapabilitiesTextDocumentSync {
	return OrServerCapabilitiesTextDocumentSync{value: v, tag: 0}
}

// NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncKind constructs the TextDocumentSyncKind variant of OrServerCapabilitiesTextDocumentSync.
func NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncKind(v TextDocumentSyncKind) OrServerCapabilitiesTextDocumentSync {
	return OrServerCapabilitiesTextDocumentSync{value: v, tag: 1}
}

// OrServerCapabilitiesNotebookDocumentSync is a generated union type.
type OrServerCapabilitiesNotebookDocumentSync struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncOptions constructs the NotebookDocumentSyncOptions variant of OrServerCapabilitiesNotebookDocumentSync.
func NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncOptions(v NotebookDocumentSyncOptions) OrServerCapabilitiesNotebookDocumentSync {
	return OrServerCapabilitiesNotebookDocumentSync{value: v, tag: 0}
}

// NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncRegistrationOptions constructs the NotebookDocumentSyncRegistrationOptions variant of OrServerCapabilitiesNotebookDocumentSync.
func NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncRegistrationOptions(v NotebookDocumentSyncRegistrationOptions) OrServerCapabilitiesNotebookDocumentSync {
	return OrServerCapabilitiesNotebookDocumentSync{value: v, tag: 1}
}

// OrServerCapabilitiesHoverProvider is a generated union type.
type OrServerCapabilitiesHoverProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesHoverProviderBoolean constructs the boolean variant of OrServerCapabilitiesHoverProvider.
func NewOrServerCapabilitiesHoverProviderBoolean(v bool) OrServerCapabilitiesHoverProvider {
	return OrServerCapabilitiesHoverProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesHoverProviderHoverOptions constructs the HoverOptions variant of OrServerCapabilitiesHoverProvider.
func NewOrServerCapabilitiesHoverProviderHoverOptions(v HoverOptions) OrServerCapabilitiesHoverProvider {
	return OrServerCapabilitiesHoverProvider{value: v, tag: 1}
}

// OrServerCapabilitiesDeclarationProvider is a generated union type.
type OrServerCapabilitiesDeclarationProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDeclarationProviderBoolean constructs the boolean variant of OrServerCapabilitiesDeclarationProvider.
func NewOrServerCapabilitiesDeclarationProviderBoolean(v bool) OrServerCapabilitiesDeclarationProvider {
	return OrServerCapabilitiesDeclarationProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDeclarationProviderDeclarationOptions constructs the DeclarationOptions variant of OrServerCapabilitiesDeclarationProvider.
func NewOrServerCapabilitiesDeclarationProviderDeclarationOptions(v DeclarationOptions) OrServerCapabilitiesDeclarationProvider {
	return OrServerCapabilitiesDeclarationProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesDeclarationProviderDeclarationRegistrationOptions constructs the DeclarationRegistrationOptions variant of OrServerCapabilitiesDeclarationProvider.
func NewOrServerCapabilitiesDeclarationProviderDeclarationRegistrationOptions(v DeclarationRegistrationOptions) OrServerCapabilitiesDeclarationProvider {
	return OrServerCapabilitiesDeclarationProvider{value: v, tag: 2}
}

// OrServerCapabilitiesDefinitionProvider is a generated union type.
type OrServerCapabilitiesDefinitionProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDefinitionProviderBoolean constructs the boolean variant of OrServerCapabilitiesDefinitionProvider.
func NewOrServerCapabilitiesDefinitionProviderBoolean(v bool) OrServerCapabilitiesDefinitionProvider {
	return OrServerCapabilitiesDefinitionProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDefinitionProviderDefinitionOptions constructs the DefinitionOptions variant of OrServerCapabilitiesDefinitionProvider.
func NewOrServerCapabilitiesDefinitionProviderDefinitionOptions(v DefinitionOptions) OrServerCapabilitiesDefinitionProvider {
	return OrServerCapabilitiesDefinitionProvider{value: v, tag: 1}
}

// OrServerCapabilitiesTypeDefinitionProvider is a generated union type.
type OrServerCapabilitiesTypeDefinitionProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesTypeDefinitionProviderBoolean constructs the boolean variant of OrServerCapabilitiesTypeDefinitionProvider.
func NewOrServerCapabilitiesTypeDefinitionProviderBoolean(v bool) OrServerCapabilitiesTypeDefinitionProvider {
	return OrServerCapabilitiesTypeDefinitionProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionOptions constructs the TypeDefinitionOptions variant of OrServerCapabilitiesTypeDefinitionProvider.
func NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionOptions(v TypeDefinitionOptions) OrServerCapabilitiesTypeDefinitionProvider {
	return OrServerCapabilitiesTypeDefinitionProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionRegistrationOptions constructs the TypeDefinitionRegistrationOptions variant of OrServerCapabilitiesTypeDefinitionProvider.
func NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionRegistrationOptions(v TypeDefinitionRegistrationOptions) OrServerCapabilitiesTypeDefinitionProvider {
	return OrServerCapabilitiesTypeDefinitionProvider{value: v, tag: 2}
}

// OrServerCapabilitiesImplementationProvider is a generated union type.
type OrServerCapabilitiesImplementationProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesImplementationProviderBoolean constructs the boolean variant of OrServerCapabilitiesImplementationProvider.
func NewOrServerCapabilitiesImplementationProviderBoolean(v bool) OrServerCapabilitiesImplementationProvider {
	return OrServerCapabilitiesImplementationProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesImplementationProviderImplementationOptions constructs the ImplementationOptions variant of OrServerCapabilitiesImplementationProvider.
func NewOrServerCapabilitiesImplementationProviderImplementationOptions(v ImplementationOptions) OrServerCapabilitiesImplementationProvider {
	return OrServerCapabilitiesImplementationProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesImplementationProviderImplementationRegistrationOptions constructs the ImplementationRegistrationOptions variant of OrServerCapabilitiesImplementationProvider.
func NewOrServerCapabilitiesImplementationProviderImplementationRegistrationOptions(v ImplementationRegistrationOptions) OrServerCapabilitiesImplementationProvider {
	return OrServerCapabilitiesImplementationProvider{value: v, tag: 2}
}

// OrServerCapabilitiesReferencesProvider is a generated union type.
type OrServerCapabilitiesReferencesProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesReferencesProviderBoolean constructs the boolean variant of OrServerCapabilitiesReferencesProvider.
func NewOrServerCapabilitiesReferencesProviderBoolean(v bool) OrServerCapabilitiesReferencesProvider {
	return OrServerCapabilitiesReferencesProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesReferencesProviderReferenceOptions constructs the ReferenceOptions variant of OrServerCapabilitiesReferencesProvider.
func NewOrServerCapabilitiesReferencesProviderReferenceOptions(v ReferenceOptions) OrServerCapabilitiesReferencesProvider {
	return OrServerCapabilitiesReferencesProvider{value: v, tag: 1}
}

// OrServerCapabilitiesDocumentHighlightProvider is a generated union type.
type OrServerCapabilitiesDocumentHighlightProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDocumentHighlightProviderBoolean constructs the boolean variant of OrServerCapabilitiesDocumentHighlightProvider.
func NewOrServerCapabilitiesDocumentHighlightProviderBoolean(v bool) OrServerCapabilitiesDocumentHighlightProvider {
	return OrServerCapabilitiesDocumentHighlightProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDocumentHighlightProviderDocumentHighlightOptions constructs the DocumentHighlightOptions variant of OrServerCapabilitiesDocumentHighlightProvider.
func NewOrServerCapabilitiesDocumentHighlightProviderDocumentHighlightOptions(v DocumentHighlightOptions) OrServerCapabilitiesDocumentHighlightProvider {
	return OrServerCapabilitiesDocumentHighlightProvider{value: v, tag: 1}
}

// OrServerCapabilitiesDocumentSymbolProvider is a generated union type.
type OrServerCapabilitiesDocumentSymbolProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDocumentSymbolProviderBoolean constructs the boolean variant of OrServerCapabilitiesDocumentSymbolProvider.
func NewOrServerCapabilitiesDocumentSymbolProviderBoolean(v bool) OrServerCapabilitiesDocumentSymbolProvider {
	return OrServerCapabilitiesDocumentSymbolProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDocumentSymbolProviderDocumentSymbolOptions constructs the DocumentSymbolOptions variant of OrServerCapabilitiesDocumentSymbolProvider.
func NewOrServerCapabilitiesDocumentSymbolProviderDocumentSymbolOptions(v DocumentSymbolOptions) OrServerCapabilitiesDocumentSymbolProvider {
	return OrServerCapabilitiesDocumentSymbolProvider{value: v, tag: 1}
}

// OrServerCapabilitiesCodeActionProvider is a generated union type.
type OrServerCapabilitiesCodeActionProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesCodeActionProviderBoolean constructs the boolean variant of OrServerCapabilitiesCodeActionProvider.
func NewOrServerCapabilitiesCodeActionProviderBoolean(v bool) OrServerCapabilitiesCodeActionProvider {
	return OrServerCapabilitiesCodeActionProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesCodeActionProviderCodeActionOptions constructs the CodeActionOptions variant of OrServerCapabilitiesCodeActionProvider.
func NewOrServerCapabilitiesCodeActionProviderCodeActionOptions(v CodeActionOptions) OrServerCapabilitiesCodeActionProvider {
	return OrServerCapabilitiesCodeActionProvider{value: v, tag: 1}
}

// OrServerCapabilitiesColorProvider is a generated union type.
type OrServerCapabilitiesColorProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesColorProviderBoolean constructs the boolean variant of OrServerCapabilitiesColorProvider.
func NewOrServerCapabilitiesColorProviderBoolean(v bool) OrServerCapabilitiesColorProvider {
	return OrServerCapabilitiesColorProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesColorProviderDocumentColorOptions constructs the DocumentColorOptions variant of OrServerCapabilitiesColorProvider.
func NewOrServerCapabilitiesColorProviderDocumentColorOptions(v DocumentColorOptions) OrServerCapabilitiesColorProvider {
	return OrServerCapabilitiesColorProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesColorProviderDocumentColorRegistrationOptions constructs the DocumentColorRegistrationOptions variant of OrServerCapabilitiesColorProvider.
func NewOrServerCapabilitiesColorProviderDocumentColorRegistrationOptions(v DocumentColorRegistrationOptions) OrServerCapabilitiesColorProvider {
	return OrServerCapabilitiesColorProvider{value: v, tag: 2}
}

// OrServerCapabilitiesWorkspaceSymbolProvider is a generated union type.
type OrServerCapabilitiesWorkspaceSymbolProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesWorkspaceSymbolProviderBoolean constructs the boolean variant of OrServerCapabilitiesWorkspaceSymbolProvider.
func NewOrServerCapabilitiesWorkspaceSymbolProviderBoolean(v bool) OrServerCapabilitiesWorkspaceSymbolProvider {
	return OrServerCapabilitiesWorkspaceSymbolProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesWorkspaceSymbolProviderWorkspaceSymbolOptions constructs the WorkspaceSymbolOptions variant of OrServerCapabilitiesWorkspaceSymbolProvider.
func NewOrServerCapabilitiesWorkspaceSymbolProviderWorkspaceSymbolOptions(v WorkspaceSymbolOptions) OrServerCapabilitiesWorkspaceSymbolProvider {
	return OrServerCapabilitiesWorkspaceSymbolProvider{value: v, tag: 1}
}

// OrServerCapabilitiesDocumentFormattingProvider is a generated union type.
type OrServerCapabilitiesDocumentFormattingProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDocumentFormattingProviderBoolean constructs the boolean variant of OrServerCapabilitiesDocumentFormattingProvider.
func NewOrServerCapabilitiesDocumentFormattingProviderBoolean(v bool) OrServerCapabilitiesDocumentFormattingProvider {
	return OrServerCapabilitiesDocumentFormattingProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDocumentFormattingProviderDocumentFormattingOptions constructs the DocumentFormattingOptions variant of OrServerCapabilitiesDocumentFormattingProvider.
func NewOrServerCapabilitiesDocumentFormattingProviderDocumentFormattingOptions(v DocumentFormattingOptions) OrServerCapabilitiesDocumentFormattingProvider {
	return OrServerCapabilitiesDocumentFormattingProvider{value: v, tag: 1}
}

// OrServerCapabilitiesDocumentRangeFormattingProvider is a generated union type.
type OrServerCapabilitiesDocumentRangeFormattingProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDocumentRangeFormattingProviderBoolean constructs the boolean variant of OrServerCapabilitiesDocumentRangeFormattingProvider.
func NewOrServerCapabilitiesDocumentRangeFormattingProviderBoolean(v bool) OrServerCapabilitiesDocumentRangeFormattingProvider {
	return OrServerCapabilitiesDocumentRangeFormattingProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDocumentRangeFormattingProviderDocumentRangeFormattingOptions constructs the DocumentRangeFormattingOptions variant of OrServerCapabilitiesDocumentRangeFormattingProvider.
func NewOrServerCapabilitiesDocumentRangeFormattingProviderDocumentRangeFormattingOptions(v DocumentRangeFormattingOptions) OrServerCapabilitiesDocumentRangeFormattingProvider {
	return OrServerCapabilitiesDocumentRangeFormattingProvider{value: v, tag: 1}
}

// OrServerCapabilitiesRenameProvider is a generated union type.
type OrServerCapabilitiesRenameProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesRenameProviderBoolean constructs the boolean variant of OrServerCapabilitiesRenameProvider.
func NewOrServerCapabilitiesRenameProviderBoolean(v bool) OrServerCapabilitiesRenameProvider {
	return OrServerCapabilitiesRenameProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesRenameProviderRenameOptions constructs the RenameOptions variant of OrServerCapabilitiesRenameProvider.
func NewOrServerCapabilitiesRenameProviderRenameOptions(v RenameOptions) OrServerCapabilitiesRenameProvider {
	return OrServerCapabilitiesRenameProvider{value: v, tag: 1}
}

// OrServerCapabilitiesFoldingRangeProvider is a generated union type.
type OrServerCapabilitiesFoldingRangeProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesFoldingRangeProviderBoolean constructs the boolean variant of OrServerCapabilitiesFoldingRangeProvider.
func NewOrServerCapabilitiesFoldingRangeProviderBoolean(v bool) OrServerCapabilitiesFoldingRangeProvider {
	return OrServerCapabilitiesFoldingRangeProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeOptions constructs the FoldingRangeOptions variant of OrServerCapabilitiesFoldingRangeProvider.
func NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeOptions(v FoldingRangeOptions) OrServerCapabilitiesFoldingRangeProvider {
	return OrServerCapabilitiesFoldingRangeProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeRegistrationOptions constructs the FoldingRangeRegistrationOptions variant of OrServerCapabilitiesFoldingRangeProvider.
func NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeRegistrationOptions(v FoldingRangeRegistrationOptions) OrServerCapabilitiesFoldingRangeProvider {
	return OrServerCapabilitiesFoldingRangeProvider{value: v, tag: 2}
}

// OrServerCapabilitiesSelectionRangeProvider is a generated union type.
type OrServerCapabilitiesSelectionRangeProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesSelectionRangeProviderBoolean constructs the boolean variant of OrServerCapabilitiesSelectionRangeProvider.
func NewOrServerCapabilitiesSelectionRangeProviderBoolean(v bool) OrServerCapabilitiesSelectionRangeProvider {
	return OrServerCapabilitiesSelectionRangeProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeOptions constructs the SelectionRangeOptions variant of OrServerCapabilitiesSelectionRangeProvider.
func NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeOptions(v SelectionRangeOptions) OrServerCapabilitiesSelectionRangeProvider {
	return OrServerCapabilitiesSelectionRangeProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeRegistrationOptions constructs the SelectionRangeRegistrationOptions variant of OrServerCapabilitiesSelectionRangeProvider.
func NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeRegistrationOptions(v SelectionRangeRegistrationOptions) OrServerCapabilitiesSelectionRangeProvider {
	return OrServerCapabilitiesSelectionRangeProvider{value: v, tag: 2}
}

// OrServerCapabilitiesCallHierarchyProvider is a generated union type.
type OrServerCapabilitiesCallHierarchyProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesCallHierarchyProviderBoolean constructs the boolean variant of OrServerCapabilitiesCallHierarchyProvider.
func NewOrServerCapabilitiesCallHierarchyProviderBoolean(v bool) OrServerCapabilitiesCallHierarchyProvider {
	return OrServerCapabilitiesCallHierarchyProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyOptions constructs the CallHierarchyOptions variant of OrServerCapabilitiesCallHierarchyProvider.
func NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyOptions(v CallHierarchyOptions) OrServerCapabilitiesCallHierarchyProvider {
	return OrServerCapabilitiesCallHierarchyProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyRegistrationOptions constructs the CallHierarchyRegistrationOptions variant of OrServerCapabilitiesCallHierarchyProvider.
func NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyRegistrationOptions(v CallHierarchyRegistrationOptions) OrServerCapabilitiesCallHierarchyProvider {
	return OrServerCapabilitiesCallHierarchyProvider{value: v, tag: 2}
}

// OrServerCapabilitiesLinkedEditingRangeProvider is a generated union type.
type OrServerCapabilitiesLinkedEditingRangeProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesLinkedEditingRangeProviderBoolean constructs the boolean variant of OrServerCapabilitiesLinkedEditingRangeProvider.
func NewOrServerCapabilitiesLinkedEditingRangeProviderBoolean(v bool) OrServerCapabilitiesLinkedEditingRangeProvider {
	return OrServerCapabilitiesLinkedEditingRangeProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeOptions constructs the LinkedEditingRangeOptions variant of OrServerCapabilitiesLinkedEditingRangeProvider.
func NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeOptions(v LinkedEditingRangeOptions) OrServerCapabilitiesLinkedEditingRangeProvider {
	return OrServerCapabilitiesLinkedEditingRangeProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeRegistrationOptions constructs the LinkedEditingRangeRegistrationOptions variant of OrServerCapabilitiesLinkedEditingRangeProvider.
func NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeRegistrationOptions(v LinkedEditingRangeRegistrationOptions) OrServerCapabilitiesLinkedEditingRangeProvider {
	return OrServerCapabilitiesLinkedEditingRangeProvider{value: v, tag: 2}
}

// OrServerCapabilitiesSemanticTokensProvider is a generated union type.
type OrServerCapabilitiesSemanticTokensProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensOptions constructs the SemanticTokensOptions variant of OrServerCapabilitiesSemanticTokensProvider.
func NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensOptions(v SemanticTokensOptions) OrServerCapabilitiesSemanticTokensProvider {
	return OrServerCapabilitiesSemanticTokensProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensRegistrationOptions constructs the SemanticTokensRegistrationOptions variant of OrServerCapabilitiesSemanticTokensProvider.
func NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensRegistrationOptions(v SemanticTokensRegistrationOptions) OrServerCapabilitiesSemanticTokensProvider {
	return OrServerCapabilitiesSemanticTokensProvider{value: v, tag: 1}
}

// OrServerCapabilitiesMonikerProvider is a generated union type.
type OrServerCapabilitiesMonikerProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesMonikerProviderBoolean constructs the boolean variant of OrServerCapabilitiesMonikerProvider.
func NewOrServerCapabilitiesMonikerProviderBoolean(v bool) OrServerCapabilitiesMonikerProvider {
	return OrServerCapabilitiesMonikerProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesMonikerProviderMonikerOptions constructs the MonikerOptions variant of OrServerCapabilitiesMonikerProvider.
func NewOrServerCapabilitiesMonikerProviderMonikerOptions(v MonikerOptions) OrServerCapabilitiesMonikerProvider {
	return OrServerCapabilitiesMonikerProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesMonikerProviderMonikerRegistrationOptions constructs the MonikerRegistrationOptions variant of OrServerCapabilitiesMonikerProvider.
func NewOrServerCapabilitiesMonikerProviderMonikerRegistrationOptions(v MonikerRegistrationOptions) OrServerCapabilitiesMonikerProvider {
	return OrServerCapabilitiesMonikerProvider{value: v, tag: 2}
}

// OrServerCapabilitiesTypeHierarchyProvider is a generated union type.
type OrServerCapabilitiesTypeHierarchyProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesTypeHierarchyProviderBoolean constructs the boolean variant of OrServerCapabilitiesTypeHierarchyProvider.
func NewOrServerCapabilitiesTypeHierarchyProviderBoolean(v bool) OrServerCapabilitiesTypeHierarchyProvider {
	return OrServerCapabilitiesTypeHierarchyProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyOptions constructs the TypeHierarchyOptions variant of OrServerCapabilitiesTypeHierarchyProvider.
func NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyOptions(v TypeHierarchyOptions) OrServerCapabilitiesTypeHierarchyProvider {
	return OrServerCapabilitiesTypeHierarchyProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyRegistrationOptions constructs the TypeHierarchyRegistrationOptions variant of OrServerCapabilitiesTypeHierarchyProvider.
func NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyRegistrationOptions(v TypeHierarchyRegistrationOptions) OrServerCapabilitiesTypeHierarchyProvider {
	return OrServerCapabilitiesTypeHierarchyProvider{value: v, tag: 2}
}

// OrServerCapabilitiesInlineValueProvider is a generated union type.
type OrServerCapabilitiesInlineValueProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesInlineValueProviderBoolean constructs the boolean variant of OrServerCapabilitiesInlineValueProvider.
func NewOrServerCapabilitiesInlineValueProviderBoolean(v bool) OrServerCapabilitiesInlineValueProvider {
	return OrServerCapabilitiesInlineValueProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesInlineValueProviderInlineValueOptions constructs the InlineValueOptions variant of OrServerCapabilitiesInlineValueProvider.
func NewOrServerCapabilitiesInlineValueProviderInlineValueOptions(v InlineValueOptions) OrServerCapabilitiesInlineValueProvider {
	return OrServerCapabilitiesInlineValueProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesInlineValueProviderInlineValueRegistrationOptions constructs the InlineValueRegistrationOptions variant of OrServerCapabilitiesInlineValueProvider.
func NewOrServerCapabilitiesInlineValueProviderInlineValueRegistrationOptions(v InlineValueRegistrationOptions) OrServerCapabilitiesInlineValueProvider {
	return OrServerCapabilitiesInlineValueProvider{value: v, tag: 2}
}

// OrServerCapabilitiesInlayHintProvider is a generated union type.
type OrServerCapabilitiesInlayHintProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesInlayHintProviderBoolean constructs the boolean variant of OrServerCapabilitiesInlayHintProvider.
func NewOrServerCapabilitiesInlayHintProviderBoolean(v bool) OrServerCapabilitiesInlayHintProvider {
	return OrServerCapabilitiesInlayHintProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesInlayHintProviderInlayHintOptions constructs the InlayHintOptions variant of OrServerCapabilitiesInlayHintProvider.
func NewOrServerCapabilitiesInlayHintProviderInlayHintOptions(v InlayHintOptions) OrServerCapabilitiesInlayHintProvider {
	return OrServerCapabilitiesInlayHintProvider{value: v, tag: 1}
}

// NewOrServerCapabilitiesInlayHintProviderInlayHintRegistrationOptions constructs the InlayHintRegistrationOptions variant of OrServerCapabilitiesInlayHintProvider.
func NewOrServerCapabilitiesInlayHintProviderInlayHintRegistrationOptions(v InlayHintRegistrationOptions) OrServerCapabilitiesInlayHintProvider {
	return OrServerCapabilitiesInlayHintProvider{value: v, tag: 2}
}

// OrServerCapabilitiesDiagnosticProvider is a generated union type.
type OrServerCapabilitiesDiagnosticProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesDiagnosticProviderDiagnosticOptions constructs the DiagnosticOptions variant of OrServerCapabilitiesDiagnosticProvider.
func NewOrServerCapabilitiesDiagnosticProviderDiagnosticOptions(v DiagnosticOptions) OrServerCapabilitiesDiagnosticProvider {
	return OrServerCapabilitiesDiagnosticProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesDiagnosticProviderDiagnosticRegistrationOptions constructs the DiagnosticRegistrationOptions variant of OrServerCapabilitiesDiagnosticProvider.
func NewOrServerCapabilitiesDiagnosticProviderDiagnosticRegistrationOptions(v DiagnosticRegistrationOptions) OrServerCapabilitiesDiagnosticProvider {
	return OrServerCapabilitiesDiagnosticProvider{value: v, tag: 1}
}

// OrServerCapabilitiesInlineCompletionProvider is a generated union type.
type OrServerCapabilitiesInlineCompletionProvider struct {
	value any
	tag   int
}

// NewOrServerCapabilitiesInlineCompletionProviderBoolean constructs the boolean variant of OrServerCapabilitiesInlineCompletionProvider.
func NewOrServerCapabilitiesInlineCompletionProviderBoolean(v bool) OrServerCapabilitiesInlineCompletionProvider {
	return OrServerCapabilitiesInlineCompletionProvider{value: v, tag: 0}
}

// NewOrServerCapabilitiesInlineCompletionProviderInlineCompletionOptions constructs the InlineCompletionOptions variant of OrServerCapabilitiesInlineCompletionProvider.
func NewOrServerCapabilitiesInlineCompletionProviderInlineCompletionOptions(v InlineCompletionOptions) OrServerCapabilitiesInlineCompletionProvider {
	return OrServerCapabilitiesInlineCompletionProvider{value: v, tag: 1}
}

// OrSignatureHelpActiveParameter is a generated union type.
type OrSignatureHelpActiveParameter struct {
	value any
	tag   int
}

// NewOrSignatureHelpActiveParameterUinteger constructs the uinteger variant of OrSignatureHelpActiveParameter.
func NewOrSignatureHelpActiveParameterUinteger(v uint32) OrSignatureHelpActiveParameter {
	return OrSignatureHelpActiveParameter{value: v, tag: 0}
}

// OrSignatureInformationDocumentation is a generated union type.
type OrSignatureInformationDocumentation struct {
	value any
	tag   int
}

// NewOrSignatureInformationDocumentationString constructs the string variant of OrSignatureInformationDocumentation.
func NewOrSignatureInformationDocumentationString(v string) OrSignatureInformationDocumentation {
	return OrSignatureInformationDocumentation{value: v, tag: 0}
}

// NewOrSignatureInformationDocumentationMarkupContent constructs the MarkupContent variant of OrSignatureInformationDocumentation.
func NewOrSignatureInformationDocumentationMarkupContent(v MarkupContent) OrSignatureInformationDocumentation {
	return OrSignatureInformationDocumentation{value: v, tag: 1}
}

// OrSignatureInformationActiveParameter is a generated union type.
type OrSignatureInformationActiveParameter struct {
	value any
	tag   int
}

// NewOrSignatureInformationActiveParameterUinteger constructs the uinteger variant of OrSignatureInformationActiveParameter.
func NewOrSignatureInformationActiveParameterUinteger(v uint32) OrSignatureInformationActiveParameter {
	return OrSignatureInformationActiveParameter{value: v, tag: 0}
}

// OrTextDocumentEditEditsElem is a generated union type.
type OrTextDocumentEditEditsElem struct {
	value any
	tag   int
}

// NewOrTextDocumentEditEditsElemTextEdit constructs the TextEdit variant of OrTextDocumentEditEditsElem.
func NewOrTextDocumentEditEditsElemTextEdit(v TextEdit) OrTextDocumentEditEditsElem {
	return OrTextDocumentEditEditsElem{value: v, tag: 0}
}

// NewOrTextDocumentEditEditsElemAnnotatedTextEdit constructs the AnnotatedTextEdit variant of OrTextDocumentEditEditsElem.
func NewOrTextDocumentEditEditsElemAnnotatedTextEdit(v AnnotatedTextEdit) OrTextDocumentEditEditsElem {
	return OrTextDocumentEditEditsElem{value: v, tag: 1}
}

// NewOrTextDocumentEditEditsElemSnippetTextEdit constructs the SnippetTextEdit variant of OrTextDocumentEditEditsElem.
func NewOrTextDocumentEditEditsElemSnippetTextEdit(v SnippetTextEdit) OrTextDocumentEditEditsElem {
	return OrTextDocumentEditEditsElem{value: v, tag: 2}
}

// OrTextDocumentRegistrationOptionsDocumentSelector is a generated union type.
type OrTextDocumentRegistrationOptionsDocumentSelector struct {
	value any
	tag   int
}

// NewOrTextDocumentRegistrationOptionsDocumentSelectorDocumentSelector constructs the DocumentSelector variant of OrTextDocumentRegistrationOptionsDocumentSelector.
func NewOrTextDocumentRegistrationOptionsDocumentSelectorDocumentSelector(v DocumentSelector) OrTextDocumentRegistrationOptionsDocumentSelector {
	return OrTextDocumentRegistrationOptionsDocumentSelector{value: v, tag: 0}
}

// OrTextDocumentSyncOptionsSave is a generated union type.
type OrTextDocumentSyncOptionsSave struct {
	value any
	tag   int
}

// NewOrTextDocumentSyncOptionsSaveBoolean constructs the boolean variant of OrTextDocumentSyncOptionsSave.
func NewOrTextDocumentSyncOptionsSaveBoolean(v bool) OrTextDocumentSyncOptionsSave {
	return OrTextDocumentSyncOptionsSave{value: v, tag: 0}
}

// NewOrTextDocumentSyncOptionsSaveSaveOptions constructs the SaveOptions variant of OrTextDocumentSyncOptionsSave.
func NewOrTextDocumentSyncOptionsSaveSaveOptions(v SaveOptions) OrTextDocumentSyncOptionsSave {
	return OrTextDocumentSyncOptionsSave{value: v, tag: 1}
}

// OrWorkspaceEditDocumentChangesElem is a generated union type.
type OrWorkspaceEditDocumentChangesElem struct {
	value any
	tag   int
}

// NewOrWorkspaceEditDocumentChangesElemTextDocumentEdit constructs the TextDocumentEdit variant of OrWorkspaceEditDocumentChangesElem.
func NewOrWorkspaceEditDocumentChangesElemTextDocumentEdit(v TextDocumentEdit) OrWorkspaceEditDocumentChangesElem {
	return OrWorkspaceEditDocumentChangesElem{value: v, tag: 0}
}

// NewOrWorkspaceEditDocumentChangesElemCreateFile constructs the CreateFile variant of OrWorkspaceEditDocumentChangesElem.
func NewOrWorkspaceEditDocumentChangesElemCreateFile(v CreateFile) OrWorkspaceEditDocumentChangesElem {
	return OrWorkspaceEditDocumentChangesElem{value: v, tag: 1}
}

// NewOrWorkspaceEditDocumentChangesElemRenameFile constructs the RenameFile variant of OrWorkspaceEditDocumentChangesElem.
func NewOrWorkspaceEditDocumentChangesElemRenameFile(v RenameFile) OrWorkspaceEditDocumentChangesElem {
	return OrWorkspaceEditDocumentChangesElem{value: v, tag: 2}
}

// NewOrWorkspaceEditDocumentChangesElemDeleteFile constructs the DeleteFile variant of OrWorkspaceEditDocumentChangesElem.
func NewOrWorkspaceEditDocumentChangesElemDeleteFile(v DeleteFile) OrWorkspaceEditDocumentChangesElem {
	return OrWorkspaceEditDocumentChangesElem{value: v, tag: 3}
}

// OrWorkspaceFoldersInitializeParamsWorkspaceFolders is a generated union type.
type OrWorkspaceFoldersInitializeParamsWorkspaceFolders struct {
	value any
	tag   int
}

// NewOrWorkspaceFoldersInitializeParamsWorkspaceFoldersArray0 constructs the WorkspaceFolder[] variant of OrWorkspaceFoldersInitializeParamsWorkspaceFolders.
func NewOrWorkspaceFoldersInitializeParamsWorkspaceFoldersArray0(v []WorkspaceFolder) OrWorkspaceFoldersInitializeParamsWorkspaceFolders {
	return OrWorkspaceFoldersInitializeParamsWorkspaceFolders{value: v, tag: 0}
}

// OrWorkspaceFoldersServerCapabilitiesChangeNotifications is a generated union type.
type OrWorkspaceFoldersServerCapabilitiesChangeNotifications struct {
	value any
	tag   int
}

// NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsString constructs the string variant of OrWorkspaceFoldersServerCapabilitiesChangeNotifications.
func NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsString(v string) OrWorkspaceFoldersServerCapabilitiesChangeNotifications {
	return OrWorkspaceFoldersServerCapabilitiesChangeNotifications{value: v, tag: 0}
}

// NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsBoolean constructs the boolean variant of OrWorkspaceFoldersServerCapabilitiesChangeNotifications.
func NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsBoolean(v bool) OrWorkspaceFoldersServerCapabilitiesChangeNotifications {
	return OrWorkspaceFoldersServerCapabilitiesChangeNotifications{value: v, tag: 1}
}

// OrWorkspaceFullDocumentDiagnosticReportVersion is a generated union type.
type OrWorkspaceFullDocumentDiagnosticReportVersion struct {
	value any
	tag   int
}

// NewOrWorkspaceFullDocumentDiagnosticReportVersionInteger constructs the integer variant of OrWorkspaceFullDocumentDiagnosticReportVersion.
func NewOrWorkspaceFullDocumentDiagnosticReportVersionInteger(v int32) OrWorkspaceFullDocumentDiagnosticReportVersion {
	return OrWorkspaceFullDocumentDiagnosticReportVersion{value: v, tag: 0}
}

// OrWorkspaceOptionsTextDocumentContent is a generated union type.
type OrWorkspaceOptionsTextDocumentContent struct {
	value any
	tag   int
}

// NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentOptions constructs the TextDocumentContentOptions variant of OrWorkspaceOptionsTextDocumentContent.
func NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentOptions(v TextDocumentContentOptions) OrWorkspaceOptionsTextDocumentContent {
	return OrWorkspaceOptionsTextDocumentContent{value: v, tag: 0}
}

// NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentRegistrationOptions constructs the TextDocumentContentRegistrationOptions variant of OrWorkspaceOptionsTextDocumentContent.
func NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentRegistrationOptions(v TextDocumentContentRegistrationOptions) OrWorkspaceOptionsTextDocumentContent {
	return OrWorkspaceOptionsTextDocumentContent{value: v, tag: 1}
}

// OrWorkspaceSymbolLocation is a generated union type.
type OrWorkspaceSymbolLocation struct {
	value any
	tag   int
}

// NewOrWorkspaceSymbolLocationLocation constructs the Location variant of OrWorkspaceSymbolLocation.
func NewOrWorkspaceSymbolLocationLocation(v Location) OrWorkspaceSymbolLocation {
	return OrWorkspaceSymbolLocation{value: v, tag: 0}
}

// NewOrWorkspaceSymbolLocationLocationUriOnly constructs the LocationUriOnly variant of OrWorkspaceSymbolLocation.
func NewOrWorkspaceSymbolLocationLocationUriOnly(v LocationUriOnly) OrWorkspaceSymbolLocation {
	return OrWorkspaceSymbolLocation{value: v, tag: 1}
}

// OrWorkspaceUnchangedDocumentDiagnosticReportVersion is a generated union type.
type OrWorkspaceUnchangedDocumentDiagnosticReportVersion struct {
	value any
	tag   int
}

// NewOrWorkspaceUnchangedDocumentDiagnosticReportVersionInteger constructs the integer variant of OrWorkspaceUnchangedDocumentDiagnosticReportVersion.
func NewOrWorkspaceUnchangedDocumentDiagnosticReportVersionInteger(v int32) OrWorkspaceUnchangedDocumentDiagnosticReportVersion {
	return OrWorkspaceUnchangedDocumentDiagnosticReportVersion{value: v, tag: 0}
}

// OrInitializeParamsProcessId is a generated union type.
type OrInitializeParamsProcessId struct {
	value any
	tag   int
}

// NewOrInitializeParamsProcessIdInteger constructs the integer variant of OrInitializeParamsProcessId.
func NewOrInitializeParamsProcessIdInteger(v int32) OrInitializeParamsProcessId {
	return OrInitializeParamsProcessId{value: v, tag: 0}
}

// OrInitializeParamsRootPath is a generated union type.
type OrInitializeParamsRootPath struct {
	value any
	tag   int
}

// NewOrInitializeParamsRootPathString constructs the string variant of OrInitializeParamsRootPath.
func NewOrInitializeParamsRootPathString(v string) OrInitializeParamsRootPath {
	return OrInitializeParamsRootPath{value: v, tag: 0}
}

// OrInitializeParamsRootUri is a generated union type.
type OrInitializeParamsRootUri struct {
	value any
	tag   int
}

// NewOrInitializeParamsRootUriDocumentURI constructs the DocumentUri variant of OrInitializeParamsRootUri.
func NewOrInitializeParamsRootUriDocumentURI(v DocumentURI) OrInitializeParamsRootUri {
	return OrInitializeParamsRootUri{value: v, tag: 0}
}

// OrResultCallHierarchyIncomingCalls is a generated union type.
type OrResultCallHierarchyIncomingCalls struct {
	value any
	tag   int
}

// NewOrResultCallHierarchyIncomingCallsArray0 constructs the CallHierarchyIncomingCall[] variant of OrResultCallHierarchyIncomingCalls.
func NewOrResultCallHierarchyIncomingCallsArray0(v []CallHierarchyIncomingCall) OrResultCallHierarchyIncomingCalls {
	return OrResultCallHierarchyIncomingCalls{value: v, tag: 0}
}

// OrResultCallHierarchyOutgoingCalls is a generated union type.
type OrResultCallHierarchyOutgoingCalls struct {
	value any
	tag   int
}

// NewOrResultCallHierarchyOutgoingCallsArray0 constructs the CallHierarchyOutgoingCall[] variant of OrResultCallHierarchyOutgoingCalls.
func NewOrResultCallHierarchyOutgoingCallsArray0(v []CallHierarchyOutgoingCall) OrResultCallHierarchyOutgoingCalls {
	return OrResultCallHierarchyOutgoingCalls{value: v, tag: 0}
}

// OrResultTextDocumentCodeAction is a generated union type.
type OrResultTextDocumentCodeAction struct {
	value any
	tag   int
}

// NewOrResultTextDocumentCodeActionArray0 constructs the Command | CodeAction[] variant of OrResultTextDocumentCodeAction.
func NewOrResultTextDocumentCodeActionArray0(v []OrResultTextDocumentCodeActionItem0Elem) OrResultTextDocumentCodeAction {
	return OrResultTextDocumentCodeAction{value: v, tag: 0}
}

// OrResultTextDocumentCodeActionItem0Elem is a generated union type.
type OrResultTextDocumentCodeActionItem0Elem struct {
	value any
	tag   int
}

// NewOrResultTextDocumentCodeActionItem0ElemCommand constructs the Command variant of OrResultTextDocumentCodeActionItem0Elem.
func NewOrResultTextDocumentCodeActionItem0ElemCommand(v Command) OrResultTextDocumentCodeActionItem0Elem {
	return OrResultTextDocumentCodeActionItem0Elem{value: v, tag: 0}
}

// NewOrResultTextDocumentCodeActionItem0ElemCodeAction constructs the CodeAction variant of OrResultTextDocumentCodeActionItem0Elem.
func NewOrResultTextDocumentCodeActionItem0ElemCodeAction(v CodeAction) OrResultTextDocumentCodeActionItem0Elem {
	return OrResultTextDocumentCodeActionItem0Elem{value: v, tag: 1}
}

// OrResultTextDocumentCodeLens is a generated union type.
type OrResultTextDocumentCodeLens struct {
	value any
	tag   int
}

// NewOrResultTextDocumentCodeLensArray0 constructs the CodeLens[] variant of OrResultTextDocumentCodeLens.
func NewOrResultTextDocumentCodeLensArray0(v []CodeLens) OrResultTextDocumentCodeLens {
	return OrResultTextDocumentCodeLens{value: v, tag: 0}
}

// AndRegOptTextDocumentColorPresentation is a generated intersection type.
type AndRegOptTextDocumentColorPresentation struct {
	WorkDoneProgressOptions
	TextDocumentRegistrationOptions
}

// OrResultTextDocumentCompletion is a generated union type.
type OrResultTextDocumentCompletion struct {
	value any
	tag   int
}

// NewOrResultTextDocumentCompletionArray0 constructs the CompletionItem[] variant of OrResultTextDocumentCompletion.
func NewOrResultTextDocumentCompletionArray0(v []CompletionItem) OrResultTextDocumentCompletion {
	return OrResultTextDocumentCompletion{value: v, tag: 0}
}

// NewOrResultTextDocumentCompletionCompletionList constructs the CompletionList variant of OrResultTextDocumentCompletion.
func NewOrResultTextDocumentCompletionCompletionList(v CompletionList) OrResultTextDocumentCompletion {
	return OrResultTextDocumentCompletion{value: v, tag: 1}
}

// OrResultTextDocumentDeclaration is a generated union type.
type OrResultTextDocumentDeclaration struct {
	value any
	tag   int
}

// NewOrResultTextDocumentDeclarationDeclaration constructs the Declaration variant of OrResultTextDocumentDeclaration.
func NewOrResultTextDocumentDeclarationDeclaration(v Declaration) OrResultTextDocumentDeclaration {
	return OrResultTextDocumentDeclaration{value: v, tag: 0}
}

// NewOrResultTextDocumentDeclarationArray1 constructs the DeclarationLink[] variant of OrResultTextDocumentDeclaration.
func NewOrResultTextDocumentDeclarationArray1(v []DeclarationLink) OrResultTextDocumentDeclaration {
	return OrResultTextDocumentDeclaration{value: v, tag: 1}
}

// OrResultTextDocumentDefinition is a generated union type.
type OrResultTextDocumentDefinition struct {
	value any
	tag   int
}

// NewOrResultTextDocumentDefinitionDefinition constructs the Definition variant of OrResultTextDocumentDefinition.
func NewOrResultTextDocumentDefinitionDefinition(v Definition) OrResultTextDocumentDefinition {
	return OrResultTextDocumentDefinition{value: v, tag: 0}
}

// NewOrResultTextDocumentDefinitionArray1 constructs the DefinitionLink[] variant of OrResultTextDocumentDefinition.
func NewOrResultTextDocumentDefinitionArray1(v []DefinitionLink) OrResultTextDocumentDefinition {
	return OrResultTextDocumentDefinition{value: v, tag: 1}
}

// OrResultTextDocumentDocumentHighlight is a generated union type.
type OrResultTextDocumentDocumentHighlight struct {
	value any
	tag   int
}

// NewOrResultTextDocumentDocumentHighlightArray0 constructs the DocumentHighlight[] variant of OrResultTextDocumentDocumentHighlight.
func NewOrResultTextDocumentDocumentHighlightArray0(v []DocumentHighlight) OrResultTextDocumentDocumentHighlight {
	return OrResultTextDocumentDocumentHighlight{value: v, tag: 0}
}

// OrResultTextDocumentDocumentLink is a generated union type.
type OrResultTextDocumentDocumentLink struct {
	value any
	tag   int
}

// NewOrResultTextDocumentDocumentLinkArray0 constructs the DocumentLink[] variant of OrResultTextDocumentDocumentLink.
func NewOrResultTextDocumentDocumentLinkArray0(v []DocumentLink) OrResultTextDocumentDocumentLink {
	return OrResultTextDocumentDocumentLink{value: v, tag: 0}
}

// OrResultTextDocumentDocumentSymbol is a generated union type.
type OrResultTextDocumentDocumentSymbol struct {
	value any
	tag   int
}

// NewOrResultTextDocumentDocumentSymbolArray0 constructs the SymbolInformation[] variant of OrResultTextDocumentDocumentSymbol.
func NewOrResultTextDocumentDocumentSymbolArray0(v []SymbolInformation) OrResultTextDocumentDocumentSymbol {
	return OrResultTextDocumentDocumentSymbol{value: v, tag: 0}
}

// NewOrResultTextDocumentDocumentSymbolArray1 constructs the DocumentSymbol[] variant of OrResultTextDocumentDocumentSymbol.
func NewOrResultTextDocumentDocumentSymbolArray1(v []DocumentSymbol) OrResultTextDocumentDocumentSymbol {
	return OrResultTextDocumentDocumentSymbol{value: v, tag: 1}
}

// OrResultTextDocumentFoldingRange is a generated union type.
type OrResultTextDocumentFoldingRange struct {
	value any
	tag   int
}

// NewOrResultTextDocumentFoldingRangeArray0 constructs the FoldingRange[] variant of OrResultTextDocumentFoldingRange.
func NewOrResultTextDocumentFoldingRangeArray0(v []FoldingRange) OrResultTextDocumentFoldingRange {
	return OrResultTextDocumentFoldingRange{value: v, tag: 0}
}

// OrResultTextDocumentFormatting is a generated union type.
type OrResultTextDocumentFormatting struct {
	value any
	tag   int
}

// NewOrResultTextDocumentFormattingArray0 constructs the TextEdit[] variant of OrResultTextDocumentFormatting.
func NewOrResultTextDocumentFormattingArray0(v []TextEdit) OrResultTextDocumentFormatting {
	return OrResultTextDocumentFormatting{value: v, tag: 0}
}

// OrResultTextDocumentHover is a generated union type.
type OrResultTextDocumentHover struct {
	value any
	tag   int
}

// NewOrResultTextDocumentHoverHover constructs the Hover variant of OrResultTextDocumentHover.
func NewOrResultTextDocumentHoverHover(v Hover) OrResultTextDocumentHover {
	return OrResultTextDocumentHover{value: v, tag: 0}
}

// OrResultTextDocumentImplementation is a generated union type.
type OrResultTextDocumentImplementation struct {
	value any
	tag   int
}

// NewOrResultTextDocumentImplementationDefinition constructs the Definition variant of OrResultTextDocumentImplementation.
func NewOrResultTextDocumentImplementationDefinition(v Definition) OrResultTextDocumentImplementation {
	return OrResultTextDocumentImplementation{value: v, tag: 0}
}

// NewOrResultTextDocumentImplementationArray1 constructs the DefinitionLink[] variant of OrResultTextDocumentImplementation.
func NewOrResultTextDocumentImplementationArray1(v []DefinitionLink) OrResultTextDocumentImplementation {
	return OrResultTextDocumentImplementation{value: v, tag: 1}
}

// OrResultTextDocumentInlayHint is a generated union type.
type OrResultTextDocumentInlayHint struct {
	value any
	tag   int
}

// NewOrResultTextDocumentInlayHintArray0 constructs the InlayHint[] variant of OrResultTextDocumentInlayHint.
func NewOrResultTextDocumentInlayHintArray0(v []InlayHint) OrResultTextDocumentInlayHint {
	return OrResultTextDocumentInlayHint{value: v, tag: 0}
}

// OrResultTextDocumentInlineCompletion is a generated union type.
type OrResultTextDocumentInlineCompletion struct {
	value any
	tag   int
}

// NewOrResultTextDocumentInlineCompletionInlineCompletionList constructs the InlineCompletionList variant of OrResultTextDocumentInlineCompletion.
func NewOrResultTextDocumentInlineCompletionInlineCompletionList(v InlineCompletionList) OrResultTextDocumentInlineCompletion {
	return OrResultTextDocumentInlineCompletion{value: v, tag: 0}
}

// NewOrResultTextDocumentInlineCompletionArray1 constructs the InlineCompletionItem[] variant of OrResultTextDocumentInlineCompletion.
func NewOrResultTextDocumentInlineCompletionArray1(v []InlineCompletionItem) OrResultTextDocumentInlineCompletion {
	return OrResultTextDocumentInlineCompletion{value: v, tag: 1}
}

// OrResultTextDocumentInlineValue is a generated union type.
type OrResultTextDocumentInlineValue struct {
	value any
	tag   int
}

// NewOrResultTextDocumentInlineValueArray0 constructs the InlineValue[] variant of OrResultTextDocumentInlineValue.
func NewOrResultTextDocumentInlineValueArray0(v []InlineValue) OrResultTextDocumentInlineValue {
	return OrResultTextDocumentInlineValue{value: v, tag: 0}
}

// OrResultTextDocumentLinkedEditingRange is a generated union type.
type OrResultTextDocumentLinkedEditingRange struct {
	value any
	tag   int
}

// NewOrResultTextDocumentLinkedEditingRangeLinkedEditingRanges constructs the LinkedEditingRanges variant of OrResultTextDocumentLinkedEditingRange.
func NewOrResultTextDocumentLinkedEditingRangeLinkedEditingRanges(v LinkedEditingRanges) OrResultTextDocumentLinkedEditingRange {
	return OrResultTextDocumentLinkedEditingRange{value: v, tag: 0}
}

// OrResultTextDocumentMoniker is a generated union type.
type OrResultTextDocumentMoniker struct {
	value any
	tag   int
}

// NewOrResultTextDocumentMonikerArray0 constructs the Moniker[] variant of OrResultTextDocumentMoniker.
func NewOrResultTextDocumentMonikerArray0(v []Moniker) OrResultTextDocumentMoniker {
	return OrResultTextDocumentMoniker{value: v, tag: 0}
}

// OrResultTextDocumentOnTypeFormatting is a generated union type.
type OrResultTextDocumentOnTypeFormatting struct {
	value any
	tag   int
}

// NewOrResultTextDocumentOnTypeFormattingArray0 constructs the TextEdit[] variant of OrResultTextDocumentOnTypeFormatting.
func NewOrResultTextDocumentOnTypeFormattingArray0(v []TextEdit) OrResultTextDocumentOnTypeFormatting {
	return OrResultTextDocumentOnTypeFormatting{value: v, tag: 0}
}

// OrResultTextDocumentPrepareCallHierarchy is a generated union type.
type OrResultTextDocumentPrepareCallHierarchy struct {
	value any
	tag   int
}

// NewOrResultTextDocumentPrepareCallHierarchyArray0 constructs the CallHierarchyItem[] variant of OrResultTextDocumentPrepareCallHierarchy.
func NewOrResultTextDocumentPrepareCallHierarchyArray0(v []CallHierarchyItem) OrResultTextDocumentPrepareCallHierarchy {
	return OrResultTextDocumentPrepareCallHierarchy{value: v, tag: 0}
}

// OrResultTextDocumentPrepareRename is a generated union type.
type OrResultTextDocumentPrepareRename struct {
	value any
	tag   int
}

// NewOrResultTextDocumentPrepareRenamePrepareRenameResult constructs the PrepareRenameResult variant of OrResultTextDocumentPrepareRename.
func NewOrResultTextDocumentPrepareRenamePrepareRenameResult(v PrepareRenameResult) OrResultTextDocumentPrepareRename {
	return OrResultTextDocumentPrepareRename{value: v, tag: 0}
}

// OrResultTextDocumentPrepareTypeHierarchy is a generated union type.
type OrResultTextDocumentPrepareTypeHierarchy struct {
	value any
	tag   int
}

// NewOrResultTextDocumentPrepareTypeHierarchyArray0 constructs the TypeHierarchyItem[] variant of OrResultTextDocumentPrepareTypeHierarchy.
func NewOrResultTextDocumentPrepareTypeHierarchyArray0(v []TypeHierarchyItem) OrResultTextDocumentPrepareTypeHierarchy {
	return OrResultTextDocumentPrepareTypeHierarchy{value: v, tag: 0}
}

// OrResultTextDocumentRangeFormatting is a generated union type.
type OrResultTextDocumentRangeFormatting struct {
	value any
	tag   int
}

// NewOrResultTextDocumentRangeFormattingArray0 constructs the TextEdit[] variant of OrResultTextDocumentRangeFormatting.
func NewOrResultTextDocumentRangeFormattingArray0(v []TextEdit) OrResultTextDocumentRangeFormatting {
	return OrResultTextDocumentRangeFormatting{value: v, tag: 0}
}

// OrResultTextDocumentRangesFormatting is a generated union type.
type OrResultTextDocumentRangesFormatting struct {
	value any
	tag   int
}

// NewOrResultTextDocumentRangesFormattingArray0 constructs the TextEdit[] variant of OrResultTextDocumentRangesFormatting.
func NewOrResultTextDocumentRangesFormattingArray0(v []TextEdit) OrResultTextDocumentRangesFormatting {
	return OrResultTextDocumentRangesFormatting{value: v, tag: 0}
}

// OrResultTextDocumentReferences is a generated union type.
type OrResultTextDocumentReferences struct {
	value any
	tag   int
}

// NewOrResultTextDocumentReferencesArray0 constructs the Location[] variant of OrResultTextDocumentReferences.
func NewOrResultTextDocumentReferencesArray0(v []Location) OrResultTextDocumentReferences {
	return OrResultTextDocumentReferences{value: v, tag: 0}
}

// OrResultTextDocumentRename is a generated union type.
type OrResultTextDocumentRename struct {
	value any
	tag   int
}

// NewOrResultTextDocumentRenameWorkspaceEdit constructs the WorkspaceEdit variant of OrResultTextDocumentRename.
func NewOrResultTextDocumentRenameWorkspaceEdit(v WorkspaceEdit) OrResultTextDocumentRename {
	return OrResultTextDocumentRename{value: v, tag: 0}
}

// OrResultTextDocumentSelectionRange is a generated union type.
type OrResultTextDocumentSelectionRange struct {
	value any
	tag   int
}

// NewOrResultTextDocumentSelectionRangeArray0 constructs the SelectionRange[] variant of OrResultTextDocumentSelectionRange.
func NewOrResultTextDocumentSelectionRangeArray0(v []SelectionRange) OrResultTextDocumentSelectionRange {
	return OrResultTextDocumentSelectionRange{value: v, tag: 0}
}

// OrResultTextDocumentSemanticTokensFull is a generated union type.
type OrResultTextDocumentSemanticTokensFull struct {
	value any
	tag   int
}

// NewOrResultTextDocumentSemanticTokensFullSemanticTokens constructs the SemanticTokens variant of OrResultTextDocumentSemanticTokensFull.
func NewOrResultTextDocumentSemanticTokensFullSemanticTokens(v SemanticTokens) OrResultTextDocumentSemanticTokensFull {
	return OrResultTextDocumentSemanticTokensFull{value: v, tag: 0}
}

// OrResultTextDocumentSemanticTokensFullDelta is a generated union type.
type OrResultTextDocumentSemanticTokensFullDelta struct {
	value any
	tag   int
}

// NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokens constructs the SemanticTokens variant of OrResultTextDocumentSemanticTokensFullDelta.
func NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokens(v SemanticTokens) OrResultTextDocumentSemanticTokensFullDelta {
	return OrResultTextDocumentSemanticTokensFullDelta{value: v, tag: 0}
}

// NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokensDelta constructs the SemanticTokensDelta variant of OrResultTextDocumentSemanticTokensFullDelta.
func NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokensDelta(v SemanticTokensDelta) OrResultTextDocumentSemanticTokensFullDelta {
	return OrResultTextDocumentSemanticTokensFullDelta{value: v, tag: 1}
}

// OrResultTextDocumentSemanticTokensRange is a generated union type.
type OrResultTextDocumentSemanticTokensRange struct {
	value any
	tag   int
}

// NewOrResultTextDocumentSemanticTokensRangeSemanticTokens constructs the SemanticTokens variant of OrResultTextDocumentSemanticTokensRange.
func NewOrResultTextDocumentSemanticTokensRangeSemanticTokens(v SemanticTokens) OrResultTextDocumentSemanticTokensRange {
	return OrResultTextDocumentSemanticTokensRange{value: v, tag: 0}
}

// OrResultTextDocumentSignatureHelp is a generated union type.
type OrResultTextDocumentSignatureHelp struct {
	value any
	tag   int
}

// NewOrResultTextDocumentSignatureHelpSignatureHelp constructs the SignatureHelp variant of OrResultTextDocumentSignatureHelp.
func NewOrResultTextDocumentSignatureHelpSignatureHelp(v SignatureHelp) OrResultTextDocumentSignatureHelp {
	return OrResultTextDocumentSignatureHelp{value: v, tag: 0}
}

// OrResultTextDocumentTypeDefinition is a generated union type.
type OrResultTextDocumentTypeDefinition struct {
	value any
	tag   int
}

// NewOrResultTextDocumentTypeDefinitionDefinition constructs the Definition variant of OrResultTextDocumentTypeDefinition.
func NewOrResultTextDocumentTypeDefinitionDefinition(v Definition) OrResultTextDocumentTypeDefinition {
	return OrResultTextDocumentTypeDefinition{value: v, tag: 0}
}

// NewOrResultTextDocumentTypeDefinitionArray1 constructs the DefinitionLink[] variant of OrResultTextDocumentTypeDefinition.
func NewOrResultTextDocumentTypeDefinitionArray1(v []DefinitionLink) OrResultTextDocumentTypeDefinition {
	return OrResultTextDocumentTypeDefinition{value: v, tag: 1}
}

// OrResultTextDocumentWillSaveWaitUntil is a generated union type.
type OrResultTextDocumentWillSaveWaitUntil struct {
	value any
	tag   int
}

// NewOrResultTextDocumentWillSaveWaitUntilArray0 constructs the TextEdit[] variant of OrResultTextDocumentWillSaveWaitUntil.
func NewOrResultTextDocumentWillSaveWaitUntilArray0(v []TextEdit) OrResultTextDocumentWillSaveWaitUntil {
	return OrResultTextDocumentWillSaveWaitUntil{value: v, tag: 0}
}

// OrResultTypeHierarchySubtypes is a generated union type.
type OrResultTypeHierarchySubtypes struct {
	value any
	tag   int
}

// NewOrResultTypeHierarchySubtypesArray0 constructs the TypeHierarchyItem[] variant of OrResultTypeHierarchySubtypes.
func NewOrResultTypeHierarchySubtypesArray0(v []TypeHierarchyItem) OrResultTypeHierarchySubtypes {
	return OrResultTypeHierarchySubtypes{value: v, tag: 0}
}

// OrResultTypeHierarchySupertypes is a generated union type.
type OrResultTypeHierarchySupertypes struct {
	value any
	tag   int
}

// NewOrResultTypeHierarchySupertypesArray0 constructs the TypeHierarchyItem[] variant of OrResultTypeHierarchySupertypes.
func NewOrResultTypeHierarchySupertypesArray0(v []TypeHierarchyItem) OrResultTypeHierarchySupertypes {
	return OrResultTypeHierarchySupertypes{value: v, tag: 0}
}

// OrResultWindowShowMessageRequest is a generated union type.
type OrResultWindowShowMessageRequest struct {
	value any
	tag   int
}

// NewOrResultWindowShowMessageRequestMessageActionItem constructs the MessageActionItem variant of OrResultWindowShowMessageRequest.
func NewOrResultWindowShowMessageRequestMessageActionItem(v MessageActionItem) OrResultWindowShowMessageRequest {
	return OrResultWindowShowMessageRequest{value: v, tag: 0}
}

// OrResultWorkspaceExecuteCommand is a generated union type.
type OrResultWorkspaceExecuteCommand struct {
	value any
	tag   int
}

// NewOrResultWorkspaceExecuteCommandLSPAny constructs the LSPAny variant of OrResultWorkspaceExecuteCommand.
func NewOrResultWorkspaceExecuteCommandLSPAny(v LSPAny) OrResultWorkspaceExecuteCommand {
	return OrResultWorkspaceExecuteCommand{value: v, tag: 0}
}

// OrResultWorkspaceSymbol is a generated union type.
type OrResultWorkspaceSymbol struct {
	value any
	tag   int
}

// NewOrResultWorkspaceSymbolArray0 constructs the SymbolInformation[] variant of OrResultWorkspaceSymbol.
func NewOrResultWorkspaceSymbolArray0(v []SymbolInformation) OrResultWorkspaceSymbol {
	return OrResultWorkspaceSymbol{value: v, tag: 0}
}

// NewOrResultWorkspaceSymbolArray1 constructs the WorkspaceSymbol[] variant of OrResultWorkspaceSymbol.
func NewOrResultWorkspaceSymbolArray1(v []WorkspaceSymbol) OrResultWorkspaceSymbol {
	return OrResultWorkspaceSymbol{value: v, tag: 1}
}

// OrResultWorkspaceWillCreateFiles is a generated union type.
type OrResultWorkspaceWillCreateFiles struct {
	value any
	tag   int
}

// NewOrResultWorkspaceWillCreateFilesWorkspaceEdit constructs the WorkspaceEdit variant of OrResultWorkspaceWillCreateFiles.
func NewOrResultWorkspaceWillCreateFilesWorkspaceEdit(v WorkspaceEdit) OrResultWorkspaceWillCreateFiles {
	return OrResultWorkspaceWillCreateFiles{value: v, tag: 0}
}

// OrResultWorkspaceWillDeleteFiles is a generated union type.
type OrResultWorkspaceWillDeleteFiles struct {
	value any
	tag   int
}

// NewOrResultWorkspaceWillDeleteFilesWorkspaceEdit constructs the WorkspaceEdit variant of OrResultWorkspaceWillDeleteFiles.
func NewOrResultWorkspaceWillDeleteFilesWorkspaceEdit(v WorkspaceEdit) OrResultWorkspaceWillDeleteFiles {
	return OrResultWorkspaceWillDeleteFiles{value: v, tag: 0}
}

// OrResultWorkspaceWillRenameFiles is a generated union type.
type OrResultWorkspaceWillRenameFiles struct {
	value any
	tag   int
}

// NewOrResultWorkspaceWillRenameFilesWorkspaceEdit constructs the WorkspaceEdit variant of OrResultWorkspaceWillRenameFiles.
func NewOrResultWorkspaceWillRenameFilesWorkspaceEdit(v WorkspaceEdit) OrResultWorkspaceWillRenameFiles {
	return OrResultWorkspaceWillRenameFiles{value: v, tag: 0}
}

// OrResultWorkspaceWorkspaceFolders is a generated union type.
type OrResultWorkspaceWorkspaceFolders struct {
	value any
	tag   int
}

// NewOrResultWorkspaceWorkspaceFoldersArray0 constructs the WorkspaceFolder[] variant of OrResultWorkspaceWorkspaceFolders.
func NewOrResultWorkspaceWorkspaceFoldersArray0(v []WorkspaceFolder) OrResultWorkspaceWorkspaceFolders {
	return OrResultWorkspaceWorkspaceFolders{value: v, tag: 0}
}

// A special text edit with an additional change annotation.
// @since 3.16.0.
// @since 3.16.0.
type AnnotatedTextEdit struct {
	// The range of the text document to be manipulated. To insert
	// text into a document create a range where start === end.
	Range Range `json:"range"`
	// The string to be inserted. For delete operations use an
	// empty string.
	NewText string `json:"newText"`
	// The actual identifier of the change annotation
	AnnotationID ChangeAnnotationIdentifier `json:"annotationId"`
}

// The parameters passed via an apply workspace edit request.
type ApplyWorkspaceEditParams struct {
	// An optional label of the workspace edit. This label is
	// presented in the user interface for example on an undo
	// stack to undo the workspace edit.
	Label Optional[string] `json:"label,omitzero"`
	// The edits to apply.
	Edit WorkspaceEdit `json:"edit"`
	// Additional data about the edit.
	// @since 3.18.0
	// @since 3.18.0
	Metadata Optional[WorkspaceEditMetadata] `json:"metadata,omitzero"`
}

// The result returned from the apply workspace edit request.
// @since 3.17 renamed from ApplyWorkspaceEditResponse
// @since 3.17 renamed from ApplyWorkspaceEditResponse
type ApplyWorkspaceEditResult struct {
	// Indicates whether the edit was applied or not.
	Applied bool `json:"applied"`
	// An optional textual description for why the edit was not applied.
	// This may be used by the server for diagnostic logging or to provide
	// a suitable error for a request that triggered the edit.
	FailureReason Optional[string] `json:"failureReason,omitzero"`
	// Depending on the client's failure handling strategy `failedChange` might
	// contain the index of the change that failed. This property is only available
	// if the client signals a `failureHandlingStrategy` in its client capabilities.
	FailedChange Optional[uint32] `json:"failedChange,omitzero"`
}

// A base for all symbol information.
type BaseSymbolInformation struct {
	// The name of this symbol.
	Name string `json:"name"`
	// The kind of this symbol.
	Kind SymbolKind `json:"kind"`
	// Tags for this symbol.
	// @since 3.16.0
	// @since 3.16.0
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// The name of the symbol containing this symbol. This information is for
	// user interface purposes (e.g. to render a qualifier in the user interface
	// if necessary). It can't be used to re-infer a hierarchy for the document
	// symbols.
	ContainerName Optional[string] `json:"containerName,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type CallHierarchyClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Represents an incoming call, e.g. a caller of a method or constructor.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyIncomingCall struct {
	// The item that makes the call.
	From CallHierarchyItem `json:"from"`
	// The ranges at which the calls appear. This is relative to the caller
	// denoted by CallHierarchyIncomingCall.from `this.from`.
	FromRanges []Range `json:"fromRanges"`
}

// The parameter of a `callHierarchy/incomingCalls` request.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyIncomingCallsParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	Item               CallHierarchyItem       `json:"item"`
}

// Represents programming constructs like functions or constructors in the context
// of call hierarchy.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyItem struct {
	// The name of this item.
	Name string `json:"name"`
	// The kind of this item.
	Kind SymbolKind `json:"kind"`
	// Tags for this item.
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// More detail for this item, e.g. the signature of a function.
	Detail Optional[string] `json:"detail,omitzero"`
	// The resource identifier of this item.
	URI DocumentURI `json:"uri"`
	// The range enclosing this symbol not including leading/trailing whitespace but everything else, e.g. comments and code.
	Range Range `json:"range"`
	// The range that should be selected and revealed when this symbol is being picked, e.g. the name of a function.
	// Must be contained by the CallHierarchyItem.range `range`.
	SelectionRange Range `json:"selectionRange"`
	// A data entry field that is preserved between a call hierarchy prepare and
	// incoming calls or outgoing calls requests.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Call hierarchy options used during static registration.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Represents an outgoing call, e.g. calling a getter from a method or a method from a constructor etc.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyOutgoingCall struct {
	// The item that is called.
	To CallHierarchyItem `json:"to"`
	// The range at which this item is called. This is the range relative to the caller, e.g the item
	// passed to CallHierarchyItemProvider.provideCallHierarchyOutgoingCalls `provideCallHierarchyOutgoingCalls`
	// and not CallHierarchyOutgoingCall.to `this.to`.
	FromRanges []Range `json:"fromRanges"`
}

// The parameter of a `callHierarchy/outgoingCalls` request.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyOutgoingCallsParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	Item               CallHierarchyItem       `json:"item"`
}

// The parameter of a `textDocument/prepareCallHierarchy` request.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyPrepareParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

// Call hierarchy options used during static or dynamic registration.
// @since 3.16.0
// @since 3.16.0
type CallHierarchyRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

type CancelParams struct {
	// The request id to cancel.
	ID OrCancelParamsId `json:"id"`
}

// Additional information that describes document changes.
// @since 3.16.0
// @since 3.16.0
type ChangeAnnotation struct {
	// A human-readable string describing the actual change. The string
	// is rendered prominent in the user interface.
	Label string `json:"label"`
	// A flag which indicates that user confirmation is needed
	// before applying the change.
	NeedsConfirmation Optional[bool] `json:"needsConfirmation,omitzero"`
	// A human-readable string which is rendered less prominent in
	// the user interface.
	Description Optional[string] `json:"description,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ChangeAnnotationsSupportOptions struct {
	// Whether the client groups edits with equal labels into tree nodes,
	// for instance all edits labelled with "Changes in Strings" would
	// be a tree node.
	GroupsOnLabel Optional[bool] `json:"groupsOnLabel,omitzero"`
}

// Defines the capabilities provided by the client.
type ClientCapabilities struct {
	// Workspace specific client capabilities.
	Workspace Optional[WorkspaceClientCapabilities] `json:"workspace,omitzero"`
	// Text document specific client capabilities.
	TextDocument Optional[TextDocumentClientCapabilities] `json:"textDocument,omitzero"`
	// Capabilities specific to the notebook document support.
	// @since 3.17.0
	// @since 3.17.0
	NotebookDocument Optional[NotebookDocumentClientCapabilities] `json:"notebookDocument,omitzero"`
	// Window specific client capabilities.
	Window Optional[WindowClientCapabilities] `json:"window,omitzero"`
	// General client capabilities.
	// @since 3.16.0
	// @since 3.16.0
	General Optional[GeneralClientCapabilities] `json:"general,omitzero"`
	// Experimental client capabilities.
	Experimental Optional[LSPAny] `json:"experimental,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCodeActionKindOptions struct {
	// The code action kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	ValueSet []CodeActionKind `json:"valueSet"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCodeActionLiteralOptions struct {
	// The code action kind is support with the following value
	// set.
	CodeActionKind ClientCodeActionKindOptions `json:"codeActionKind"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCodeActionResolveOptions struct {
	// The properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCodeLensResolveOptions struct {
	// The properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCompletionItemInsertTextModeOptions struct {
	ValueSet []InsertTextMode `json:"valueSet"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCompletionItemOptions struct {
	// Client supports snippets as insert text.
	// A snippet can define tab stops and placeholders with `$1`, `$2`
	// and `${3:foo`. `$0` defines the final tab stop, it defaults to
	// the end of the snippet. Placeholders with equal identifiers are linked,
	// that is typing in one will update others too.
	SnippetSupport Optional[bool] `json:"snippetSupport,omitzero"`
	// Client supports commit characters on a completion item.
	CommitCharactersSupport Optional[bool] `json:"commitCharactersSupport,omitzero"`
	// Client supports the following content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat Optional[[]MarkupKind] `json:"documentationFormat,omitzero"`
	// Client supports the deprecated property on a completion item.
	DeprecatedSupport Optional[bool] `json:"deprecatedSupport,omitzero"`
	// Client supports the preselect property on a completion item.
	PreselectSupport Optional[bool] `json:"preselectSupport,omitzero"`
	// Client supports the tag property on a completion item. Clients supporting
	// tags have to handle unknown tags gracefully. Clients especially need to
	// preserve unknown tags when sending a completion item back to the server in
	// a resolve call.
	// @since 3.15.0
	// @since 3.15.0
	TagSupport Optional[CompletionItemTagOptions] `json:"tagSupport,omitzero"`
	// Client support insert replace edit to control different behavior if a
	// completion item is inserted in the text or should replace text.
	// @since 3.16.0
	// @since 3.16.0
	InsertReplaceSupport Optional[bool] `json:"insertReplaceSupport,omitzero"`
	// Indicates which properties a client can resolve lazily on a completion
	// item. Before version 3.16.0 only the predefined properties `documentation`
	// and `details` could be resolved lazily.
	// @since 3.16.0
	// @since 3.16.0
	ResolveSupport Optional[ClientCompletionItemResolveOptions] `json:"resolveSupport,omitzero"`
	// The client supports the `insertTextMode` property on
	// a completion item to override the whitespace handling mode
	// as defined by the client (see `insertTextMode`).
	// @since 3.16.0
	// @since 3.16.0
	InsertTextModeSupport Optional[ClientCompletionItemInsertTextModeOptions] `json:"insertTextModeSupport,omitzero"`
	// The client has support for completion item label
	// details (see also `CompletionItemLabelDetails`).
	// @since 3.17.0
	// @since 3.17.0
	LabelDetailsSupport Optional[bool] `json:"labelDetailsSupport,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCompletionItemOptionsKind struct {
	// The completion item kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	// If this property is not present the client only supports
	// the completion items kinds from `Text` to `Reference` as defined in
	// the initial version of the protocol.
	ValueSet Optional[[]CompletionItemKind] `json:"valueSet,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientCompletionItemResolveOptions struct {
	// The properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// @since 3.18.0
// @since 3.18.0
type ClientDiagnosticsTagOptions struct {
	// The tags supported by the client.
	ValueSet []DiagnosticTag `json:"valueSet"`
}

// @since 3.18.0
// @since 3.18.0
type ClientFoldingRangeKindOptions struct {
	// The folding range kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	ValueSet Optional[[]FoldingRangeKind] `json:"valueSet,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientFoldingRangeOptions struct {
	// If set, the client signals that it supports setting collapsedText on
	// folding ranges to display custom labels instead of the default text.
	// @since 3.17.0
	// @since 3.17.0
	CollapsedText Optional[bool] `json:"collapsedText,omitzero"`
}

// Information about the client
// @since 3.15.0
// @since 3.18.0 ClientInfo type name added.
// @since 3.18.0 ClientInfo type name added.
type ClientInfo struct {
	// The name of the client as defined by the client.
	Name string `json:"name"`
	// The client's version as defined by the client.
	Version Optional[string] `json:"version,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientInlayHintResolveOptions struct {
	// The properties that a client can resolve lazily.
	Properties []string `json:"properties"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSemanticTokensRequestFullDelta struct {
	// The client will send the `textDocument/semanticTokens/full/delta` request if
	// the server provides a corresponding handler.
	Delta Optional[bool] `json:"delta,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSemanticTokensRequestOptions struct {
	// The client will send the `textDocument/semanticTokens/range` request if
	// the server provides a corresponding handler.
	Range Optional[OrClientSemanticTokensRequestOptionsRange] `json:"range,omitzero"`
	// The client will send the `textDocument/semanticTokens/full` request if
	// the server provides a corresponding handler.
	Full Optional[OrClientSemanticTokensRequestOptionsFull] `json:"full,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientShowMessageActionItemOptions struct {
	// Whether the client supports additional attributes which
	// are preserved and send back to the server in the
	// request's response.
	AdditionalPropertiesSupport Optional[bool] `json:"additionalPropertiesSupport,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSignatureInformationOptions struct {
	// Client supports the following content formats for the documentation
	// property. The order describes the preferred format of the client.
	DocumentationFormat Optional[[]MarkupKind] `json:"documentationFormat,omitzero"`
	// Client capabilities specific to parameter information.
	ParameterInformation Optional[ClientSignatureParameterInformationOptions] `json:"parameterInformation,omitzero"`
	// The client supports the `activeParameter` property on `SignatureInformation`
	// literal.
	// @since 3.16.0
	// @since 3.16.0
	ActiveParameterSupport Optional[bool] `json:"activeParameterSupport,omitzero"`
	// The client supports the `activeParameter` property on
	// `SignatureHelp`/`SignatureInformation` being set to `null` to
	// indicate that no parameter should be active.
	// @since 3.18.0
	// @since 3.18.0
	NoActiveParameterSupport Optional[bool] `json:"noActiveParameterSupport,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSignatureParameterInformationOptions struct {
	// The client supports processing label offsets instead of a
	// simple label string.
	// @since 3.14.0
	// @since 3.14.0
	LabelOffsetSupport Optional[bool] `json:"labelOffsetSupport,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSymbolKindOptions struct {
	// The symbol kind values the client supports. When this
	// property exists the client also guarantees that it will
	// handle values outside its set gracefully and falls back
	// to a default value when unknown.
	// If this property is not present the client only supports
	// the symbol kinds from `File` to `Array` as defined in
	// the initial version of the protocol.
	ValueSet Optional[[]SymbolKind] `json:"valueSet,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSymbolResolveOptions struct {
	// The properties that a client can resolve lazily. Usually
	// `location.range`
	Properties []string `json:"properties"`
}

// @since 3.18.0
// @since 3.18.0
type ClientSymbolTagOptions struct {
	// The tags supported by the client.
	ValueSet []SymbolTag `json:"valueSet"`
}

// A code action represents a change that can be performed in code, e.g. to fix a problem or
// to refactor code.
// A CodeAction must set either `edit` and/or a `command`. If both are supplied, the `edit` is applied first, then the `command` is executed.
type CodeAction struct {
	// A short, human-readable, title for this code action.
	Title string `json:"title"`
	// The kind of the code action.
	// Used to filter code actions.
	Kind Optional[CodeActionKind] `json:"kind,omitzero"`
	// The diagnostics that this code action resolves.
	Diagnostics Optional[[]Diagnostic] `json:"diagnostics,omitzero"`
	// Marks this as a preferred action. Preferred actions are used by the `auto fix` command and can be targeted
	// by keybindings.
	// A quick fix should be marked preferred if it properly addresses the underlying error.
	// A refactoring should be marked preferred if it is the most reasonable choice of actions to take.
	// @since 3.15.0
	// @since 3.15.0
	IsPreferred Optional[bool] `json:"isPreferred,omitzero"`
	// Marks that the code action cannot currently be applied.
	// Clients should follow the following guidelines regarding disabled code actions:
	// - Disabled code actions are not shown in automatic [lightbulbs](https://code.visualstudio.com/docs/editor/editingevolved#_code-action)
	// code action menus.
	// - Disabled actions are shown as faded out in the code action menu when the user requests a more specific type
	// of code action, such as refactorings.
	// - If the user has a [keybinding](https://code.visualstudio.com/docs/editor/refactoring#_keybindings-for-code-actions)
	// that auto applies a code action and only disabled code actions are returned, the client should show the user an
	// error message with `reason` in the editor.
	// @since 3.16.0
	// @since 3.16.0
	Disabled Optional[CodeActionDisabled] `json:"disabled,omitzero"`
	// The workspace edit this code action performs.
	Edit Optional[WorkspaceEdit] `json:"edit,omitzero"`
	// A command this code action executes. If a code action
	// provides an edit and a command, first the edit is
	// executed and then the command.
	Command Optional[Command] `json:"command,omitzero"`
	// A data entry field that is preserved on a code action between
	// a `textDocument/codeAction` and a `codeAction/resolve` request.
	// @since 3.16.0
	// @since 3.16.0
	Data Optional[LSPAny] `json:"data,omitzero"`
	// Tags for this code action.
	// @since 3.18.0
	// @since 3.18.0
	Tags Optional[[]CodeActionTag] `json:"tags,omitzero"`
}

// The Client Capabilities of a CodeActionRequest.
type CodeActionClientCapabilities struct {
	// Whether code action supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client support code action literals of type `CodeAction` as a valid
	// response of the `textDocument/codeAction` request. If the property is not
	// set the request can only return `Command` literals.
	// @since 3.8.0
	// @since 3.8.0
	CodeActionLiteralSupport Optional[ClientCodeActionLiteralOptions] `json:"codeActionLiteralSupport,omitzero"`
	// Whether code action supports the `isPreferred` property.
	// @since 3.15.0
	// @since 3.15.0
	IsPreferredSupport Optional[bool] `json:"isPreferredSupport,omitzero"`
	// Whether code action supports the `disabled` property.
	// @since 3.16.0
	// @since 3.16.0
	DisabledSupport Optional[bool] `json:"disabledSupport,omitzero"`
	// Whether code action supports the `data` property which is
	// preserved between a `textDocument/codeAction` and a
	// `codeAction/resolve` request.
	// @since 3.16.0
	// @since 3.16.0
	DataSupport Optional[bool] `json:"dataSupport,omitzero"`
	// Whether the client supports resolving additional code action
	// properties via a separate `codeAction/resolve` request.
	// @since 3.16.0
	// @since 3.16.0
	ResolveSupport Optional[ClientCodeActionResolveOptions] `json:"resolveSupport,omitzero"`
	// Whether the client honors the change annotations in
	// text edits and resource operations returned via the
	// `CodeAction#edit` property by for example presenting
	// the workspace edit in the user interface and asking
	// for confirmation.
	// @since 3.16.0
	// @since 3.16.0
	HonorsChangeAnnotations Optional[bool] `json:"honorsChangeAnnotations,omitzero"`
	// Whether the client supports documentation for a class of
	// code actions.
	// @since 3.18.0
	// @since 3.18.0
	DocumentationSupport Optional[bool] `json:"documentationSupport,omitzero"`
	// Client supports the tag property on a code action. Clients
	// supporting tags have to handle unknown tags gracefully.
	// @since 3.18.0
	// @since 3.18.0
	TagSupport Optional[CodeActionTagOptions] `json:"tagSupport,omitzero"`
}

// Contains additional diagnostic information about the context in which
// a CodeActionProvider.provideCodeActions code action is run.
type CodeActionContext struct {
	// An array of diagnostics known on the client side overlapping the range provided to the
	// `textDocument/codeAction` request. They are provided so that the server knows which
	// errors are currently presented to the user for the given range. There is no guarantee
	// that these accurately reflect the error state of the resource. The primary parameter
	// to compute code actions is the provided range.
	Diagnostics []Diagnostic `json:"diagnostics"`
	// Requested kind of actions to return.
	// Actions not of this kind are filtered out by the client before being shown. So servers
	// can omit computing them.
	Only Optional[[]CodeActionKind] `json:"only,omitzero"`
	// The reason why code actions were requested.
	// @since 3.17.0
	// @since 3.17.0
	TriggerKind Optional[CodeActionTriggerKind] `json:"triggerKind,omitzero"`
}

// Captures why the code action is currently disabled.
// @since 3.18.0
// @since 3.18.0
type CodeActionDisabled struct {
	// Human readable description of why the code action is currently disabled.
	// This is displayed in the code actions UI.
	Reason string `json:"reason"`
}

// Documentation for a class of code actions.
// @since 3.18.0
// @since 3.18.0
type CodeActionKindDocumentation struct {
	// The kind of the code action being documented.
	// If the kind is generic, such as `CodeActionKind.Refactor`, the documentation will be shown whenever any
	// refactorings are returned. If the kind if more specific, such as `CodeActionKind.RefactorExtract`, the
	// documentation will only be shown when extract refactoring code actions are returned.
	Kind CodeActionKind `json:"kind"`
	// Command that is ued to display the documentation to the user.
	// The title of this documentation code action is taken from {@linkcode Command.title
	Command Command `json:"command"`
}

// Provider options for a CodeActionRequest.
type CodeActionOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// CodeActionKinds that this server may return.
	// The list of kinds may be generic, such as `CodeActionKind.Refactor`, or the server
	// may list out every specific kind they provide.
	CodeActionKinds Optional[[]CodeActionKind] `json:"codeActionKinds,omitzero"`
	// Static documentation for a class of code actions.
	// Documentation from the provider should be shown in the code actions menu if either:
	// - Code actions of `kind` are requested by the editor. In this case, the editor will show the documentation that
	// most closely matches the requested code action kind. For example, if a provider has documentation for
	// both `Refactor` and `RefactorExtract`, when the user requests code actions for `RefactorExtract`,
	// the editor will use the documentation for `RefactorExtract` instead of the documentation for `Refactor`.
	// - Any code actions of `kind` are returned by the provider.
	// At most one documentation entry should be shown per provider.
	// @since 3.18.0
	// @since 3.18.0
	Documentation Optional[[]CodeActionKindDocumentation] `json:"documentation,omitzero"`
	// The server provides support to resolve additional
	// information for a code action.
	// @since 3.16.0
	// @since 3.16.0
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// The parameters of a CodeActionRequest.
type CodeActionParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The document in which the command was invoked.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The range for which the command was invoked.
	Range Range `json:"range"`
	// Context carrying additional information.
	Context CodeActionContext `json:"context"`
}

// Registration options for a CodeActionRequest.
type CodeActionRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// CodeActionKinds that this server may return.
	// The list of kinds may be generic, such as `CodeActionKind.Refactor`, or the server
	// may list out every specific kind they provide.
	CodeActionKinds Optional[[]CodeActionKind] `json:"codeActionKinds,omitzero"`
	// Static documentation for a class of code actions.
	// Documentation from the provider should be shown in the code actions menu if either:
	// - Code actions of `kind` are requested by the editor. In this case, the editor will show the documentation that
	// most closely matches the requested code action kind. For example, if a provider has documentation for
	// both `Refactor` and `RefactorExtract`, when the user requests code actions for `RefactorExtract`,
	// the editor will use the documentation for `RefactorExtract` instead of the documentation for `Refactor`.
	// - Any code actions of `kind` are returned by the provider.
	// At most one documentation entry should be shown per provider.
	// @since 3.18.0
	// @since 3.18.0
	Documentation Optional[[]CodeActionKindDocumentation] `json:"documentation,omitzero"`
	// The server provides support to resolve additional
	// information for a code action.
	// @since 3.16.0
	// @since 3.16.0
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type CodeActionTagOptions struct {
	// The tags supported by the client.
	ValueSet []CodeActionTag `json:"valueSet"`
}

// Structure to capture a description for an error code.
// @since 3.16.0
// @since 3.16.0
type CodeDescription struct {
	// An URI to open with more information about the diagnostic error.
	Href string `json:"href"`
}

// A code lens represents a Command command that should be shown along with
// source text, like the number of references, a way to run tests, etc.
// A code lens is _unresolved_ when no command is associated to it. For performance
// reasons the creation of a code lens and resolving should be done in two stages.
type CodeLens struct {
	// The range in which this code lens is valid. Should only span a single line.
	Range Range `json:"range"`
	// The command this code lens represents.
	Command Optional[Command] `json:"command,omitzero"`
	// A data entry field that is preserved on a code lens item between
	// a CodeLensRequest and a CodeLensResolveRequest
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// The client capabilities  of a CodeLensRequest.
type CodeLensClientCapabilities struct {
	// Whether code lens supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Whether the client supports resolving additional code lens
	// properties via a separate `codeLens/resolve` request.
	// @since 3.18.0
	// @since 3.18.0
	ResolveSupport Optional[ClientCodeLensResolveOptions] `json:"resolveSupport,omitzero"`
}

// Code Lens provider options of a CodeLensRequest.
type CodeLensOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Code lens has a resolve provider as well.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// The parameters of a CodeLensRequest.
type CodeLensParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The document to request code lens for.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Registration options for a CodeLensRequest.
type CodeLensRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// Code lens has a resolve provider as well.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type CodeLensWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from the
	// server to the client.
	// Note that this event is global and will force the client to refresh all
	// code lenses currently shown. It should be used with absolute care and is
	// useful for situation where a server for example detect a project wide
	// change that requires such a calculation.
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// Represents a color in RGBA space.
type Color struct {
	// The red component of this color in the range [0-1].
	Red float64 `json:"red"`
	// The green component of this color in the range [0-1].
	Green float64 `json:"green"`
	// The blue component of this color in the range [0-1].
	Blue float64 `json:"blue"`
	// The alpha component of this color in the range [0-1].
	Alpha float64 `json:"alpha"`
}

// Represents a color range from a document.
type ColorInformation struct {
	// The range in the document where this color appears.
	Range Range `json:"range"`
	// The actual color value for this color range.
	Color Color `json:"color"`
}

type ColorPresentation struct {
	// The label of this color presentation. It will be shown on the color
	// picker header. By default this is also the text that is inserted when selecting
	// this color presentation.
	Label string `json:"label"`
	// An TextEdit edit which is applied to a document when selecting
	// this presentation for the color.  When `falsy` the ColorPresentation.label label
	// is used.
	TextEdit Optional[TextEdit] `json:"textEdit,omitzero"`
	// An optional array of additional TextEdit text edits that are applied when
	// selecting this color presentation. Edits must not overlap with the main ColorPresentation.textEdit edit nor with themselves.
	AdditionalTextEdits Optional[[]TextEdit] `json:"additionalTextEdits,omitzero"`
}

// Parameters for a ColorPresentationRequest.
type ColorPresentationParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The color to request presentations for.
	Color Color `json:"color"`
	// The range where the color would be inserted. Serves as a context.
	Range Range `json:"range"`
}

// Represents a reference to a command. Provides a title which
// will be used to represent a command in the UI and, optionally,
// an array of arguments which will be passed to the command handler
// function when invoked.
type Command struct {
	// Title of the command, like `save`.
	Title string `json:"title"`
	// An optional tooltip.
	// @since 3.18.0
	// @since 3.18.0
	Tooltip Optional[string] `json:"tooltip,omitzero"`
	// The identifier of the actual command handler.
	Command string `json:"command"`
	// Arguments that the command handler should be
	// invoked with.
	Arguments Optional[[]LSPAny] `json:"arguments,omitzero"`
}

// Completion client capabilities
type CompletionClientCapabilities struct {
	// Whether completion supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports the following `CompletionItem` specific
	// capabilities.
	CompletionItem Optional[ClientCompletionItemOptions] `json:"completionItem,omitzero"`
	// The client supports the following completion item kinds.
	CompletionItemKind Optional[ClientCompletionItemOptionsKind] `json:"completionItemKind,omitzero"`
	// Defines how the client handles whitespace and indentation
	// when accepting a completion item that uses multi line
	// text in either `insertText` or `textEdit`.
	// @since 3.17.0
	// @since 3.17.0
	InsertTextMode Optional[InsertTextMode] `json:"insertTextMode,omitzero"`
	// The client supports to send additional context information for a
	// `textDocument/completion` request.
	ContextSupport Optional[bool] `json:"contextSupport,omitzero"`
	// The client supports the following `CompletionList` specific
	// capabilities.
	// @since 3.17.0
	// @since 3.17.0
	CompletionList Optional[CompletionListCapabilities] `json:"completionList,omitzero"`
}

// Contains additional information about the context in which a completion request is triggered.
type CompletionContext struct {
	// How the completion was triggered.
	TriggerKind CompletionTriggerKind `json:"triggerKind"`
	// The trigger character (a single character) that has trigger code complete.
	// Is undefined if `triggerKind !== CompletionTriggerKind.TriggerCharacter`
	TriggerCharacter Optional[string] `json:"triggerCharacter,omitzero"`
}

// A completion item represents a text snippet that is
// proposed to complete text that is being typed.
type CompletionItem struct {
	// The label of this completion item.
	// The label property is also by default the text that
	// is inserted when selecting this completion.
	// If label details are provided the label itself should
	// be an unqualified name of the completion item.
	Label string `json:"label"`
	// Additional details for the label
	// @since 3.17.0
	// @since 3.17.0
	LabelDetails Optional[CompletionItemLabelDetails] `json:"labelDetails,omitzero"`
	// The kind of this completion item. Based of the kind
	// an icon is chosen by the editor.
	Kind Optional[CompletionItemKind] `json:"kind,omitzero"`
	// Tags for this completion item.
	// @since 3.15.0
	// @since 3.15.0
	Tags Optional[[]CompletionItemTag] `json:"tags,omitzero"`
	// A human-readable string with additional information
	// about this item, like type or symbol information.
	Detail Optional[string] `json:"detail,omitzero"`
	// A human-readable string that represents a doc-comment.
	Documentation Optional[OrCompletionItemDocumentation] `json:"documentation,omitzero"`
	// Indicates if this item is deprecated.
	// @deprecated Use `tags` instead.
	// Deprecated: Use `tags` instead.
	Deprecated Optional[bool] `json:"deprecated,omitzero"`
	// Select this item when showing.
	// *Note* that only one completion item can be selected and that the
	// tool / client decides which item that is. The rule is that the *first*
	// item of those that match best is selected.
	Preselect Optional[bool] `json:"preselect,omitzero"`
	// A string that should be used when comparing this item
	// with other items. When `falsy` the CompletionItem.label label
	// is used.
	SortText Optional[string] `json:"sortText,omitzero"`
	// A string that should be used when filtering a set of
	// completion items. When `falsy` the CompletionItem.label label
	// is used.
	FilterText Optional[string] `json:"filterText,omitzero"`
	// A string that should be inserted into a document when selecting
	// this completion. When `falsy` the CompletionItem.label label
	// is used.
	// The `insertText` is subject to interpretation by the client side.
	// Some tools might not take the string literally. For example
	// VS Code when code complete is requested in this example
	// `con<cursor position>` and a completion item with an `insertText` of
	// `console` is provided it will only insert `sole`. Therefore it is
	// recommended to use `textEdit` instead since it avoids additional client
	// side interpretation.
	InsertText Optional[string] `json:"insertText,omitzero"`
	// The format of the insert text. The format applies to both the
	// `insertText` property and the `newText` property of a provided
	// `textEdit`. If omitted defaults to `InsertTextFormat.PlainText`.
	// Please note that the insertTextFormat doesn't apply to
	// `additionalTextEdits`.
	InsertTextFormat Optional[InsertTextFormat] `json:"insertTextFormat,omitzero"`
	// How whitespace and indentation is handled during completion
	// item insertion. If not provided the clients default value depends on
	// the `textDocument.completion.insertTextMode` client capability.
	// @since 3.16.0
	// @since 3.16.0
	InsertTextMode Optional[InsertTextMode] `json:"insertTextMode,omitzero"`
	// An TextEdit edit which is applied to a document when selecting
	// this completion. When an edit is provided the value of
	// CompletionItem.insertText insertText is ignored.
	// Most editors support two different operations when accepting a completion
	// item. One is to insert a completion text and the other is to replace an
	// existing text with a completion text. Since this can usually not be
	// predetermined by a server it can report both ranges. Clients need to
	// signal support for `InsertReplaceEdits` via the
	// `textDocument.completion.insertReplaceSupport` client capability
	// property.
	// *Note 1:* The text edit's range as well as both ranges from an insert
	// replace edit must be a [single line] and they must contain the position
	// at which completion has been requested.
	// *Note 2:* If an `InsertReplaceEdit` is returned the edit's insert range
	// must be a prefix of the edit's replace range, that means it must be
	// contained and starting at the same position.
	// @since 3.16.0 additional type `InsertReplaceEdit`
	// @since 3.16.0 additional type `InsertReplaceEdit`
	TextEdit Optional[OrCompletionItemTextEdit] `json:"textEdit,omitzero"`
	// The edit text used if the completion item is part of a CompletionList and
	// CompletionList defines an item default for the text edit range.
	// Clients will only honor this property if they opt into completion list
	// item defaults using the capability `completionList.itemDefaults`.
	// If not provided and a list's default range is provided the label
	// property is used as a text.
	// @since 3.17.0
	// @since 3.17.0
	TextEditText Optional[string] `json:"textEditText,omitzero"`
	// An optional array of additional TextEdit text edits that are applied when
	// selecting this completion. Edits must not overlap (including the same insert position)
	// with the main CompletionItem.textEdit edit nor with themselves.
	// Additional text edits should be used to change text unrelated to the current cursor position
	// (for example adding an import statement at the top of the file if the completion item will
	// insert an unqualified type).
	AdditionalTextEdits Optional[[]TextEdit] `json:"additionalTextEdits,omitzero"`
	// An optional set of characters that when pressed while this completion is active will accept it first and
	// then type that character. *Note* that all commit characters should have `length=1` and that superfluous
	// characters will be ignored.
	CommitCharacters Optional[[]string] `json:"commitCharacters,omitzero"`
	// An optional Command command that is executed *after* inserting this completion. *Note* that
	// additional modifications to the current document should be described with the
	// CompletionItem.additionalTextEdits additionalTextEdits-property.
	Command Optional[Command] `json:"command,omitzero"`
	// A data entry field that is preserved on a completion item between a
	// CompletionRequest and a CompletionResolveRequest.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Specifies how fields from a completion item should be combined with those
// from `completionList.itemDefaults`.
// If unspecified, all fields will be treated as ApplyKind.Replace.
// If a field's value is ApplyKind.Replace, the value from a completion item (if
// provided and not `null`) will always be used instead of the value from
// `completionItem.itemDefaults`.
// If a field's value is ApplyKind.Merge, the values will be merged using the rules
// defined against each field below.
// Servers are only allowed to return `applyKind` if the client
// signals support for this via the `completionList.applyKindSupport`
// capability.
// @since 3.18.0
// @since 3.18.0
type CompletionItemApplyKinds struct {
	// Specifies whether commitCharacters on a completion will replace or be
	// merged with those in `completionList.itemDefaults.commitCharacters`.
	// If ApplyKind.Replace, the commit characters from the completion item will
	// always be used unless not provided, in which case those from
	// `completionList.itemDefaults.commitCharacters` will be used. An
	// empty list can be used if a completion item does not have any commit
	// characters and also should not use those from
	// `completionList.itemDefaults.commitCharacters`.
	// If ApplyKind.Merge the commitCharacters for the completion will be the
	// union of all values in both `completionList.itemDefaults.commitCharacters`
	// and the completion's own `commitCharacters`.
	// @since 3.18.0
	// @since 3.18.0
	CommitCharacters Optional[ApplyKind] `json:"commitCharacters,omitzero"`
	// Specifies whether the `data` field on a completion will replace or
	// be merged with data from `completionList.itemDefaults.data`.
	// If ApplyKind.Replace, the data from the completion item will be used if
	// provided (and not `null`), otherwise
	// `completionList.itemDefaults.data` will be used. An empty object can
	// be used if a completion item does not have any data but also should
	// not use the value from `completionList.itemDefaults.data`.
	// If ApplyKind.Merge, a shallow merge will be performed between
	// `completionList.itemDefaults.data` and the completion's own data
	// using the following rules:
	// - If a completion's `data` field is not provided (or `null`), the
	// entire `data` field from `completionList.itemDefaults.data` will be
	// used as-is.
	// - If a completion's `data` field is provided, each field will
	// overwrite the field of the same name in
	// `completionList.itemDefaults.data` but no merging of nested fields
	// within that value will occur.
	// @since 3.18.0
	// @since 3.18.0
	Data Optional[ApplyKind] `json:"data,omitzero"`
}

// In many cases the items of an actual completion result share the same
// value for properties like `commitCharacters` or the range of a text
// edit. A completion list can therefore define item defaults which will
// be used if a completion item itself doesn't specify the value.
// If a completion list specifies a default value and a completion item
// also specifies a corresponding value, the rules for combining these are
// defined by `applyKinds` (if the client supports it), defaulting to
// ApplyKind.Replace.
// Servers are only allowed to return default values if the client
// signals support for this via the `completionList.itemDefaults`
// capability.
// @since 3.17.0
// @since 3.17.0
type CompletionItemDefaults struct {
	// A default commit character set.
	// @since 3.17.0
	// @since 3.17.0
	CommitCharacters Optional[[]string] `json:"commitCharacters,omitzero"`
	// A default edit range.
	// @since 3.17.0
	// @since 3.17.0
	EditRange Optional[OrCompletionItemDefaultsEditRange] `json:"editRange,omitzero"`
	// A default insert text format.
	// @since 3.17.0
	// @since 3.17.0
	InsertTextFormat Optional[InsertTextFormat] `json:"insertTextFormat,omitzero"`
	// A default insert text mode.
	// @since 3.17.0
	// @since 3.17.0
	InsertTextMode Optional[InsertTextMode] `json:"insertTextMode,omitzero"`
	// A default data value.
	// @since 3.17.0
	// @since 3.17.0
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Additional details for a completion item label.
// @since 3.17.0
// @since 3.17.0
type CompletionItemLabelDetails struct {
	// An optional string which is rendered less prominently directly after CompletionItem.label label,
	// without any spacing. Should be used for function signatures and type annotations.
	Detail Optional[string] `json:"detail,omitzero"`
	// An optional string which is rendered less prominently after CompletionItem.detail. Should be used
	// for fully qualified names and file paths.
	Description Optional[string] `json:"description,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type CompletionItemTagOptions struct {
	// The tags supported by the client.
	ValueSet []CompletionItemTag `json:"valueSet"`
}

// Represents a collection of CompletionItem completion items to be presented
// in the editor.
type CompletionList struct {
	// This list it not complete. Further typing results in recomputing this list.
	// Recomputed lists have all their items replaced (not appended) in the
	// incomplete completion sessions.
	IsIncomplete bool `json:"isIncomplete"`
	// In many cases the items of an actual completion result share the same
	// value for properties like `commitCharacters` or the range of a text
	// edit. A completion list can therefore define item defaults which will
	// be used if a completion item itself doesn't specify the value.
	// If a completion list specifies a default value and a completion item
	// also specifies a corresponding value, the rules for combining these are
	// defined by `applyKinds` (if the client supports it), defaulting to
	// ApplyKind.Replace.
	// Servers are only allowed to return default values if the client
	// signals support for this via the `completionList.itemDefaults`
	// capability.
	// @since 3.17.0
	// @since 3.17.0
	ItemDefaults Optional[CompletionItemDefaults] `json:"itemDefaults,omitzero"`
	// Specifies how fields from a completion item should be combined with those
	// from `completionList.itemDefaults`.
	// If unspecified, all fields will be treated as ApplyKind.Replace.
	// If a field's value is ApplyKind.Replace, the value from a completion item
	// (if provided and not `null`) will always be used instead of the value
	// from `completionItem.itemDefaults`.
	// If a field's value is ApplyKind.Merge, the values will be merged using
	// the rules defined against each field below.
	// Servers are only allowed to return `applyKind` if the client
	// signals support for this via the `completionList.applyKindSupport`
	// capability.
	// @since 3.18.0
	// @since 3.18.0
	ApplyKind Optional[CompletionItemApplyKinds] `json:"applyKind,omitzero"`
	// The completion items.
	Items []CompletionItem `json:"items"`
}

// The client supports the following `CompletionList` specific
// capabilities.
// @since 3.17.0
// @since 3.17.0
type CompletionListCapabilities struct {
	// The client supports the following itemDefaults on
	// a completion list.
	// The value lists the supported property names of the
	// `CompletionList.itemDefaults` object. If omitted
	// no properties are supported.
	// @since 3.17.0
	// @since 3.17.0
	ItemDefaults Optional[[]string] `json:"itemDefaults,omitzero"`
	// Specifies whether the client supports `CompletionList.applyKind` to
	// indicate how supported values from `completionList.itemDefaults`
	// and `completion` will be combined.
	// If a client supports `applyKind` it must support it for all fields
	// that it supports that are listed in `CompletionList.applyKind`. This
	// means when clients add support for new/future fields in completion
	// items the MUST also support merge for them if those fields are
	// defined in `CompletionList.applyKind`.
	// @since 3.18.0
	// @since 3.18.0
	ApplyKindSupport Optional[bool] `json:"applyKindSupport,omitzero"`
}

// Completion options.
type CompletionOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Most tools trigger completion request automatically without explicitly requesting
	// it using a keyboard shortcut (e.g. Ctrl+Space). Typically they do so when the user
	// starts to type an identifier. For example if the user types `c` in a JavaScript file
	// code complete will automatically pop up present `console` besides others as a
	// completion item. Characters that make up identifiers don't need to be listed here.
	// If code complete should automatically be trigger on characters not being valid inside
	// an identifier (for example `.` in JavaScript) list them in `triggerCharacters`.
	TriggerCharacters Optional[[]string] `json:"triggerCharacters,omitzero"`
	// The list of all possible characters that commit a completion. This field can be used
	// if clients don't support individual commit characters per completion item. See
	// `ClientCapabilities.textDocument.completion.completionItem.commitCharactersSupport`
	// If a server provides both `allCommitCharacters` and commit characters on an individual
	// completion item the ones on the completion item win.
	// @since 3.2.0
	// @since 3.2.0
	AllCommitCharacters Optional[[]string] `json:"allCommitCharacters,omitzero"`
	// The server provides support to resolve additional
	// information for a completion item.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
	// The server supports the following `CompletionItem` specific
	// capabilities.
	// @since 3.17.0
	// @since 3.17.0
	CompletionItem Optional[ServerCompletionItemOptions] `json:"completionItem,omitzero"`
}

// Completion parameters
type CompletionParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The completion context. This is only available it the client specifies
	// to send this using the client capability `textDocument.completion.contextSupport === true`
	Context Optional[CompletionContext] `json:"context,omitzero"`
}

// Registration options for a CompletionRequest.
type CompletionRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// Most tools trigger completion request automatically without explicitly requesting
	// it using a keyboard shortcut (e.g. Ctrl+Space). Typically they do so when the user
	// starts to type an identifier. For example if the user types `c` in a JavaScript file
	// code complete will automatically pop up present `console` besides others as a
	// completion item. Characters that make up identifiers don't need to be listed here.
	// If code complete should automatically be trigger on characters not being valid inside
	// an identifier (for example `.` in JavaScript) list them in `triggerCharacters`.
	TriggerCharacters Optional[[]string] `json:"triggerCharacters,omitzero"`
	// The list of all possible characters that commit a completion. This field can be used
	// if clients don't support individual commit characters per completion item. See
	// `ClientCapabilities.textDocument.completion.completionItem.commitCharactersSupport`
	// If a server provides both `allCommitCharacters` and commit characters on an individual
	// completion item the ones on the completion item win.
	// @since 3.2.0
	// @since 3.2.0
	AllCommitCharacters Optional[[]string] `json:"allCommitCharacters,omitzero"`
	// The server provides support to resolve additional
	// information for a completion item.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
	// The server supports the following `CompletionItem` specific
	// capabilities.
	// @since 3.17.0
	// @since 3.17.0
	CompletionItem Optional[ServerCompletionItemOptions] `json:"completionItem,omitzero"`
}

type ConfigurationItem struct {
	// The scope to get the configuration section for.
	ScopeURI Optional[string] `json:"scopeUri,omitzero"`
	// The configuration section asked for.
	Section Optional[string] `json:"section,omitzero"`
}

// The parameters of a configuration request.
type ConfigurationParams struct {
	Items []ConfigurationItem `json:"items"`
}

// Create file operation.
type CreateFile struct {
	// The resource operation kind.
	Kind string `json:"kind"`
	// An optional annotation identifier describing the operation.
	// @since 3.16.0
	// @since 3.16.0
	AnnotationID Optional[ChangeAnnotationIdentifier] `json:"annotationId,omitzero"`
	// The resource to create.
	URI DocumentURI `json:"uri"`
	// Additional options
	Options Optional[CreateFileOptions] `json:"options,omitzero"`
}

// Options to create a file.
type CreateFileOptions struct {
	// Overwrite existing file. Overwrite wins over `ignoreIfExists`
	Overwrite Optional[bool] `json:"overwrite,omitzero"`
	// Ignore if exists.
	IgnoreIfExists Optional[bool] `json:"ignoreIfExists,omitzero"`
}

// The parameters sent in notifications/requests for user-initiated creation of
// files.
// @since 3.16.0
// @since 3.16.0
type CreateFilesParams struct {
	// An array of all files/folders created in this operation.
	Files []FileCreate `json:"files"`
}

// @since 3.14.0
// @since 3.14.0
type DeclarationClientCapabilities struct {
	// Whether declaration supports dynamic registration. If this is set to `true`
	// the client supports the new `DeclarationRegistrationOptions` return value
	// for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports additional metadata in the form of declaration links.
	LinkSupport Optional[bool] `json:"linkSupport,omitzero"`
}

type DeclarationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type DeclarationParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

type DeclarationRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Client Capabilities for a DefinitionRequest.
type DefinitionClientCapabilities struct {
	// Whether definition supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports additional metadata in the form of definition links.
	// @since 3.14.0
	// @since 3.14.0
	LinkSupport Optional[bool] `json:"linkSupport,omitzero"`
}

// Server Capabilities for a DefinitionRequest.
type DefinitionOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a DefinitionRequest.
type DefinitionParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

// Registration options for a DefinitionRequest.
type DefinitionRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// Delete file operation
type DeleteFile struct {
	// The resource operation kind.
	Kind string `json:"kind"`
	// An optional annotation identifier describing the operation.
	// @since 3.16.0
	// @since 3.16.0
	AnnotationID Optional[ChangeAnnotationIdentifier] `json:"annotationId,omitzero"`
	// The file to delete.
	URI DocumentURI `json:"uri"`
	// Delete options.
	Options Optional[DeleteFileOptions] `json:"options,omitzero"`
}

// Delete file options
type DeleteFileOptions struct {
	// Delete the content recursively if a folder is denoted.
	Recursive Optional[bool] `json:"recursive,omitzero"`
	// Ignore the operation if the file doesn't exist.
	IgnoreIfNotExists Optional[bool] `json:"ignoreIfNotExists,omitzero"`
}

// The parameters sent in notifications/requests for user-initiated deletes of
// files.
// @since 3.16.0
// @since 3.16.0
type DeleteFilesParams struct {
	// An array of all files/folders deleted in this operation.
	Files []FileDelete `json:"files"`
}

// Represents a diagnostic, such as a compiler error or warning. Diagnostic objects
// are only valid in the scope of a resource.
type Diagnostic struct {
	// The range at which the message applies
	Range Range `json:"range"`
	// The diagnostic's severity. To avoid interpretation mismatches when a
	// server is used with different clients it is highly recommended that servers
	// always provide a severity value.
	Severity Optional[DiagnosticSeverity] `json:"severity,omitzero"`
	// The diagnostic's code, which usually appear in the user interface.
	Code Optional[OrDiagnosticCode] `json:"code,omitzero"`
	// An optional property to describe the error code.
	// Requires the code field (above) to be present/not null.
	// @since 3.16.0
	// @since 3.16.0
	CodeDescription Optional[CodeDescription] `json:"codeDescription,omitzero"`
	// A human-readable string describing the source of this
	// diagnostic, e.g. 'typescript' or 'super lint'. It usually
	// appears in the user interface.
	Source Optional[string] `json:"source,omitzero"`
	// The diagnostic's message. It usually appears in the user interface.
	// @since 3.18.0 - support for MarkupContent. This is guarded by the client
	// capability `textDocument.diagnostic.markupMessageSupport`.
	// @since 3.18.0 - support for MarkupContent. This is guarded by the client
	// capability `textDocument.diagnostic.markupMessageSupport`.
	Message OrDiagnosticMessage `json:"message"`
	// Additional metadata about the diagnostic.
	// @since 3.15.0
	// @since 3.15.0
	Tags Optional[[]DiagnosticTag] `json:"tags,omitzero"`
	// An array of related diagnostic information, e.g. when symbol-names within
	// a scope collide all definitions can be marked via this property.
	RelatedInformation Optional[[]DiagnosticRelatedInformation] `json:"relatedInformation,omitzero"`
	// A data entry field that is preserved between a `textDocument/publishDiagnostics`
	// notification and `textDocument/codeAction` request.
	// @since 3.16.0
	// @since 3.16.0
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Client capabilities specific to diagnostic pull requests.
// @since 3.17.0
// @since 3.17.0
type DiagnosticClientCapabilities struct {
	// Whether the clients accepts diagnostics with related information.
	RelatedInformation Optional[bool] `json:"relatedInformation,omitzero"`
	// Client supports the tag property to provide meta data about a diagnostic.
	// Clients supporting tags have to handle unknown tags gracefully.
	// @since 3.15.0
	// @since 3.15.0
	TagSupport Optional[ClientDiagnosticsTagOptions] `json:"tagSupport,omitzero"`
	// Client supports a codeDescription property
	// @since 3.16.0
	// @since 3.16.0
	CodeDescriptionSupport Optional[bool] `json:"codeDescriptionSupport,omitzero"`
	// Whether code action supports the `data` property which is
	// preserved between a `textDocument/publishDiagnostics` and
	// `textDocument/codeAction` request.
	// @since 3.16.0
	// @since 3.16.0
	DataSupport Optional[bool] `json:"dataSupport,omitzero"`
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Whether the clients supports related documents for document diagnostic pulls.
	RelatedDocumentSupport Optional[bool] `json:"relatedDocumentSupport,omitzero"`
	// Whether the client supports `MarkupContent` in diagnostic messages.
	// @since 3.18.0
	// @since 3.18.0
	MarkupMessageSupport Optional[bool] `json:"markupMessageSupport,omitzero"`
}

// Diagnostic options.
// @since 3.17.0
// @since 3.17.0
type DiagnosticOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// An optional identifier under which the diagnostics are
	// managed by the client.
	Identifier Optional[string] `json:"identifier,omitzero"`
	// Whether the language has inter file dependencies meaning that
	// editing code in one file can result in a different diagnostic
	// set in another file. Inter file dependencies are common for
	// most programming languages and typically uncommon for linters.
	InterFileDependencies bool `json:"interFileDependencies"`
	// The server provides support for workspace diagnostics as well.
	WorkspaceDiagnostics bool `json:"workspaceDiagnostics"`
}

// Diagnostic registration options.
// @since 3.17.0
// @since 3.17.0
type DiagnosticRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// An optional identifier under which the diagnostics are
	// managed by the client.
	Identifier Optional[string] `json:"identifier,omitzero"`
	// Whether the language has inter file dependencies meaning that
	// editing code in one file can result in a different diagnostic
	// set in another file. Inter file dependencies are common for
	// most programming languages and typically uncommon for linters.
	InterFileDependencies bool `json:"interFileDependencies"`
	// The server provides support for workspace diagnostics as well.
	WorkspaceDiagnostics bool `json:"workspaceDiagnostics"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Represents a related message and source code location for a diagnostic. This should be
// used to point to code locations that cause or related to a diagnostics, e.g when duplicating
// a symbol in a scope.
type DiagnosticRelatedInformation struct {
	// The location of this related diagnostic information.
	Location Location `json:"location"`
	// The message of this related diagnostic information.
	Message string `json:"message"`
}

// Cancellation data returned from a diagnostic request.
// @since 3.17.0
// @since 3.17.0
type DiagnosticServerCancellationData struct {
	RetriggerRequest bool `json:"retriggerRequest"`
}

// Workspace client capabilities specific to diagnostic pull requests.
// @since 3.17.0
// @since 3.17.0
type DiagnosticWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from
	// the server to the client.
	// Note that this event is global and will force the client to refresh all
	// pulled diagnostics currently shown. It should be used with absolute care and
	// is useful for situation where a server for example detects a project wide
	// change that requires such a calculation.
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// General diagnostics capabilities for pull and push model.
type DiagnosticsCapabilities struct {
	// Whether the clients accepts diagnostics with related information.
	RelatedInformation Optional[bool] `json:"relatedInformation,omitzero"`
	// Client supports the tag property to provide meta data about a diagnostic.
	// Clients supporting tags have to handle unknown tags gracefully.
	// @since 3.15.0
	// @since 3.15.0
	TagSupport Optional[ClientDiagnosticsTagOptions] `json:"tagSupport,omitzero"`
	// Client supports a codeDescription property
	// @since 3.16.0
	// @since 3.16.0
	CodeDescriptionSupport Optional[bool] `json:"codeDescriptionSupport,omitzero"`
	// Whether code action supports the `data` property which is
	// preserved between a `textDocument/publishDiagnostics` and
	// `textDocument/codeAction` request.
	// @since 3.16.0
	// @since 3.16.0
	DataSupport Optional[bool] `json:"dataSupport,omitzero"`
}

type DidChangeConfigurationClientCapabilities struct {
	// Did change configuration notification supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// The parameters of a change configuration notification.
type DidChangeConfigurationParams struct {
	// The actual changed settings
	Settings LSPAny `json:"settings"`
}

type DidChangeConfigurationRegistrationOptions struct {
	Section Optional[OrDidChangeConfigurationRegistrationOptionsSection] `json:"section,omitzero"`
}

// The params sent in a change notebook document notification.
// @since 3.17.0
// @since 3.17.0
type DidChangeNotebookDocumentParams struct {
	// The notebook document that did change. The version number points
	// to the version after all provided changes have been applied. If
	// only the text document content of a cell changes the notebook version
	// doesn't necessarily have to change.
	NotebookDocument VersionedNotebookDocumentIdentifier `json:"notebookDocument"`
	// The actual changes to the notebook document.
	// The changes describe single state changes to the notebook document.
	// So if there are two changes c1 (at array index 0) and c2 (at array
	// index 1) for a notebook in state S then c1 moves the notebook from
	// S to S' and c2 from S' to S''. So c1 is computed on the state S and
	// c2 is computed on the state S'.
	// To mirror the content of a notebook using change events use the following approach:
	// - start with the same initial content
	// - apply the 'notebookDocument/didChange' notifications in the order you receive them.
	// - apply the `NotebookChangeEvent`s in a single notification in the order
	// you receive them.
	Change NotebookDocumentChangeEvent `json:"change"`
}

// The change text document notification's parameters.
type DidChangeTextDocumentParams struct {
	// The document that did change. The version number points
	// to the version after all provided content changes have
	// been applied.
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	// The actual content changes. The content changes describe single state changes
	// to the document. So if there are two content changes c1 (at array index 0) and
	// c2 (at array index 1) for a document in state S then c1 moves the document from
	// S to S' and c2 from S' to S''. So c1 is computed on the state S and c2 is computed
	// on the state S'.
	// To mirror the content of a document using change events use the following approach:
	// - start with the same initial content
	// - apply the 'textDocument/didChange' notifications in the order you receive them.
	// - apply the `TextDocumentContentChangeEvent`s in a single notification in the order
	// you receive them.
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

type DidChangeWatchedFilesClientCapabilities struct {
	// Did change watched files notification supports dynamic registration. Please note
	// that the current protocol doesn't support static configuration for file changes
	// from the server side.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Whether the client has support for  RelativePattern relative pattern
	// or not.
	// @since 3.17.0
	// @since 3.17.0
	RelativePatternSupport Optional[bool] `json:"relativePatternSupport,omitzero"`
}

// The watched files change notification's parameters.
type DidChangeWatchedFilesParams struct {
	// The actual file events.
	Changes []FileEvent `json:"changes"`
}

// Describe options to be used when registered for text document change events.
type DidChangeWatchedFilesRegistrationOptions struct {
	// The watchers to register.
	Watchers []FileSystemWatcher `json:"watchers"`
}

// The parameters of a `workspace/didChangeWorkspaceFolders` notification.
type DidChangeWorkspaceFoldersParams struct {
	// The actual workspace folder change event.
	Event WorkspaceFoldersChangeEvent `json:"event"`
}

// The params sent in a close notebook document notification.
// @since 3.17.0
// @since 3.17.0
type DidCloseNotebookDocumentParams struct {
	// The notebook document that got closed.
	NotebookDocument NotebookDocumentIdentifier `json:"notebookDocument"`
	// The text documents that represent the content
	// of a notebook cell that got closed.
	CellTextDocuments []TextDocumentIdentifier `json:"cellTextDocuments"`
}

// The parameters sent in a close text document notification
type DidCloseTextDocumentParams struct {
	// The document that was closed.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// The params sent in an open notebook document notification.
// @since 3.17.0
// @since 3.17.0
type DidOpenNotebookDocumentParams struct {
	// The notebook document that got opened.
	NotebookDocument NotebookDocument `json:"notebookDocument"`
	// The text documents that represent the content
	// of a notebook cell.
	CellTextDocuments []TextDocumentItem `json:"cellTextDocuments"`
}

// The parameters sent in an open text document notification
type DidOpenTextDocumentParams struct {
	// The document that was opened.
	TextDocument TextDocumentItem `json:"textDocument"`
}

// The params sent in a save notebook document notification.
// @since 3.17.0
// @since 3.17.0
type DidSaveNotebookDocumentParams struct {
	// The notebook document that got saved.
	NotebookDocument NotebookDocumentIdentifier `json:"notebookDocument"`
}

// The parameters sent in a save text document notification
type DidSaveTextDocumentParams struct {
	// The document that was saved.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// Optional the content when saved. Depends on the includeText value
	// when the save notification was requested.
	Text Optional[string] `json:"text,omitzero"`
}

type DocumentColorClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `DocumentColorRegistrationOptions` return value
	// for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

type DocumentColorOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a DocumentColorRequest.
type DocumentColorParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DocumentColorRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Parameters of the document diagnostic request.
// @since 3.17.0
// @since 3.17.0
type DocumentDiagnosticParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The additional identifier  provided during registration.
	Identifier Optional[string] `json:"identifier,omitzero"`
	// The result id of a previous response if provided.
	PreviousResultID Optional[string] `json:"previousResultId,omitzero"`
}

// A partial result for a document diagnostic report.
// @since 3.17.0
// @since 3.17.0
type DocumentDiagnosticReportPartialResult struct {
	RelatedDocuments map[DocumentURI]any `json:"relatedDocuments"`
}

// Client capabilities of a DocumentFormattingRequest.
type DocumentFormattingClientCapabilities struct {
	// Whether formatting supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Provider options for a DocumentFormattingRequest.
type DocumentFormattingOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// The parameters of a DocumentFormattingRequest.
type DocumentFormattingParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The format options.
	Options FormattingOptions `json:"options"`
}

// Registration options for a DocumentFormattingRequest.
type DocumentFormattingRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// A document highlight is a range inside a text document which deserves
// special attention. Usually a document highlight is visualized by changing
// the background color of its range.
type DocumentHighlight struct {
	// The range this highlight applies to.
	Range Range `json:"range"`
	// The highlight kind, default is DocumentHighlightKind.Text text.
	Kind Optional[DocumentHighlightKind] `json:"kind,omitzero"`
}

// Client Capabilities for a DocumentHighlightRequest.
type DocumentHighlightClientCapabilities struct {
	// Whether document highlight supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Provider options for a DocumentHighlightRequest.
type DocumentHighlightOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a DocumentHighlightRequest.
type DocumentHighlightParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

// Registration options for a DocumentHighlightRequest.
type DocumentHighlightRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// A document link is a range in a text document that links to an internal or external resource, like another
// text document or a web site.
type DocumentLink struct {
	// The range this link applies to.
	Range Range `json:"range"`
	// The uri this link points to. If missing a resolve request is sent later.
	Target Optional[string] `json:"target,omitzero"`
	// The tooltip text when you hover over this link.
	// If a tooltip is provided, is will be displayed in a string that includes instructions on how to
	// trigger the link, such as `{0 (ctrl + click)`. The specific instructions vary depending on OS,
	// user settings, and localization.
	// @since 3.15.0
	// @since 3.15.0
	Tooltip Optional[string] `json:"tooltip,omitzero"`
	// A data entry field that is preserved on a document link between a
	// DocumentLinkRequest and a DocumentLinkResolveRequest.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// The client capabilities of a DocumentLinkRequest.
type DocumentLinkClientCapabilities struct {
	// Whether document link supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Whether the client supports the `tooltip` property on `DocumentLink`.
	// @since 3.15.0
	// @since 3.15.0
	TooltipSupport Optional[bool] `json:"tooltipSupport,omitzero"`
}

// Provider options for a DocumentLinkRequest.
type DocumentLinkOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Document links have a resolve provider as well.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// The parameters of a DocumentLinkRequest.
type DocumentLinkParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The document to provide document links for.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Registration options for a DocumentLinkRequest.
type DocumentLinkRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// Document links have a resolve provider as well.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// Client capabilities of a DocumentOnTypeFormattingRequest.
type DocumentOnTypeFormattingClientCapabilities struct {
	// Whether on type formatting supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Provider options for a DocumentOnTypeFormattingRequest.
type DocumentOnTypeFormattingOptions struct {
	// A character on which formatting should be triggered, like `{`.
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`
	// More trigger characters.
	MoreTriggerCharacter Optional[[]string] `json:"moreTriggerCharacter,omitzero"`
}

// The parameters of a DocumentOnTypeFormattingRequest.
type DocumentOnTypeFormattingParams struct {
	// The document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position around which the on type formatting should happen.
	// This is not necessarily the exact position where the character denoted
	// by the property `ch` got typed.
	Position Position `json:"position"`
	// The character that has been typed that triggered the formatting
	// on type request. That is not necessarily the last character that
	// got inserted into the document since the client could auto insert
	// characters as well (e.g. like automatic brace completion).
	Ch string `json:"ch"`
	// The formatting options.
	Options FormattingOptions `json:"options"`
}

// Registration options for a DocumentOnTypeFormattingRequest.
type DocumentOnTypeFormattingRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// A character on which formatting should be triggered, like `{`.
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`
	// More trigger characters.
	MoreTriggerCharacter Optional[[]string] `json:"moreTriggerCharacter,omitzero"`
}

// Client capabilities of a DocumentRangeFormattingRequest.
type DocumentRangeFormattingClientCapabilities struct {
	// Whether range formatting supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Whether the client supports formatting multiple ranges at once.
	// @since 3.18.0
	// @since 3.18.0
	RangesSupport Optional[bool] `json:"rangesSupport,omitzero"`
}

// Provider options for a DocumentRangeFormattingRequest.
type DocumentRangeFormattingOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Whether the server supports formatting multiple ranges at once.
	// @since 3.18.0
	// @since 3.18.0
	RangesSupport Optional[bool] `json:"rangesSupport,omitzero"`
}

// The parameters of a DocumentRangeFormattingRequest.
type DocumentRangeFormattingParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The range to format
	Range Range `json:"range"`
	// The format options
	Options FormattingOptions `json:"options"`
}

// Registration options for a DocumentRangeFormattingRequest.
type DocumentRangeFormattingRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// Whether the server supports formatting multiple ranges at once.
	// @since 3.18.0
	// @since 3.18.0
	RangesSupport Optional[bool] `json:"rangesSupport,omitzero"`
}

// The parameters of a DocumentRangesFormattingRequest.
// @since 3.18.0
// @since 3.18.0
type DocumentRangesFormattingParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The ranges to format
	Ranges []Range `json:"ranges"`
	// The format options
	Options FormattingOptions `json:"options"`
}

// Represents programming constructs like variables, classes, interfaces etc.
// that appear in a document. Document symbols can be hierarchical and they
// have two ranges: one that encloses its definition and one that points to
// its most interesting range, e.g. the range of an identifier.
type DocumentSymbol struct {
	// The name of this symbol. Will be displayed in the user interface and therefore must not be
	// an empty string or a string only consisting of white spaces.
	Name string `json:"name"`
	// More detail for this symbol, e.g the signature of a function.
	Detail Optional[string] `json:"detail,omitzero"`
	// The kind of this symbol.
	Kind SymbolKind `json:"kind"`
	// Tags for this document symbol.
	// @since 3.16.0
	// @since 3.16.0
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// Indicates if this symbol is deprecated.
	// @deprecated Use tags instead
	// Deprecated: Use tags instead
	Deprecated Optional[bool] `json:"deprecated,omitzero"`
	// The range enclosing this symbol not including leading/trailing whitespace but everything else
	// like comments. This information is typically used to determine if the clients cursor is
	// inside the symbol to reveal in the symbol in the UI.
	Range Range `json:"range"`
	// The range that should be selected and revealed when this symbol is being picked, e.g the name of a function.
	// Must be contained by the `range`.
	SelectionRange Range `json:"selectionRange"`
	// Children of this symbol, e.g. properties of a class.
	Children Optional[[]DocumentSymbol] `json:"children,omitzero"`
}

// Client Capabilities for a DocumentSymbolRequest.
type DocumentSymbolClientCapabilities struct {
	// Whether document symbol supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Specific capabilities for the `SymbolKind` in the
	// `textDocument/documentSymbol` request.
	SymbolKind Optional[ClientSymbolKindOptions] `json:"symbolKind,omitzero"`
	// The client supports hierarchical document symbols.
	HierarchicalDocumentSymbolSupport Optional[bool] `json:"hierarchicalDocumentSymbolSupport,omitzero"`
	// The client supports tags on `SymbolInformation`. Tags are supported on
	// `DocumentSymbol` if `hierarchicalDocumentSymbolSupport` is set to true.
	// Clients supporting tags have to handle unknown tags gracefully.
	// @since 3.16.0
	// @since 3.16.0
	TagSupport Optional[ClientSymbolTagOptions] `json:"tagSupport,omitzero"`
	// The client supports an additional label presented in the UI when
	// registering a document symbol provider.
	// @since 3.16.0
	// @since 3.16.0
	LabelSupport Optional[bool] `json:"labelSupport,omitzero"`
}

// Provider options for a DocumentSymbolRequest.
type DocumentSymbolOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// A human-readable string that is shown when multiple outlines trees
	// are shown for the same document.
	// @since 3.16.0
	// @since 3.16.0
	Label Optional[string] `json:"label,omitzero"`
}

// Parameters for a DocumentSymbolRequest.
type DocumentSymbolParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// Registration options for a DocumentSymbolRequest.
type DocumentSymbolRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// A human-readable string that is shown when multiple outlines trees
	// are shown for the same document.
	// @since 3.16.0
	// @since 3.16.0
	Label Optional[string] `json:"label,omitzero"`
}

// Edit range variant that includes ranges for insert and replace operations.
// @since 3.18.0
// @since 3.18.0
type EditRangeWithInsertReplace struct {
	Insert  Range `json:"insert"`
	Replace Range `json:"replace"`
}

// The client capabilities of a ExecuteCommandRequest.
type ExecuteCommandClientCapabilities struct {
	// Execute command supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// The server capabilities of a ExecuteCommandRequest.
type ExecuteCommandOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The commands to be executed on the server
	Commands []string `json:"commands"`
}

// The parameters of a ExecuteCommandRequest.
type ExecuteCommandParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The identifier of the actual command handler.
	Command string `json:"command"`
	// Arguments that the command should be invoked with.
	Arguments Optional[[]LSPAny] `json:"arguments,omitzero"`
}

// Registration options for a ExecuteCommandRequest.
type ExecuteCommandRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The commands to be executed on the server
	Commands []string `json:"commands"`
}

type ExecutionSummary struct {
	// A strict monotonically increasing value
	// indicating the execution order of a cell
	// inside a notebook.
	ExecutionOrder uint32 `json:"executionOrder"`
	// Whether the execution was successful or
	// not if known by the client.
	Success Optional[bool] `json:"success,omitzero"`
}

// Represents information on a file/folder create.
// @since 3.16.0
// @since 3.16.0
type FileCreate struct {
	// A URI for the location of the file/folder being created.
	URI DocumentURI `json:"uri"`
}

// Represents information on a file/folder delete.
// @since 3.16.0
// @since 3.16.0
type FileDelete struct {
	// A URI for the location of the file/folder being deleted.
	URI DocumentURI `json:"uri"`
}

// An event describing a file change.
type FileEvent struct {
	// The file's uri.
	URI DocumentURI `json:"uri"`
	// The change type.
	Type FileChangeType `json:"type"`
}

// Capabilities relating to events from file operations by the user in the client.
// These events do not come from the file system, they come from user operations
// like renaming a file in the UI.
// @since 3.16.0
// @since 3.16.0
type FileOperationClientCapabilities struct {
	// Whether the client supports dynamic registration for file requests/notifications.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client has support for sending didCreateFiles notifications.
	DidCreate Optional[bool] `json:"didCreate,omitzero"`
	// The client has support for sending willCreateFiles requests.
	WillCreate Optional[bool] `json:"willCreate,omitzero"`
	// The client has support for sending didRenameFiles notifications.
	DidRename Optional[bool] `json:"didRename,omitzero"`
	// The client has support for sending willRenameFiles requests.
	WillRename Optional[bool] `json:"willRename,omitzero"`
	// The client has support for sending didDeleteFiles notifications.
	DidDelete Optional[bool] `json:"didDelete,omitzero"`
	// The client has support for sending willDeleteFiles requests.
	WillDelete Optional[bool] `json:"willDelete,omitzero"`
}

// A filter to describe in which file operation requests or notifications
// the server is interested in receiving.
// @since 3.16.0
// @since 3.16.0
type FileOperationFilter struct {
	// A Uri scheme like `file` or `untitled`.
	Scheme Optional[string] `json:"scheme,omitzero"`
	// The actual file operation pattern.
	Pattern FileOperationPattern `json:"pattern"`
}

// Options for notifications/requests for user operations on files.
// @since 3.16.0
// @since 3.16.0
type FileOperationOptions struct {
	// The server is interested in receiving didCreateFiles notifications.
	DidCreate Optional[FileOperationRegistrationOptions] `json:"didCreate,omitzero"`
	// The server is interested in receiving willCreateFiles requests.
	WillCreate Optional[FileOperationRegistrationOptions] `json:"willCreate,omitzero"`
	// The server is interested in receiving didRenameFiles notifications.
	DidRename Optional[FileOperationRegistrationOptions] `json:"didRename,omitzero"`
	// The server is interested in receiving willRenameFiles requests.
	WillRename Optional[FileOperationRegistrationOptions] `json:"willRename,omitzero"`
	// The server is interested in receiving didDeleteFiles file notifications.
	DidDelete Optional[FileOperationRegistrationOptions] `json:"didDelete,omitzero"`
	// The server is interested in receiving willDeleteFiles file requests.
	WillDelete Optional[FileOperationRegistrationOptions] `json:"willDelete,omitzero"`
}

// A pattern to describe in which file operation requests or notifications
// the server is interested in receiving.
// @since 3.16.0
// @since 3.16.0
type FileOperationPattern struct {
	// The glob pattern to match. Glob patterns can have the following syntax:
	// - `*` to match zero or more characters in a path segment
	// - `?` to match on one character in a path segment
	// - `**` to match any number of path segments, including none
	// - `{` to group sub patterns into an OR expression. (e.g. `**​/*.{ts,js` matches all TypeScript and JavaScript files)
	// - `[]` to declare a range of characters to match in a path segment (e.g., `example.[0-9]` to match on `example.0`, `example.1`, …)
	// - `[!...]` to negate a range of characters to match in a path segment (e.g., `example.[!0-9]` to match on `example.a`, `example.b`, but not `example.0`)
	Glob string `json:"glob"`
	// Whether to match files or folders with this pattern.
	// Matches both if undefined.
	Matches Optional[FileOperationPatternKind] `json:"matches,omitzero"`
	// Additional options used during matching.
	Options Optional[FileOperationPatternOptions] `json:"options,omitzero"`
}

// Matching options for the file operation pattern.
// @since 3.16.0
// @since 3.16.0
type FileOperationPatternOptions struct {
	// The pattern should be matched ignoring casing.
	IgnoreCase Optional[bool] `json:"ignoreCase,omitzero"`
}

// The options to register for file operations.
// @since 3.16.0
// @since 3.16.0
type FileOperationRegistrationOptions struct {
	// The actual filters.
	Filters []FileOperationFilter `json:"filters"`
}

// Represents information on a file/folder rename.
// @since 3.16.0
// @since 3.16.0
type FileRename struct {
	// A URI for the original location of the file/folder being renamed.
	OldURI DocumentURI `json:"oldUri"`
	// A URI for the new location of the file/folder being renamed.
	NewURI DocumentURI `json:"newUri"`
}

type FileSystemWatcher struct {
	// The glob pattern to watch. See GlobPattern glob pattern for more detail.
	// @since 3.17.0 support for relative patterns.
	// @since 3.17.0 support for relative patterns.
	GlobPattern GlobPattern `json:"globPattern"`
	// The kind of events of interest. If omitted it defaults
	// to WatchKind.Create | WatchKind.Change | WatchKind.Delete
	// which is 7.
	Kind Optional[WatchKind] `json:"kind,omitzero"`
}

// Represents a folding range. To be valid, start and end line must be bigger than zero and smaller
// than the number of lines in the document. Clients are free to ignore invalid ranges.
type FoldingRange struct {
	// The zero-based start line of the range to fold. The folded area starts after the line's last character.
	// To be valid, the end must be zero or larger and smaller than the number of lines in the document.
	StartLine uint32 `json:"startLine"`
	// The zero-based character offset from where the folded range starts. If not defined, defaults to the length of the start line.
	StartCharacter Optional[uint32] `json:"startCharacter,omitzero"`
	// The zero-based end line of the range to fold. The folded area ends with the line's last character.
	// To be valid, the end must be zero or larger and smaller than the number of lines in the document.
	EndLine uint32 `json:"endLine"`
	// The zero-based character offset before the folded range ends. If not defined, defaults to the length of the end line.
	EndCharacter Optional[uint32] `json:"endCharacter,omitzero"`
	// Describes the kind of the folding range such as 'comment' or 'region'. The kind
	// is used to categorize folding ranges and used by commands like 'Fold all comments'.
	// See FoldingRangeKind for an enumeration of standardized kinds.
	Kind Optional[FoldingRangeKind] `json:"kind,omitzero"`
	// The text that the client should show when the specified range is
	// collapsed. If not defined or not supported by the client, a default
	// will be chosen by the client.
	// @since 3.17.0
	// @since 3.17.0
	CollapsedText Optional[string] `json:"collapsedText,omitzero"`
}

type FoldingRangeClientCapabilities struct {
	// Whether implementation supports dynamic registration for folding range
	// providers. If this is set to `true` the client supports the new
	// `FoldingRangeRegistrationOptions` return value for the corresponding
	// server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The maximum number of folding ranges that the client prefers to receive
	// per document. The value serves as a hint, servers are free to follow the
	// limit.
	RangeLimit Optional[uint32] `json:"rangeLimit,omitzero"`
	// If set, the client signals that it only supports folding complete lines.
	// If set, client will ignore specified `startCharacter` and `endCharacter`
	// properties in a FoldingRange.
	LineFoldingOnly Optional[bool] `json:"lineFoldingOnly,omitzero"`
	// Specific options for the folding range kind.
	// @since 3.17.0
	// @since 3.17.0
	FoldingRangeKind Optional[ClientFoldingRangeKindOptions] `json:"foldingRangeKind,omitzero"`
	// Specific options for the folding range.
	// @since 3.17.0
	// @since 3.17.0
	FoldingRange Optional[ClientFoldingRangeOptions] `json:"foldingRange,omitzero"`
}

type FoldingRangeOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a FoldingRangeRequest.
type FoldingRangeParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type FoldingRangeRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Client workspace capabilities specific to folding ranges
// @since 3.18.0
// @since 3.18.0
type FoldingRangeWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from the
	// server to the client.
	// Note that this event is global and will force the client to refresh all
	// folding ranges currently shown. It should be used with absolute care and is
	// useful for situation where a server for example detects a project wide
	// change that requires such a calculation.
	// @since 3.18.0
	// @since 3.18.0
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// Value-object describing what options formatting should use.
type FormattingOptions struct {
	// Size of a tab in spaces.
	TabSize uint32 `json:"tabSize"`
	// Prefer spaces over tabs.
	InsertSpaces bool `json:"insertSpaces"`
	// Trim trailing whitespace on a line.
	// @since 3.15.0
	// @since 3.15.0
	TrimTrailingWhitespace Optional[bool] `json:"trimTrailingWhitespace,omitzero"`
	// Insert a newline character at the end of the file if one does not exist.
	// @since 3.15.0
	// @since 3.15.0
	InsertFinalNewline Optional[bool] `json:"insertFinalNewline,omitzero"`
	// Trim all newlines after the final newline at the end of the file.
	// @since 3.15.0
	// @since 3.15.0
	TrimFinalNewlines Optional[bool] `json:"trimFinalNewlines,omitzero"`
}

// A diagnostic report with a full set of problems.
// @since 3.17.0
// @since 3.17.0
type FullDocumentDiagnosticReport struct {
	// A full document diagnostic report.
	Kind string `json:"kind"`
	// An optional result id. If provided it will
	// be sent on the next diagnostic request for the
	// same document.
	ResultID Optional[string] `json:"resultId,omitzero"`
	// The actual items.
	Items []Diagnostic `json:"items"`
}

// General client capabilities.
// @since 3.16.0
// @since 3.16.0
type GeneralClientCapabilities struct {
	// Client capability that signals how the client
	// handles stale requests (e.g. a request
	// for which the client will not process the response
	// anymore since the information is outdated).
	// @since 3.17.0
	// @since 3.17.0
	StaleRequestSupport Optional[StaleRequestSupportOptions] `json:"staleRequestSupport,omitzero"`
	// Client capabilities specific to regular expressions.
	// @since 3.16.0
	// @since 3.16.0
	RegularExpressions Optional[RegularExpressionsClientCapabilities] `json:"regularExpressions,omitzero"`
	// Client capabilities specific to the client's markdown parser.
	// @since 3.16.0
	// @since 3.16.0
	Markdown Optional[MarkdownClientCapabilities] `json:"markdown,omitzero"`
	// The position encodings supported by the client. Client and server
	// have to agree on the same position encoding to ensure that offsets
	// (e.g. character position in a line) are interpreted the same on both
	// sides.
	// To keep the protocol backwards compatible the following applies: if
	// the value 'utf-16' is missing from the array of position encodings
	// servers can assume that the client supports UTF-16. UTF-16 is
	// therefore a mandatory encoding.
	// If omitted it defaults to ['utf-16'].
	// Implementation considerations: since the conversion from one encoding
	// into another requires the content of the file / line the conversion
	// is best done where the file is read which is usually on the server
	// side.
	// @since 3.17.0
	// @since 3.17.0
	PositionEncodings Optional[[]PositionEncodingKind] `json:"positionEncodings,omitzero"`
}

// The result of a hover request.
type Hover struct {
	// The hover's content
	Contents OrHoverContents `json:"contents"`
	// An optional range inside the text document that is used to
	// visualize the hover, e.g. by changing the background color.
	Range Optional[Range] `json:"range,omitzero"`
}

type HoverClientCapabilities struct {
	// Whether hover supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Client supports the following content formats for the content
	// property. The order describes the preferred format of the client.
	ContentFormat Optional[[]MarkupKind] `json:"contentFormat,omitzero"`
}

// Hover options.
type HoverOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a HoverRequest.
type HoverParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

// Registration options for a HoverRequest.
type HoverRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// @since 3.6.0
// @since 3.6.0
type ImplementationClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `ImplementationRegistrationOptions` return value
	// for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports additional metadata in the form of definition links.
	// @since 3.14.0
	// @since 3.14.0
	LinkSupport Optional[bool] `json:"linkSupport,omitzero"`
}

type ImplementationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type ImplementationParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

type ImplementationRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// The data type of the ResponseError if the
// initialize request fails.
type InitializeError struct {
	// Indicates whether the client execute the following retry logic:
	// (1) show the message provided by the ResponseError to the user
	// (2) user selects retry or cancel
	// (3) if user selected retry the initialize method is sent again.
	Retry bool `json:"retry"`
}

type InitializeParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The process Id of the parent process that started
	// the server.
	// Is `null` if the process has not been started by another process.
	// If the parent process is not alive then the server should exit.
	ProcessID Nullable[OrInitializeParamsProcessId] `json:"processId"`
	// Information about the client
	// @since 3.15.0
	// @since 3.15.0
	ClientInfo Optional[ClientInfo] `json:"clientInfo,omitzero"`
	// The locale the client is currently showing the user interface
	// in. This must not necessarily be the locale of the operating
	// system.
	// Uses IETF language tags as the value's syntax
	// (See https://en.wikipedia.org/wiki/IETF_language_tag)
	// @since 3.16.0
	// @since 3.16.0
	Locale Optional[string] `json:"locale,omitzero"`
	// The rootPath of the workspace. Is null
	// if no folder is open.
	// @deprecated in favour of rootUri.
	// Deprecated: in favour of rootUri.
	RootPath OptionalNullable[OrInitializeParamsRootPath] `json:"rootPath,omitzero"`
	// The rootUri of the workspace. Is null if no
	// folder is open. If both `rootPath` and `rootUri` are set
	// `rootUri` wins.
	// @deprecated in favour of workspaceFolders.
	// Deprecated: in favour of workspaceFolders.
	RootURI Nullable[OrInitializeParamsRootUri] `json:"rootUri"`
	// The capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities"`
	// User provided initialization options.
	InitializationOptions Optional[LSPAny] `json:"initializationOptions,omitzero"`
	// The initial trace setting. If omitted trace is disabled ('off').
	Trace Optional[TraceValue] `json:"trace,omitzero"`
	// The workspace folders configured in the client when the server starts.
	// This property is only available if the client supports workspace folders.
	// It can be `null` if the client supports workspace folders but none are
	// configured.
	// @since 3.6.0
	// @since 3.6.0
	WorkspaceFolders OptionalNullable[OrWorkspaceFoldersInitializeParamsWorkspaceFolders] `json:"workspaceFolders,omitzero"`
}

// The result returned from an initialize request.
type InitializeResult struct {
	// The capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities"`
	// Information about the server.
	// @since 3.15.0
	// @since 3.15.0
	ServerInfo Optional[ServerInfo] `json:"serverInfo,omitzero"`
}

type InitializedParams struct {
}

// Inlay hint information.
// @since 3.17.0
// @since 3.17.0
type InlayHint struct {
	// The position of this hint.
	// If multiple hints have the same position, they will be shown in the order
	// they appear in the response.
	Position Position `json:"position"`
	// The label of this hint. A human readable string or an array of
	// InlayHintLabelPart label parts.
	// *Note* that neither the string nor the label part can be empty.
	Label OrInlayHintLabel `json:"label"`
	// The kind of this hint. Can be omitted in which case the client
	// should fall back to a reasonable default.
	Kind Optional[InlayHintKind] `json:"kind,omitzero"`
	// Optional text edits that are performed when accepting this inlay hint.
	// *Note* that edits are expected to change the document so that the inlay
	// hint (or its nearest variant) is now part of the document and the inlay
	// hint itself is now obsolete.
	TextEdits Optional[[]TextEdit] `json:"textEdits,omitzero"`
	// The tooltip text when you hover over this item.
	Tooltip Optional[OrInlayHintTooltip] `json:"tooltip,omitzero"`
	// Render padding before the hint.
	// Note: Padding should use the editor's background color, not the
	// background color of the hint itself. That means padding can be used
	// to visually align/separate an inlay hint.
	PaddingLeft Optional[bool] `json:"paddingLeft,omitzero"`
	// Render padding after the hint.
	// Note: Padding should use the editor's background color, not the
	// background color of the hint itself. That means padding can be used
	// to visually align/separate an inlay hint.
	PaddingRight Optional[bool] `json:"paddingRight,omitzero"`
	// A data entry field that is preserved on an inlay hint between
	// a `textDocument/inlayHint` and a `inlayHint/resolve` request.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Inlay hint client capabilities.
// @since 3.17.0
// @since 3.17.0
type InlayHintClientCapabilities struct {
	// Whether inlay hints support dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Indicates which properties a client can resolve lazily on an inlay
	// hint.
	ResolveSupport Optional[ClientInlayHintResolveOptions] `json:"resolveSupport,omitzero"`
}

// An inlay hint label part allows for interactive and composite labels
// of inlay hints.
// @since 3.17.0
// @since 3.17.0
type InlayHintLabelPart struct {
	// The value of this label part.
	Value string `json:"value"`
	// The tooltip text when you hover over this label part. Depending on
	// the client capability `inlayHint.resolveSupport` clients might resolve
	// this property late using the resolve request.
	Tooltip Optional[OrInlayHintLabelPartTooltip] `json:"tooltip,omitzero"`
	// An optional source code location that represents this
	// label part.
	// The editor will use this location for the hover and for code navigation
	// features: This part will become a clickable link that resolves to the
	// definition of the symbol at the given location (not necessarily the
	// location itself), it shows the hover that shows at the given location,
	// and it shows a context menu with further code navigation commands.
	// Depending on the client capability `inlayHint.resolveSupport` clients
	// might resolve this property late using the resolve request.
	Location Optional[Location] `json:"location,omitzero"`
	// An optional command for this label part.
	// Depending on the client capability `inlayHint.resolveSupport` clients
	// might resolve this property late using the resolve request.
	Command Optional[Command] `json:"command,omitzero"`
}

// Inlay hint options used during static registration.
// @since 3.17.0
// @since 3.17.0
type InlayHintOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The server provides support to resolve additional
	// information for an inlay hint item.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// A parameter literal used in inlay hint requests.
// @since 3.17.0
// @since 3.17.0
type InlayHintParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The document range for which inlay hints should be computed.
	Range Range `json:"range"`
}

// Inlay hint options used during static or dynamic registration.
// @since 3.17.0
// @since 3.17.0
type InlayHintRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The server provides support to resolve additional
	// information for an inlay hint item.
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Client workspace capabilities specific to inlay hints.
// @since 3.17.0
// @since 3.17.0
type InlayHintWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from
	// the server to the client.
	// Note that this event is global and will force the client to refresh all
	// inlay hints currently shown. It should be used with absolute care and
	// is useful for situation where a server for example detects a project wide
	// change that requires such a calculation.
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// Client capabilities specific to inline completions.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionClientCapabilities struct {
	// Whether implementation supports dynamic registration for inline completion providers.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Provides information about the context in which an inline completion was requested.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionContext struct {
	// Describes how the inline completion was triggered.
	TriggerKind InlineCompletionTriggerKind `json:"triggerKind"`
	// Provides information about the currently selected item in the autocomplete widget if it is visible.
	SelectedCompletionInfo Optional[SelectedCompletionInfo] `json:"selectedCompletionInfo,omitzero"`
}

// An inline completion item represents a text snippet that is proposed inline to complete text that is being typed.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionItem struct {
	// The text to replace the range with. Must be set.
	InsertText OrInlineCompletionItemInsertText `json:"insertText"`
	// A text that is used to decide if this inline completion should be shown. When `falsy` the InlineCompletionItem.insertText is used.
	FilterText Optional[string] `json:"filterText,omitzero"`
	// The range to replace. Must begin and end on the same line.
	Range Optional[Range] `json:"range,omitzero"`
	// An optional Command that is executed *after* inserting this completion.
	Command Optional[Command] `json:"command,omitzero"`
}

// Represents a collection of InlineCompletionItem inline completion items to be presented in the editor.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionList struct {
	// The inline completion items
	Items []InlineCompletionItem `json:"items"`
}

// Inline completion options used during static registration.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// A parameter literal used in inline completion requests.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// Additional information about the context in which inline completions were
	// requested.
	Context InlineCompletionContext `json:"context"`
}

// Inline completion options used during static or dynamic registration.
// @since 3.18.0
// @since 3.18.0
type InlineCompletionRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Client capabilities specific to inline values.
// @since 3.17.0
// @since 3.17.0
type InlineValueClientCapabilities struct {
	// Whether implementation supports dynamic registration for inline value providers.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// @since 3.17.0
// @since 3.17.0
type InlineValueContext struct {
	// The stack frame (as a DAP Id) where the execution has stopped.
	FrameID int32 `json:"frameId"`
	// The document range where execution has stopped.
	// Typically the end position of the range denotes the line where the inline values are shown.
	StoppedLocation Range `json:"stoppedLocation"`
}

// To compute an inline value through an expression evaluation.
// If only a range is specified, the expression should be
// extracted from the underlying document.
// An optional expression could be evaluated instead of
// the extracted expression.
// @since 3.17.0
// @since 3.17.0
type InlineValueEvaluatableExpression struct {
	// The document range for which the inline value applies.
	// The range could be used to extract the evaluatable expression
	// from the underlying document.
	Range Range `json:"range"`
	// If specified the expression could be evaluated instead.
	Expression Optional[string] `json:"expression,omitzero"`
}

// Inline value options used during static registration.
// @since 3.17.0
// @since 3.17.0
type InlineValueOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// A parameter literal used in inline value requests.
// @since 3.17.0
// @since 3.17.0
type InlineValueParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The document range for which inline values information will be returned.
	Range Range `json:"range"`
	// Additional information about the context in which inline values information was
	// requested.
	Context InlineValueContext `json:"context"`
}

// Inline value options used during static or dynamic registration.
// @since 3.17.0
// @since 3.17.0
type InlineValueRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Returns inline value information as the complete text to be shown.
// @since 3.17.0
// @since 3.17.0
type InlineValueText struct {
	// The document range for which the inline value applies.
	Range Range `json:"range"`
	// The text of the inline value.
	Text string `json:"text"`
}

// To compute inline value through a variable lookup.
// If only a range is specified, the variable name should
// be extracted from the underlying document.
// An optional variable name could be used to lookup instead
// of the extracted name.
// @since 3.17.0
// @since 3.17.0
type InlineValueVariableLookup struct {
	// The document range for which the inline value applies.
	// The range could be used to extract the variable name
	// from the underlying document.
	Range Range `json:"range"`
	// If specified the name of the variable to look up.
	VariableName Optional[string] `json:"variableName,omitzero"`
	// How to perform the lookup.
	CaseSensitiveLookup bool `json:"caseSensitiveLookup"`
}

// Client workspace capabilities specific to inline values.
// @since 3.17.0
// @since 3.17.0
type InlineValueWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from the
	// server to the client.
	// Note that this event is global and will force the client to refresh all
	// inline values currently shown. It should be used with absolute care and is
	// useful for situation where a server for example detects a project wide
	// change that requires such a calculation.
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// A special text edit to provide an insert and a replace operation.
// @since 3.16.0
// @since 3.16.0
type InsertReplaceEdit struct {
	// The string to be inserted.
	NewText string `json:"newText"`
	// The range if the insert is requested
	Insert Range `json:"insert"`
	// The range if the replace is requested.
	Replace Range `json:"replace"`
}

// Client capabilities for the linked editing range request.
// @since 3.16.0
// @since 3.16.0
type LinkedEditingRangeClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

type LinkedEditingRangeOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type LinkedEditingRangeParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

type LinkedEditingRangeRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// The result of a linked editing range request.
// @since 3.16.0
// @since 3.16.0
type LinkedEditingRanges struct {
	// A list of ranges that can be edited together. The ranges must have
	// identical length and contain identical text content. The ranges cannot overlap.
	Ranges []Range `json:"ranges"`
	// An optional word pattern (regular expression) that describes valid contents for
	// the given ranges. If no pattern is provided, the client configuration's word
	// pattern will be used.
	WordPattern Optional[string] `json:"wordPattern,omitzero"`
}

// Represents a location inside a resource, such as a line
// inside a text file.
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// Represents the connection of two locations. Provides additional metadata over normal Location locations,
// including an origin range.
type LocationLink struct {
	// Span of the origin of this link.
	// Used as the underlined span for mouse interaction. Defaults to the word range at
	// the definition position.
	OriginSelectionRange Optional[Range] `json:"originSelectionRange,omitzero"`
	// The target resource identifier of this link.
	TargetURI DocumentURI `json:"targetUri"`
	// The full target range of this link. If the target for example is a symbol then target range is the
	// range enclosing this symbol not including leading/trailing whitespace but everything else
	// like comments. This information is typically used to highlight the range in the editor.
	TargetRange Range `json:"targetRange"`
	// The range that should be selected and revealed when this link is being followed, e.g the name of a function.
	// Must be contained by the `targetRange`. See also `DocumentSymbol#range`
	TargetSelectionRange Range `json:"targetSelectionRange"`
}

// Location with only uri and does not include range.
// @since 3.18.0
// @since 3.18.0
type LocationUriOnly struct {
	URI DocumentURI `json:"uri"`
}

// The log message parameters.
type LogMessageParams struct {
	// The message type. See MessageType
	Type MessageType `json:"type"`
	// The actual message.
	Message string `json:"message"`
}

type LogTraceParams struct {
	Message string           `json:"message"`
	Verbose Optional[string] `json:"verbose,omitzero"`
}

// Client capabilities specific to the used markdown parser.
// @since 3.16.0
// @since 3.16.0
type MarkdownClientCapabilities struct {
	// The name of the parser.
	Parser string `json:"parser"`
	// The version of the parser.
	Version Optional[string] `json:"version,omitzero"`
	// A list of HTML tags that the client allows / supports in
	// Markdown.
	// @since 3.17.0
	// @since 3.17.0
	AllowedTags Optional[[]string] `json:"allowedTags,omitzero"`
}

// @since 3.18.0
// @deprecated use MarkupContent instead.
// @since 3.18.0
// Deprecated: use MarkupContent instead.
type MarkedStringWithLanguage struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// A `MarkupContent` literal represents a string value which content is interpreted base on its
// kind flag. Currently the protocol supports `plaintext` and `markdown` as markup kinds.
// If the kind is `markdown` then the value can contain fenced code blocks like in GitHub issues.
// See https://help.github.com/articles/creating-and-highlighting-code-blocks/#syntax-highlighting
// Here is an example how such a string can be constructed using JavaScript / TypeScript:
// ```ts
// let markdown: MarkdownContent = {
// kind: MarkupKind.Markdown,
// value: [
// '# Header',
// 'Some text',
// '```typescript',
// 'someCode();',
// '```'
// ].join('\n')
// ;
// ```
// *Please Note* that clients might sanitize the return markdown. A client could decide to
// remove HTML from the markdown to avoid script execution.
type MarkupContent struct {
	// The type of the Markup
	Kind MarkupKind `json:"kind"`
	// The content itself
	Value string `json:"value"`
}

type MessageActionItem struct {
	// A short title like 'Retry', 'Open Log' etc.
	Title string `json:"title"`
}

// Moniker definition to match LSIF 0.5 moniker definition.
// @since 3.16.0
// @since 3.16.0
type Moniker struct {
	// The scheme of the moniker. For example tsc or .Net
	Scheme string `json:"scheme"`
	// The identifier of the moniker. The value is opaque in LSIF however
	// schema owners are allowed to define the structure if they want.
	Identifier string `json:"identifier"`
	// The scope in which the moniker is unique
	Unique UniquenessLevel `json:"unique"`
	// The moniker kind if known.
	Kind Optional[MonikerKind] `json:"kind,omitzero"`
}

// Client capabilities specific to the moniker request.
// @since 3.16.0
// @since 3.16.0
type MonikerClientCapabilities struct {
	// Whether moniker supports dynamic registration. If this is set to `true`
	// the client supports the new `MonikerRegistrationOptions` return value
	// for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

type MonikerOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type MonikerParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

type MonikerRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// A notebook cell.
// A cell's document URI must be unique across ALL notebook
// cells and can therefore be used to uniquely identify a
// notebook cell or the cell's text document.
// @since 3.17.0
// @since 3.17.0
type NotebookCell struct {
	// The cell's kind
	Kind NotebookCellKind `json:"kind"`
	// The URI of the cell's text document
	// content.
	Document DocumentURI `json:"document"`
	// Additional metadata stored with the cell.
	// Note: should always be an object literal (e.g. LSPObject)
	Metadata Optional[LSPObject] `json:"metadata,omitzero"`
	// Additional execution summary information
	// if supported by the client.
	ExecutionSummary Optional[ExecutionSummary] `json:"executionSummary,omitzero"`
}

// A change describing how to move a `NotebookCell`
// array from state S to S'.
// @since 3.17.0
// @since 3.17.0
type NotebookCellArrayChange struct {
	// The start oftest of the cell that changed.
	Start uint32 `json:"start"`
	// The deleted cells
	DeleteCount uint32 `json:"deleteCount"`
	// The new cells, if any
	Cells Optional[[]NotebookCell] `json:"cells,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type NotebookCellLanguage struct {
	Language string `json:"language"`
}

// A notebook cell text document filter denotes a cell text
// document by different properties.
// @since 3.17.0
// @since 3.17.0
type NotebookCellTextDocumentFilter struct {
	// A filter that matches against the notebook
	// containing the notebook cell. If a string
	// value is provided it matches against the
	// notebook type. '*' matches every notebook.
	Notebook OrNotebookCellTextDocumentFilterNotebook `json:"notebook"`
	// A language id like `python`.
	// Will be matched against the language id of the
	// notebook cell document. '*' matches every language.
	Language Optional[string] `json:"language,omitzero"`
}

// A notebook document.
// @since 3.17.0
// @since 3.17.0
type NotebookDocument struct {
	// The notebook document's uri.
	URI string `json:"uri"`
	// The type of the notebook.
	NotebookType string `json:"notebookType"`
	// The version number of this document (it will increase after each
	// change, including undo/redo).
	Version int32 `json:"version"`
	// Additional metadata stored with the notebook
	// document.
	// Note: should always be an object literal (e.g. LSPObject)
	Metadata Optional[LSPObject] `json:"metadata,omitzero"`
	// The cells of a notebook.
	Cells []NotebookCell `json:"cells"`
}

// Structural changes to cells in a notebook document.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentCellChangeStructure struct {
	// The change to the cell array.
	Array NotebookCellArrayChange `json:"array"`
	// Additional opened cell text documents.
	DidOpen Optional[[]TextDocumentItem] `json:"didOpen,omitzero"`
	// Additional closed cell text documents.
	DidClose Optional[[]TextDocumentIdentifier] `json:"didClose,omitzero"`
}

// Cell changes to a notebook document.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentCellChanges struct {
	// Changes to the cell structure to add or
	// remove cells.
	Structure Optional[NotebookDocumentCellChangeStructure] `json:"structure,omitzero"`
	// Changes to notebook cells properties like its
	// kind, execution summary or metadata.
	Data Optional[[]NotebookCell] `json:"data,omitzero"`
	// Changes to the text content of notebook cells.
	TextContent Optional[[]NotebookDocumentCellContentChanges] `json:"textContent,omitzero"`
}

// Content changes to a cell in a notebook document.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentCellContentChanges struct {
	Document VersionedTextDocumentIdentifier  `json:"document"`
	Changes  []TextDocumentContentChangeEvent `json:"changes"`
}

// A change event for a notebook document.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentChangeEvent struct {
	// The changed meta data if any.
	// Note: should always be an object literal (e.g. LSPObject)
	Metadata Optional[LSPObject] `json:"metadata,omitzero"`
	// Changes to cells
	Cells Optional[NotebookDocumentCellChanges] `json:"cells,omitzero"`
}

// Capabilities specific to the notebook document support.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentClientCapabilities struct {
	// Capabilities specific to notebook document synchronization
	// @since 3.17.0
	// @since 3.17.0
	Synchronization NotebookDocumentSyncClientCapabilities `json:"synchronization"`
}

// A notebook document filter where `notebookType` is required field.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentFilterNotebookType struct {
	// The type of the enclosing notebook.
	NotebookType string `json:"notebookType"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme Optional[string] `json:"scheme,omitzero"`
	// A glob pattern.
	Pattern Optional[GlobPattern] `json:"pattern,omitzero"`
}

// A notebook document filter where `pattern` is required field.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentFilterPattern struct {
	// The type of the enclosing notebook.
	NotebookType Optional[string] `json:"notebookType,omitzero"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme Optional[string] `json:"scheme,omitzero"`
	// A glob pattern.
	Pattern GlobPattern `json:"pattern"`
}

// A notebook document filter where `scheme` is required field.
// @since 3.18.0
// @since 3.18.0
type NotebookDocumentFilterScheme struct {
	// The type of the enclosing notebook.
	NotebookType Optional[string] `json:"notebookType,omitzero"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme string `json:"scheme"`
	// A glob pattern.
	Pattern Optional[GlobPattern] `json:"pattern,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type NotebookDocumentFilterWithCells struct {
	// The notebook to be synced If a string
	// value is provided it matches against the
	// notebook type. '*' matches every notebook.
	Notebook Optional[OrNotebookDocumentFilterWithCellsNotebook] `json:"notebook,omitzero"`
	// The cells of the matching notebook to be synced.
	Cells []NotebookCellLanguage `json:"cells"`
}

// @since 3.18.0
// @since 3.18.0
type NotebookDocumentFilterWithNotebook struct {
	// The notebook to be synced If a string
	// value is provided it matches against the
	// notebook type. '*' matches every notebook.
	Notebook OrNotebookDocumentFilterWithNotebookNotebook `json:"notebook"`
	// The cells of the matching notebook to be synced.
	Cells Optional[[]NotebookCellLanguage] `json:"cells,omitzero"`
}

// A literal to identify a notebook document in the client.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentIdentifier struct {
	// The notebook document's uri.
	URI string `json:"uri"`
}

// Notebook specific client capabilities.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentSyncClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is
	// set to `true` the client supports the new
	// `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports sending execution summary data per cell.
	ExecutionSummarySupport Optional[bool] `json:"executionSummarySupport,omitzero"`
}

// Options specific to a notebook plus its cells
// to be synced to the server.
// If a selector provides a notebook document
// filter but no cell selector all cells of a
// matching notebook document will be synced.
// If a selector provides no notebook document
// filter but only a cell selector all notebook
// document that contain at least one matching
// cell will be synced.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentSyncOptions struct {
	// The notebooks to be synced
	NotebookSelector []OrNotebookDocumentSyncOptionsNotebookSelectorElem `json:"notebookSelector"`
	// Whether save notification should be forwarded to
	// the server. Will only be honored if mode === `notebook`.
	Save Optional[bool] `json:"save,omitzero"`
}

// Registration options specific to a notebook.
// @since 3.17.0
// @since 3.17.0
type NotebookDocumentSyncRegistrationOptions struct {
	// The notebooks to be synced
	NotebookSelector []OrNotebookDocumentSyncOptionsNotebookSelectorElem `json:"notebookSelector"`
	// Whether save notification should be forwarded to
	// the server. Will only be honored if mode === `notebook`.
	Save Optional[bool] `json:"save,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// A text document identifier to optionally denote a specific version of a text document.
type OptionalVersionedTextDocumentIdentifier struct {
	// The text document's uri.
	URI DocumentURI `json:"uri"`
	// The version number of this document. If a versioned text document identifier
	// is sent from the server to the client and the file is not open in the editor
	// (the server has not received an open notification before) the server can send
	// `null` to indicate that the version is unknown and the content on disk is the
	// truth (as specified with document content ownership).
	Version Nullable[OrOptionalVersionedTextDocumentIdentifierVersion] `json:"version"`
}

// Represents a parameter of a callable-signature. A parameter can
// have a label and a doc-comment.
type ParameterInformation struct {
	// The label of this parameter information.
	// Either a string or an inclusive start and exclusive end offsets within its containing
	// signature label. (see SignatureInformation.label). The offsets are based on a UTF-16
	// string representation as `Position` and `Range` does.
	// To avoid ambiguities a server should use the [start, end] offset value instead of using
	// a substring. Whether a client support this is controlled via `labelOffsetSupport` client
	// capability.
	// *Note*: a label of type string should be a substring of its containing signature label.
	// Its intended use case is to highlight the parameter label part in the `SignatureInformation.label`.
	Label OrParameterInformationLabel `json:"label"`
	// The human-readable doc-comment of this parameter. Will be shown
	// in the UI but can be omitted.
	Documentation Optional[OrParameterInformationDocumentation] `json:"documentation,omitzero"`
}

type PartialResultParams struct {
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

// Position in a text document expressed as zero-based line and character
// offset. Prior to 3.17 the offsets were always based on a UTF-16 string
// representation. So a string of the form `a𐐀b` the character offset of the
// character `a` is 0, the character offset of `𐐀` is 1 and the character
// offset of b is 3 since `𐐀` is represented using two code units in UTF-16.
// Since 3.17 clients and servers can agree on a different string encoding
// representation (e.g. UTF-8). The client announces it's supported encoding
// via the client capability [`general.positionEncodings`](https://microsoft.github.io/language-server-protocol/specifications/specification-current/#clientCapabilities).
// The value is an array of position encodings the client supports, with
// decreasing preference (e.g. the encoding at index `0` is the most preferred
// one). To stay backwards compatible the only mandatory encoding is UTF-16
// represented via the string `utf-16`. The server can pick one of the
// encodings offered by the client and signals that encoding back to the
// client via the initialize result's property
// [`capabilities.positionEncoding`](https://microsoft.github.io/language-server-protocol/specifications/specification-current/#serverCapabilities). If the string value
// `utf-16` is missing from the client's capability `general.positionEncodings`
// servers can safely assume that the client supports UTF-16. If the server
// omits the position encoding in its initialize result the encoding defaults
// to the string value `utf-16`. Implementation considerations: since the
// conversion from one encoding into another requires the content of the
// file / line the conversion is best done where the file is read which is
// usually on the server side.
// Positions are line end character agnostic. So you can not specify a position
// that denotes `\r|\n` or `\n|` where `|` represents the character offset.
// @since 3.17.0 - support for negotiated position encoding.
// @since 3.17.0 - support for negotiated position encoding.
type Position struct {
	// Line position in a document (zero-based).
	Line uint32 `json:"line"`
	// Character offset on a line in a document (zero-based).
	// The meaning of this offset is determined by the negotiated
	// `PositionEncodingKind`.
	Character uint32 `json:"character"`
}

// @since 3.18.0
// @since 3.18.0
type PrepareRenameDefaultBehavior struct {
	DefaultBehavior bool `json:"defaultBehavior"`
}

type PrepareRenameParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type PrepareRenamePlaceholder struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder"`
}

// A previous result id in a workspace pull request.
// @since 3.17.0
// @since 3.17.0
type PreviousResultID struct {
	// The URI for which the client knowns a
	// result id.
	URI DocumentURI `json:"uri"`
	// The value of the previous result id.
	Value string `json:"value"`
}

type ProgressParams struct {
	// The progress token provided by the client or server.
	Token ProgressToken `json:"token"`
	// The progress data.
	Value LSPAny `json:"value"`
}

// The publish diagnostic client capabilities.
type PublishDiagnosticsClientCapabilities struct {
	// Whether the clients accepts diagnostics with related information.
	RelatedInformation Optional[bool] `json:"relatedInformation,omitzero"`
	// Client supports the tag property to provide meta data about a diagnostic.
	// Clients supporting tags have to handle unknown tags gracefully.
	// @since 3.15.0
	// @since 3.15.0
	TagSupport Optional[ClientDiagnosticsTagOptions] `json:"tagSupport,omitzero"`
	// Client supports a codeDescription property
	// @since 3.16.0
	// @since 3.16.0
	CodeDescriptionSupport Optional[bool] `json:"codeDescriptionSupport,omitzero"`
	// Whether code action supports the `data` property which is
	// preserved between a `textDocument/publishDiagnostics` and
	// `textDocument/codeAction` request.
	// @since 3.16.0
	// @since 3.16.0
	DataSupport Optional[bool] `json:"dataSupport,omitzero"`
	// Whether the client interprets the version property of the
	// `textDocument/publishDiagnostics` notification's parameter.
	// @since 3.15.0
	// @since 3.15.0
	VersionSupport Optional[bool] `json:"versionSupport,omitzero"`
}

// The publish diagnostic notification's parameters.
type PublishDiagnosticsParams struct {
	// The URI for which diagnostic information is reported.
	URI DocumentURI `json:"uri"`
	// Optional the version number of the document the diagnostics are published for.
	// @since 3.15.0
	// @since 3.15.0
	Version Optional[int32] `json:"version,omitzero"`
	// An array of diagnostic information items.
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// A range in a text document expressed as (zero-based) start and end positions.
// If you want to specify a range that contains a line including the line ending
// character(s) then use an end position denoting the start of the next line.
// For example:
// ```ts
// {
// start: { line: 5, character: 23
// end : { line 6, character : 0
//
// ```
type Range struct {
	// The range's start position.
	Start Position `json:"start"`
	// The range's end position.
	End Position `json:"end"`
}

// Client Capabilities for a ReferencesRequest.
type ReferenceClientCapabilities struct {
	// Whether references supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Value-object that contains additional information when
// requesting references.
type ReferenceContext struct {
	// Include the declaration of the current symbol.
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// Reference options.
type ReferenceOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// Parameters for a ReferencesRequest.
type ReferenceParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	Context            ReferenceContext        `json:"context"`
}

// Registration options for a ReferencesRequest.
type ReferenceRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
}

// General parameters to register for a notification or to register a provider.
type Registration struct {
	// The id used to register the request. The id can be used to deregister
	// the request again.
	ID string `json:"id"`
	// The method / capability to register for.
	Method string `json:"method"`
	// Options necessary for the registration.
	RegisterOptions Optional[LSPAny] `json:"registerOptions,omitzero"`
}

type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// Client capabilities specific to regular expressions.
// @since 3.16.0
// @since 3.16.0
type RegularExpressionsClientCapabilities struct {
	// The engine's name.
	Engine RegularExpressionEngineKind `json:"engine"`
	// The engine's version.
	Version Optional[string] `json:"version,omitzero"`
}

// A full diagnostic report with a set of related documents.
// @since 3.17.0
// @since 3.17.0
type RelatedFullDocumentDiagnosticReport struct {
	// A full document diagnostic report.
	Kind string `json:"kind"`
	// An optional result id. If provided it will
	// be sent on the next diagnostic request for the
	// same document.
	ResultID Optional[string] `json:"resultId,omitzero"`
	// The actual items.
	Items []Diagnostic `json:"items"`
	// Diagnostics of related documents. This information is useful
	// in programming languages where code in a file A can generate
	// diagnostics in a file B which A depends on. An example of
	// such a language is C/C++ where marco definitions in a file
	// a.cpp and result in errors in a header file b.hpp.
	// @since 3.17.0
	// @since 3.17.0
	RelatedDocuments Optional[map[DocumentURI]any] `json:"relatedDocuments,omitzero"`
}

// An unchanged diagnostic report with a set of related documents.
// @since 3.17.0
// @since 3.17.0
type RelatedUnchangedDocumentDiagnosticReport struct {
	// A document diagnostic report indicating
	// no changes to the last result. A server can
	// only return `unchanged` if result ids are
	// provided.
	Kind string `json:"kind"`
	// A result id which will be sent on the next
	// diagnostic request for the same document.
	ResultID string `json:"resultId"`
	// Diagnostics of related documents. This information is useful
	// in programming languages where code in a file A can generate
	// diagnostics in a file B which A depends on. An example of
	// such a language is C/C++ where marco definitions in a file
	// a.cpp and result in errors in a header file b.hpp.
	// @since 3.17.0
	// @since 3.17.0
	RelatedDocuments Optional[map[DocumentURI]any] `json:"relatedDocuments,omitzero"`
}

// A relative pattern is a helper to construct glob patterns that are matched
// relatively to a base URI. The common value for a `baseUri` is a workspace
// folder root, but it can be another absolute URI as well.
// @since 3.17.0
// @since 3.17.0
type RelativePattern struct {
	// A workspace folder or a base URI to which this pattern will be matched
	// against relatively.
	BaseURI OrRelativePatternBaseUri `json:"baseUri"`
	// The actual glob pattern;
	Pattern Pattern `json:"pattern"`
}

type RenameClientCapabilities struct {
	// Whether rename supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Client supports testing for validity of rename operations
	// before execution.
	// @since 3.12.0
	// @since 3.12.0
	PrepareSupport Optional[bool] `json:"prepareSupport,omitzero"`
	// Client supports the default behavior result.
	// The value indicates the default behavior used by the
	// client.
	// @since 3.16.0
	// @since 3.16.0
	PrepareSupportDefaultBehavior Optional[PrepareSupportDefaultBehavior] `json:"prepareSupportDefaultBehavior,omitzero"`
	// Whether the client honors the change annotations in
	// text edits and resource operations returned via the
	// rename request's workspace edit by for example presenting
	// the workspace edit in the user interface and asking
	// for confirmation.
	// @since 3.16.0
	// @since 3.16.0
	HonorsChangeAnnotations Optional[bool] `json:"honorsChangeAnnotations,omitzero"`
}

// Rename file operation
type RenameFile struct {
	// The resource operation kind.
	Kind string `json:"kind"`
	// An optional annotation identifier describing the operation.
	// @since 3.16.0
	// @since 3.16.0
	AnnotationID Optional[ChangeAnnotationIdentifier] `json:"annotationId,omitzero"`
	// The old (existing) location.
	OldURI DocumentURI `json:"oldUri"`
	// The new location.
	NewURI DocumentURI `json:"newUri"`
	// Rename options.
	Options Optional[RenameFileOptions] `json:"options,omitzero"`
}

// Rename file options
type RenameFileOptions struct {
	// Overwrite target if existing. Overwrite wins over `ignoreIfExists`
	Overwrite Optional[bool] `json:"overwrite,omitzero"`
	// Ignores if target exists.
	IgnoreIfExists Optional[bool] `json:"ignoreIfExists,omitzero"`
}

// The parameters sent in notifications/requests for user-initiated renames of
// files.
// @since 3.16.0
// @since 3.16.0
type RenameFilesParams struct {
	// An array of all files/folders renamed in this operation. When a folder is renamed, only
	// the folder will be included, and not its children.
	Files []FileRename `json:"files"`
}

// Provider options for a RenameRequest.
type RenameOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Renames should be checked and tested before being executed.
	// @since version 3.12.0
	// @since version 3.12.0
	PrepareProvider Optional[bool] `json:"prepareProvider,omitzero"`
}

// The parameters of a RenameRequest.
type RenameParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The new name of the symbol. If the given name is not valid the
	// request must return a ResponseError with an
	// appropriate message set.
	NewName string `json:"newName"`
}

// Registration options for a RenameRequest.
type RenameRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// Renames should be checked and tested before being executed.
	// @since version 3.12.0
	// @since version 3.12.0
	PrepareProvider Optional[bool] `json:"prepareProvider,omitzero"`
}

// A generic resource operation.
type ResourceOperation struct {
	// The resource operation kind.
	Kind string `json:"kind"`
	// An optional annotation identifier describing the operation.
	// @since 3.16.0
	// @since 3.16.0
	AnnotationID Optional[ChangeAnnotationIdentifier] `json:"annotationId,omitzero"`
}

// Save options.
type SaveOptions struct {
	// The client is supposed to include the content on save.
	IncludeText Optional[bool] `json:"includeText,omitzero"`
}

// Describes the currently selected completion item.
// @since 3.18.0
// @since 3.18.0
type SelectedCompletionInfo struct {
	// The range that will be replaced if this completion item is accepted.
	Range Range `json:"range"`
	// The text the range will be replaced with if this completion is accepted.
	Text string `json:"text"`
}

// A selection range represents a part of a selection hierarchy. A selection range
// may have a parent selection range that contains it.
type SelectionRange struct {
	// The Range range of this selection range.
	Range Range `json:"range"`
	// The parent selection range containing this range. Therefore `parent.range` must contain `this.range`.
	Parent Optional[SelectionRange] `json:"parent,omitzero"`
}

type SelectionRangeClientCapabilities struct {
	// Whether implementation supports dynamic registration for selection range providers. If this is set to `true`
	// the client supports the new `SelectionRangeRegistrationOptions` return value for the corresponding server
	// capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

type SelectionRangeOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// A parameter literal used in selection range requests.
type SelectionRangeParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The positions inside the text document.
	Positions []Position `json:"positions"`
}

type SelectionRangeRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokens struct {
	// An optional result id. If provided and clients support delta updating
	// the client will include the result id in the next semantic token request.
	// A server can then instead of computing all semantic tokens again simply
	// send a delta.
	ResultID Optional[string] `json:"resultId,omitzero"`
	// The actual tokens.
	Data []uint32 `json:"data"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Which requests the client supports and might send to the server
	// depending on the server's capability. Please note that clients might not
	// show semantic tokens or degrade some of the user experience if a range
	// or full request is advertised by the client but not provided by the
	// server. If for example the client capability `requests.full` and
	// `request.range` are both set to true but the server only provides a
	// range provider the client might not render a minimap correctly or might
	// even decide to not show any semantic tokens at all.
	Requests ClientSemanticTokensRequestOptions `json:"requests"`
	// The token types that the client supports.
	TokenTypes []string `json:"tokenTypes"`
	// The token modifiers that the client supports.
	TokenModifiers []string `json:"tokenModifiers"`
	// The token formats the clients supports.
	Formats []TokenFormat `json:"formats"`
	// Whether the client supports tokens that can overlap each other.
	OverlappingTokenSupport Optional[bool] `json:"overlappingTokenSupport,omitzero"`
	// Whether the client supports tokens that can span multiple lines.
	MultilineTokenSupport Optional[bool] `json:"multilineTokenSupport,omitzero"`
	// Whether the client allows the server to actively cancel a
	// semantic token request, e.g. supports returning
	// LSPErrorCodes.ServerCancelled. If a server does the client
	// needs to retrigger the request.
	// @since 3.17.0
	// @since 3.17.0
	ServerCancelSupport Optional[bool] `json:"serverCancelSupport,omitzero"`
	// Whether the client uses semantic tokens to augment existing
	// syntax tokens. If set to `true` client side created syntax
	// tokens and semantic tokens are both used for colorization. If
	// set to `false` the client only uses the returned semantic tokens
	// for colorization.
	// If the value is `undefined` then the client behavior is not
	// specified.
	// @since 3.17.0
	// @since 3.17.0
	AugmentsSyntaxTokens Optional[bool] `json:"augmentsSyntaxTokens,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensDelta struct {
	ResultID Optional[string] `json:"resultId,omitzero"`
	// The semantic token edits to transform a previous result into a new result.
	Edits []SemanticTokensEdit `json:"edits"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensDeltaParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The result id of a previous response. The result Id can either point to a full response
	// or a delta response depending on what was received last.
	PreviousResultID string `json:"previousResultId"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensDeltaPartialResult struct {
	Edits []SemanticTokensEdit `json:"edits"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensEdit struct {
	// The start offset of the edit.
	Start uint32 `json:"start"`
	// The count of elements to remove.
	DeleteCount uint32 `json:"deleteCount"`
	// The elements to insert.
	Data Optional[[]uint32] `json:"data,omitzero"`
}

// Semantic tokens options to support deltas for full documents
// @since 3.18.0
// @since 3.18.0
type SemanticTokensFullDelta struct {
	// The server supports deltas for full documents.
	Delta Optional[bool] `json:"delta,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensLegend struct {
	// The token types a server uses.
	TokenTypes []string `json:"tokenTypes"`
	// The token modifiers a server uses.
	TokenModifiers []string `json:"tokenModifiers"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The legend used by the server
	Legend SemanticTokensLegend `json:"legend"`
	// Server supports providing semantic tokens for a specific range
	// of a document.
	Range Optional[OrSemanticTokensOptionsRange] `json:"range,omitzero"`
	// Server supports providing semantic tokens for a full document.
	Full Optional[OrSemanticTokensOptionsFull] `json:"full,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensPartialResult struct {
	Data []uint32 `json:"data"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensRangeParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The range the semantic tokens are requested for.
	Range Range `json:"range"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The legend used by the server
	Legend SemanticTokensLegend `json:"legend"`
	// Server supports providing semantic tokens for a specific range
	// of a document.
	Range Optional[OrSemanticTokensOptionsRange] `json:"range,omitzero"`
	// Server supports providing semantic tokens for a full document.
	Full Optional[OrSemanticTokensOptionsFull] `json:"full,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// @since 3.16.0
// @since 3.16.0
type SemanticTokensWorkspaceClientCapabilities struct {
	// Whether the client implementation supports a refresh request sent from
	// the server to the client.
	// Note that this event is global and will force the client to refresh all
	// semantic tokens currently shown. It should be used with absolute care
	// and is useful for situation where a server for example detects a project
	// wide change that requires such a calculation.
	RefreshSupport Optional[bool] `json:"refreshSupport,omitzero"`
}

// Defines the capabilities provided by a language
// server.
type ServerCapabilities struct {
	// The position encoding the server picked from the encodings offered
	// by the client via the client capability `general.positionEncodings`.
	// If the client didn't provide any position encodings the only valid
	// value that a server can return is 'utf-16'.
	// If omitted it defaults to 'utf-16'.
	// @since 3.17.0
	// @since 3.17.0
	PositionEncoding Optional[PositionEncodingKind] `json:"positionEncoding,omitzero"`
	// Defines how text documents are synced. Is either a detailed structure
	// defining each notification or for backwards compatibility the
	// TextDocumentSyncKind number.
	TextDocumentSync Optional[OrServerCapabilitiesTextDocumentSync] `json:"textDocumentSync,omitzero"`
	// Defines how notebook documents are synced.
	// @since 3.17.0
	// @since 3.17.0
	NotebookDocumentSync Optional[OrServerCapabilitiesNotebookDocumentSync] `json:"notebookDocumentSync,omitzero"`
	// The server provides completion support.
	CompletionProvider Optional[CompletionOptions] `json:"completionProvider,omitzero"`
	// The server provides hover support.
	HoverProvider Optional[OrServerCapabilitiesHoverProvider] `json:"hoverProvider,omitzero"`
	// The server provides signature help support.
	SignatureHelpProvider Optional[SignatureHelpOptions] `json:"signatureHelpProvider,omitzero"`
	// The server provides Goto Declaration support.
	DeclarationProvider Optional[OrServerCapabilitiesDeclarationProvider] `json:"declarationProvider,omitzero"`
	// The server provides goto definition support.
	DefinitionProvider Optional[OrServerCapabilitiesDefinitionProvider] `json:"definitionProvider,omitzero"`
	// The server provides Goto Type Definition support.
	TypeDefinitionProvider Optional[OrServerCapabilitiesTypeDefinitionProvider] `json:"typeDefinitionProvider,omitzero"`
	// The server provides Goto Implementation support.
	ImplementationProvider Optional[OrServerCapabilitiesImplementationProvider] `json:"implementationProvider,omitzero"`
	// The server provides find references support.
	ReferencesProvider Optional[OrServerCapabilitiesReferencesProvider] `json:"referencesProvider,omitzero"`
	// The server provides document highlight support.
	DocumentHighlightProvider Optional[OrServerCapabilitiesDocumentHighlightProvider] `json:"documentHighlightProvider,omitzero"`
	// The server provides document symbol support.
	DocumentSymbolProvider Optional[OrServerCapabilitiesDocumentSymbolProvider] `json:"documentSymbolProvider,omitzero"`
	// The server provides code actions. CodeActionOptions may only be
	// specified if the client states that it supports
	// `codeActionLiteralSupport` in its initial `initialize` request.
	CodeActionProvider Optional[OrServerCapabilitiesCodeActionProvider] `json:"codeActionProvider,omitzero"`
	// The server provides code lens.
	CodeLensProvider Optional[CodeLensOptions] `json:"codeLensProvider,omitzero"`
	// The server provides document link support.
	DocumentLinkProvider Optional[DocumentLinkOptions] `json:"documentLinkProvider,omitzero"`
	// The server provides color provider support.
	ColorProvider Optional[OrServerCapabilitiesColorProvider] `json:"colorProvider,omitzero"`
	// The server provides workspace symbol support.
	WorkspaceSymbolProvider Optional[OrServerCapabilitiesWorkspaceSymbolProvider] `json:"workspaceSymbolProvider,omitzero"`
	// The server provides document formatting.
	DocumentFormattingProvider Optional[OrServerCapabilitiesDocumentFormattingProvider] `json:"documentFormattingProvider,omitzero"`
	// The server provides document range formatting.
	DocumentRangeFormattingProvider Optional[OrServerCapabilitiesDocumentRangeFormattingProvider] `json:"documentRangeFormattingProvider,omitzero"`
	// The server provides document formatting on typing.
	DocumentOnTypeFormattingProvider Optional[DocumentOnTypeFormattingOptions] `json:"documentOnTypeFormattingProvider,omitzero"`
	// The server provides rename support. RenameOptions may only be
	// specified if the client states that it supports
	// `prepareSupport` in its initial `initialize` request.
	RenameProvider Optional[OrServerCapabilitiesRenameProvider] `json:"renameProvider,omitzero"`
	// The server provides folding provider support.
	FoldingRangeProvider Optional[OrServerCapabilitiesFoldingRangeProvider] `json:"foldingRangeProvider,omitzero"`
	// The server provides selection range support.
	SelectionRangeProvider Optional[OrServerCapabilitiesSelectionRangeProvider] `json:"selectionRangeProvider,omitzero"`
	// The server provides execute command support.
	ExecuteCommandProvider Optional[ExecuteCommandOptions] `json:"executeCommandProvider,omitzero"`
	// The server provides call hierarchy support.
	// @since 3.16.0
	// @since 3.16.0
	CallHierarchyProvider Optional[OrServerCapabilitiesCallHierarchyProvider] `json:"callHierarchyProvider,omitzero"`
	// The server provides linked editing range support.
	// @since 3.16.0
	// @since 3.16.0
	LinkedEditingRangeProvider Optional[OrServerCapabilitiesLinkedEditingRangeProvider] `json:"linkedEditingRangeProvider,omitzero"`
	// The server provides semantic tokens support.
	// @since 3.16.0
	// @since 3.16.0
	SemanticTokensProvider Optional[OrServerCapabilitiesSemanticTokensProvider] `json:"semanticTokensProvider,omitzero"`
	// The server provides moniker support.
	// @since 3.16.0
	// @since 3.16.0
	MonikerProvider Optional[OrServerCapabilitiesMonikerProvider] `json:"monikerProvider,omitzero"`
	// The server provides type hierarchy support.
	// @since 3.17.0
	// @since 3.17.0
	TypeHierarchyProvider Optional[OrServerCapabilitiesTypeHierarchyProvider] `json:"typeHierarchyProvider,omitzero"`
	// The server provides inline values.
	// @since 3.17.0
	// @since 3.17.0
	InlineValueProvider Optional[OrServerCapabilitiesInlineValueProvider] `json:"inlineValueProvider,omitzero"`
	// The server provides inlay hints.
	// @since 3.17.0
	// @since 3.17.0
	InlayHintProvider Optional[OrServerCapabilitiesInlayHintProvider] `json:"inlayHintProvider,omitzero"`
	// The server has support for pull model diagnostics.
	// @since 3.17.0
	// @since 3.17.0
	DiagnosticProvider Optional[OrServerCapabilitiesDiagnosticProvider] `json:"diagnosticProvider,omitzero"`
	// Inline completion options used during static registration.
	// @since 3.18.0
	// @since 3.18.0
	InlineCompletionProvider Optional[OrServerCapabilitiesInlineCompletionProvider] `json:"inlineCompletionProvider,omitzero"`
	// Workspace specific server capabilities.
	Workspace Optional[WorkspaceOptions] `json:"workspace,omitzero"`
	// Experimental server capabilities.
	Experimental Optional[LSPAny] `json:"experimental,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type ServerCompletionItemOptions struct {
	// The server has support for completion item label
	// details (see also `CompletionItemLabelDetails`) when
	// receiving a completion item in a resolve call.
	// @since 3.17.0
	// @since 3.17.0
	LabelDetailsSupport Optional[bool] `json:"labelDetailsSupport,omitzero"`
}

// Information about the server
// @since 3.15.0
// @since 3.18.0 ServerInfo type name added.
// @since 3.18.0 ServerInfo type name added.
type ServerInfo struct {
	// The name of the server as defined by the server.
	Name string `json:"name"`
	// The server's version as defined by the server.
	Version Optional[string] `json:"version,omitzero"`
}

type SetTraceParams struct {
	Value TraceValue `json:"value"`
}

// Client capabilities for the showDocument request.
// @since 3.16.0
// @since 3.16.0
type ShowDocumentClientCapabilities struct {
	// The client has support for the showDocument
	// request.
	Support bool `json:"support"`
}

// Params to show a resource in the UI.
// @since 3.16.0
// @since 3.16.0
type ShowDocumentParams struct {
	// The uri to show.
	URI string `json:"uri"`
	// Indicates to show the resource in an external program.
	// To show, for example, `https://code.visualstudio.com/`
	// in the default WEB browser set `external` to `true`.
	External Optional[bool] `json:"external,omitzero"`
	// An optional property to indicate whether the editor
	// showing the document should take focus or not.
	// Clients might ignore this property if an external
	// program is started.
	TakeFocus Optional[bool] `json:"takeFocus,omitzero"`
	// An optional selection range if the document is a text
	// document. Clients might ignore the property if an
	// external program is started or the file is not a text
	// file.
	Selection Optional[Range] `json:"selection,omitzero"`
}

// The result of a showDocument request.
// @since 3.16.0
// @since 3.16.0
type ShowDocumentResult struct {
	// A boolean indicating if the show was successful.
	Success bool `json:"success"`
}

// The parameters of a notification message.
type ShowMessageParams struct {
	// The message type. See MessageType
	Type MessageType `json:"type"`
	// The actual message.
	Message string `json:"message"`
}

// Show message request client capabilities
type ShowMessageRequestClientCapabilities struct {
	// Capabilities specific to the `MessageActionItem` type.
	MessageActionItem Optional[ClientShowMessageActionItemOptions] `json:"messageActionItem,omitzero"`
}

type ShowMessageRequestParams struct {
	// The message type. See MessageType
	Type MessageType `json:"type"`
	// The actual message.
	Message string `json:"message"`
	// The message action items to present.
	Actions Optional[[]MessageActionItem] `json:"actions,omitzero"`
}

// Signature help represents the signature of something
// callable. There can be multiple signature but only one
// active and only one active parameter.
type SignatureHelp struct {
	// One or more signatures.
	Signatures []SignatureInformation `json:"signatures"`
	// The active signature. If omitted or the value lies outside the
	// range of `signatures` the value defaults to zero or is ignored if
	// the `SignatureHelp` has no signatures.
	// Whenever possible implementors should make an active decision about
	// the active signature and shouldn't rely on a default value.
	// In future version of the protocol this property might become
	// mandatory to better express this.
	ActiveSignature Optional[uint32] `json:"activeSignature,omitzero"`
	// The active parameter of the active signature.
	// If `null`, no parameter of the signature is active (for example a named
	// argument that does not match any declared parameters). This is only valid
	// if the client specifies the client capability
	// `textDocument.signatureHelp.noActiveParameterSupport === true`
	// If omitted or the value lies outside the range of
	// `signatures[activeSignature].parameters` defaults to 0 if the active
	// signature has parameters.
	// If the active signature has no parameters it is ignored.
	// In future version of the protocol this property might become
	// mandatory (but still nullable) to better express the active parameter if
	// the active signature does have any.
	// Since version 3.16.0 the `SignatureInformation` itself provides a
	// `activeParameter` property and it should be used instead of this one.
	ActiveParameter OptionalNullable[OrSignatureHelpActiveParameter] `json:"activeParameter,omitzero"`
}

// Client Capabilities for a SignatureHelpRequest.
type SignatureHelpClientCapabilities struct {
	// Whether signature help supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports the following `SignatureInformation`
	// specific properties.
	SignatureInformation Optional[ClientSignatureInformationOptions] `json:"signatureInformation,omitzero"`
	// The client supports to send additional context information for a
	// `textDocument/signatureHelp` request. A client that opts into
	// contextSupport will also support the `retriggerCharacters` on
	// `SignatureHelpOptions`.
	// @since 3.15.0
	// @since 3.15.0
	ContextSupport Optional[bool] `json:"contextSupport,omitzero"`
}

// Additional information about the context in which a signature help request was triggered.
// @since 3.15.0
// @since 3.15.0
type SignatureHelpContext struct {
	// Action that caused signature help to be triggered.
	TriggerKind SignatureHelpTriggerKind `json:"triggerKind"`
	// Character that caused signature help to be triggered.
	// This is undefined when `triggerKind !== SignatureHelpTriggerKind.TriggerCharacter`
	TriggerCharacter Optional[string] `json:"triggerCharacter,omitzero"`
	// `true` if signature help was already showing when it was triggered.
	// Retriggers occurs when the signature help is already active and can be caused by actions such as
	// typing a trigger character, a cursor move, or document content changes.
	IsRetrigger bool `json:"isRetrigger"`
	// The currently active `SignatureHelp`.
	// The `activeSignatureHelp` has its `SignatureHelp.activeSignature` field updated based on
	// the user navigating through available signatures.
	ActiveSignatureHelp Optional[SignatureHelp] `json:"activeSignatureHelp,omitzero"`
}

// Server Capabilities for a SignatureHelpRequest.
type SignatureHelpOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// List of characters that trigger signature help automatically.
	TriggerCharacters Optional[[]string] `json:"triggerCharacters,omitzero"`
	// List of characters that re-trigger signature help.
	// These trigger characters are only active when signature help is already showing. All trigger characters
	// are also counted as re-trigger characters.
	// @since 3.15.0
	// @since 3.15.0
	RetriggerCharacters Optional[[]string] `json:"retriggerCharacters,omitzero"`
}

// Parameters for a SignatureHelpRequest.
type SignatureHelpParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The signature help context. This is only available if the client specifies
	// to send this using the client capability `textDocument.signatureHelp.contextSupport === true`
	// @since 3.15.0
	// @since 3.15.0
	Context Optional[SignatureHelpContext] `json:"context,omitzero"`
}

// Registration options for a SignatureHelpRequest.
type SignatureHelpRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// List of characters that trigger signature help automatically.
	TriggerCharacters Optional[[]string] `json:"triggerCharacters,omitzero"`
	// List of characters that re-trigger signature help.
	// These trigger characters are only active when signature help is already showing. All trigger characters
	// are also counted as re-trigger characters.
	// @since 3.15.0
	// @since 3.15.0
	RetriggerCharacters Optional[[]string] `json:"retriggerCharacters,omitzero"`
}

// Represents the signature of something callable. A signature
// can have a label, like a function-name, a doc-comment, and
// a set of parameters.
type SignatureInformation struct {
	// The label of this signature. Will be shown in
	// the UI.
	Label string `json:"label"`
	// The human-readable doc-comment of this signature. Will be shown
	// in the UI but can be omitted.
	Documentation Optional[OrSignatureInformationDocumentation] `json:"documentation,omitzero"`
	// The parameters of this signature.
	Parameters Optional[[]ParameterInformation] `json:"parameters,omitzero"`
	// The index of the active parameter.
	// If `null`, no parameter of the signature is active (for example a named
	// argument that does not match any declared parameters). This is only valid
	// if the client specifies the client capability
	// `textDocument.signatureHelp.noActiveParameterSupport === true`
	// If provided (or `null`), this is used in place of
	// `SignatureHelp.activeParameter`.
	// @since 3.16.0
	// @since 3.16.0
	ActiveParameter OptionalNullable[OrSignatureInformationActiveParameter] `json:"activeParameter,omitzero"`
}

// An interactive text edit.
// @since 3.18.0
// @since 3.18.0
type SnippetTextEdit struct {
	// The range of the text document to be manipulated.
	Range Range `json:"range"`
	// The snippet to be inserted.
	Snippet StringValue `json:"snippet"`
	// The actual identifier of the snippet edit.
	AnnotationID Optional[ChangeAnnotationIdentifier] `json:"annotationId,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type StaleRequestSupportOptions struct {
	// The client will actively cancel the request.
	Cancel bool `json:"cancel"`
	// The list of requests for which the client
	// will retry the request if it receives a
	// response with error code `ContentModified`
	RetryOnContentModified []string `json:"retryOnContentModified"`
}

// Static registration options to be returned in the initialize
// request.
type StaticRegistrationOptions struct {
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// A string value used as a snippet is a template which allows to insert text
// and to control the editor cursor when insertion happens.
// A snippet can define tab stops and placeholders with `$1`, `$2`
// and `${3:foo`. `$0` defines the final tab stop, it defaults to
// the end of the snippet. Variables are defined with `$name` and
// `${name:default value`.
// @since 3.18.0
// @since 3.18.0
type StringValue struct {
	// The kind of string value.
	Kind string `json:"kind"`
	// The snippet string.
	Value string `json:"value"`
}

// Represents information about programming constructs like variables, classes,
// interfaces etc.
type SymbolInformation struct {
	// The name of this symbol.
	Name string `json:"name"`
	// The kind of this symbol.
	Kind SymbolKind `json:"kind"`
	// Tags for this symbol.
	// @since 3.16.0
	// @since 3.16.0
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// The name of the symbol containing this symbol. This information is for
	// user interface purposes (e.g. to render a qualifier in the user interface
	// if necessary). It can't be used to re-infer a hierarchy for the document
	// symbols.
	ContainerName Optional[string] `json:"containerName,omitzero"`
	// Indicates if this symbol is deprecated.
	// @deprecated Use tags instead
	// Deprecated: Use tags instead
	Deprecated Optional[bool] `json:"deprecated,omitzero"`
	// The location of this symbol. The location's range is used by a tool
	// to reveal the location in the editor. If the symbol is selected in the
	// tool the range's start information is used to position the cursor. So
	// the range usually spans more than the actual symbol's name and does
	// normally include things like visibility modifiers.
	// The range doesn't have to denote a node range in the sense of an abstract
	// syntax tree. It can therefore not be used to re-construct a hierarchy of
	// the symbols.
	Location Location `json:"location"`
}

// Describe options to be used when registered for text document change events.
type TextDocumentChangeRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// How documents are synced to the server.
	SyncKind TextDocumentSyncKind `json:"syncKind"`
}

// Text document specific client capabilities.
type TextDocumentClientCapabilities struct {
	// Defines which synchronization capabilities the client supports.
	Synchronization Optional[TextDocumentSyncClientCapabilities] `json:"synchronization,omitzero"`
	// Defines which filters the client supports.
	// @since 3.18.0
	// @since 3.18.0
	Filters Optional[TextDocumentFilterClientCapabilities] `json:"filters,omitzero"`
	// Capabilities specific to the `textDocument/completion` request.
	Completion Optional[CompletionClientCapabilities] `json:"completion,omitzero"`
	// Capabilities specific to the `textDocument/hover` request.
	Hover Optional[HoverClientCapabilities] `json:"hover,omitzero"`
	// Capabilities specific to the `textDocument/signatureHelp` request.
	SignatureHelp Optional[SignatureHelpClientCapabilities] `json:"signatureHelp,omitzero"`
	// Capabilities specific to the `textDocument/declaration` request.
	// @since 3.14.0
	// @since 3.14.0
	Declaration Optional[DeclarationClientCapabilities] `json:"declaration,omitzero"`
	// Capabilities specific to the `textDocument/definition` request.
	Definition Optional[DefinitionClientCapabilities] `json:"definition,omitzero"`
	// Capabilities specific to the `textDocument/typeDefinition` request.
	// @since 3.6.0
	// @since 3.6.0
	TypeDefinition Optional[TypeDefinitionClientCapabilities] `json:"typeDefinition,omitzero"`
	// Capabilities specific to the `textDocument/implementation` request.
	// @since 3.6.0
	// @since 3.6.0
	Implementation Optional[ImplementationClientCapabilities] `json:"implementation,omitzero"`
	// Capabilities specific to the `textDocument/references` request.
	References Optional[ReferenceClientCapabilities] `json:"references,omitzero"`
	// Capabilities specific to the `textDocument/documentHighlight` request.
	DocumentHighlight Optional[DocumentHighlightClientCapabilities] `json:"documentHighlight,omitzero"`
	// Capabilities specific to the `textDocument/documentSymbol` request.
	DocumentSymbol Optional[DocumentSymbolClientCapabilities] `json:"documentSymbol,omitzero"`
	// Capabilities specific to the `textDocument/codeAction` request.
	CodeAction Optional[CodeActionClientCapabilities] `json:"codeAction,omitzero"`
	// Capabilities specific to the `textDocument/codeLens` request.
	CodeLens Optional[CodeLensClientCapabilities] `json:"codeLens,omitzero"`
	// Capabilities specific to the `textDocument/documentLink` request.
	DocumentLink Optional[DocumentLinkClientCapabilities] `json:"documentLink,omitzero"`
	// Capabilities specific to the `textDocument/documentColor` and the
	// `textDocument/colorPresentation` request.
	// @since 3.6.0
	// @since 3.6.0
	ColorProvider Optional[DocumentColorClientCapabilities] `json:"colorProvider,omitzero"`
	// Capabilities specific to the `textDocument/formatting` request.
	Formatting Optional[DocumentFormattingClientCapabilities] `json:"formatting,omitzero"`
	// Capabilities specific to the `textDocument/rangeFormatting` request.
	RangeFormatting Optional[DocumentRangeFormattingClientCapabilities] `json:"rangeFormatting,omitzero"`
	// Capabilities specific to the `textDocument/onTypeFormatting` request.
	OnTypeFormatting Optional[DocumentOnTypeFormattingClientCapabilities] `json:"onTypeFormatting,omitzero"`
	// Capabilities specific to the `textDocument/rename` request.
	Rename Optional[RenameClientCapabilities] `json:"rename,omitzero"`
	// Capabilities specific to the `textDocument/foldingRange` request.
	// @since 3.10.0
	// @since 3.10.0
	FoldingRange Optional[FoldingRangeClientCapabilities] `json:"foldingRange,omitzero"`
	// Capabilities specific to the `textDocument/selectionRange` request.
	// @since 3.15.0
	// @since 3.15.0
	SelectionRange Optional[SelectionRangeClientCapabilities] `json:"selectionRange,omitzero"`
	// Capabilities specific to the `textDocument/publishDiagnostics` notification.
	PublishDiagnostics Optional[PublishDiagnosticsClientCapabilities] `json:"publishDiagnostics,omitzero"`
	// Capabilities specific to the various call hierarchy requests.
	// @since 3.16.0
	// @since 3.16.0
	CallHierarchy Optional[CallHierarchyClientCapabilities] `json:"callHierarchy,omitzero"`
	// Capabilities specific to the various semantic token request.
	// @since 3.16.0
	// @since 3.16.0
	SemanticTokens Optional[SemanticTokensClientCapabilities] `json:"semanticTokens,omitzero"`
	// Capabilities specific to the `textDocument/linkedEditingRange` request.
	// @since 3.16.0
	// @since 3.16.0
	LinkedEditingRange Optional[LinkedEditingRangeClientCapabilities] `json:"linkedEditingRange,omitzero"`
	// Client capabilities specific to the `textDocument/moniker` request.
	// @since 3.16.0
	// @since 3.16.0
	Moniker Optional[MonikerClientCapabilities] `json:"moniker,omitzero"`
	// Capabilities specific to the various type hierarchy requests.
	// @since 3.17.0
	// @since 3.17.0
	TypeHierarchy Optional[TypeHierarchyClientCapabilities] `json:"typeHierarchy,omitzero"`
	// Capabilities specific to the `textDocument/inlineValue` request.
	// @since 3.17.0
	// @since 3.17.0
	InlineValue Optional[InlineValueClientCapabilities] `json:"inlineValue,omitzero"`
	// Capabilities specific to the `textDocument/inlayHint` request.
	// @since 3.17.0
	// @since 3.17.0
	InlayHint Optional[InlayHintClientCapabilities] `json:"inlayHint,omitzero"`
	// Capabilities specific to the diagnostic pull model.
	// @since 3.17.0
	// @since 3.17.0
	Diagnostic Optional[DiagnosticClientCapabilities] `json:"diagnostic,omitzero"`
	// Client capabilities specific to inline completions.
	// @since 3.18.0
	// @since 3.18.0
	InlineCompletion Optional[InlineCompletionClientCapabilities] `json:"inlineCompletion,omitzero"`
}

// @since 3.18.0
// @since 3.18.0
type TextDocumentContentChangePartial struct {
	// The range of the document that changed.
	Range Range `json:"range"`
	// The optional length of the range that got replaced.
	// @deprecated use range instead.
	// Deprecated: use range instead.
	RangeLength Optional[uint32] `json:"rangeLength,omitzero"`
	// The new text for the provided range.
	Text string `json:"text"`
}

// @since 3.18.0
// @since 3.18.0
type TextDocumentContentChangeWholeDocument struct {
	// The new text of the whole document.
	Text string `json:"text"`
}

// Client capabilities for a text document content provider.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentClientCapabilities struct {
	// Text document content provider supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// Text document content provider options.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentOptions struct {
	// The schemes for which the server provides content.
	Schemes []string `json:"schemes"`
}

// Parameters for the `workspace/textDocumentContent` request.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentParams struct {
	// The uri of the text document.
	URI DocumentURI `json:"uri"`
}

// Parameters for the `workspace/textDocumentContent/refresh` request.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentRefreshParams struct {
	// The uri of the text document to refresh.
	URI DocumentURI `json:"uri"`
}

// Text document content provider registration options.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentRegistrationOptions struct {
	// The schemes for which the server provides content.
	Schemes []string `json:"schemes"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// Result of the `workspace/textDocumentContent` request.
// @since 3.18.0
// @since 3.18.0
type TextDocumentContentResult struct {
	// The text content of the text document. Please note, that the content of
	// any subsequent open notifications for the text document might differ
	// from the returned content due to whitespace and line ending
	// normalizations done on the client
	Text string `json:"text"`
}

// Describes textual changes on a text document. A TextDocumentEdit describes all changes
// on a document version Si and after they are applied move the document to version Si+1.
// So the creator of a TextDocumentEdit doesn't need to sort the array of edits or do any
// kind of ordering. However the edits must be non overlapping.
type TextDocumentEdit struct {
	// The text document to change.
	TextDocument OptionalVersionedTextDocumentIdentifier `json:"textDocument"`
	// The edits to be applied.
	// @since 3.16.0 - support for AnnotatedTextEdit. This is guarded using a
	// client capability.
	// @since 3.18.0 - support for SnippetTextEdit. This is guarded using a
	// client capability.
	// @since 3.18.0 - support for SnippetTextEdit. This is guarded using a
	// client capability.
	Edits []OrTextDocumentEditEditsElem `json:"edits"`
}

type TextDocumentFilterClientCapabilities struct {
	// The client supports Relative Patterns.
	// @since 3.18.0
	// @since 3.18.0
	RelativePatternSupport Optional[bool] `json:"relativePatternSupport,omitzero"`
}

// A document filter where `language` is required field.
// @since 3.18.0
// @since 3.18.0
type TextDocumentFilterLanguage struct {
	// A language id, like `typescript`.
	Language string `json:"language"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme Optional[string] `json:"scheme,omitzero"`
	// A glob pattern, like **​/*.{ts,js. See TextDocumentFilter for examples.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	Pattern Optional[GlobPattern] `json:"pattern,omitzero"`
}

// A document filter where `pattern` is required field.
// @since 3.18.0
// @since 3.18.0
type TextDocumentFilterPattern struct {
	// A language id, like `typescript`.
	Language Optional[string] `json:"language,omitzero"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme Optional[string] `json:"scheme,omitzero"`
	// A glob pattern, like **​/*.{ts,js. See TextDocumentFilter for examples.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	Pattern GlobPattern `json:"pattern"`
}

// A document filter where `scheme` is required field.
// @since 3.18.0
// @since 3.18.0
type TextDocumentFilterScheme struct {
	// A language id, like `typescript`.
	Language Optional[string] `json:"language,omitzero"`
	// A Uri Uri.scheme scheme, like `file` or `untitled`.
	Scheme string `json:"scheme"`
	// A glob pattern, like **​/*.{ts,js. See TextDocumentFilter for examples.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	// @since 3.18.0 - support for relative patterns. Whether clients support
	// relative patterns depends on the client capability
	// `textDocuments.filters.relativePatternSupport`.
	Pattern Optional[GlobPattern] `json:"pattern,omitzero"`
}

// A literal to identify a text document in the client.
type TextDocumentIdentifier struct {
	// The text document's uri.
	URI DocumentURI `json:"uri"`
}

// An item to transfer a text document from the client to the
// server.
type TextDocumentItem struct {
	// The text document's uri.
	URI DocumentURI `json:"uri"`
	// The text document's language identifier.
	LanguageID LanguageKind `json:"languageId"`
	// The version number of this document (it will increase after each
	// change, including undo/redo).
	Version int32 `json:"version"`
	// The content of the opened text document.
	Text string `json:"text"`
}

// A parameter literal used in requests to pass a text document and a position inside that
// document.
type TextDocumentPositionParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
}

// General text document registration options.
type TextDocumentRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
}

// Save registration options.
type TextDocumentSaveRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	// The client is supposed to include the content on save.
	IncludeText Optional[bool] `json:"includeText,omitzero"`
}

type TextDocumentSyncClientCapabilities struct {
	// Whether text document synchronization supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports sending will save notifications.
	WillSave Optional[bool] `json:"willSave,omitzero"`
	// The client supports sending a will save request and
	// waits for a response providing text edits which will
	// be applied to the document before it is saved.
	WillSaveWaitUntil Optional[bool] `json:"willSaveWaitUntil,omitzero"`
	// The client supports did save notifications.
	DidSave Optional[bool] `json:"didSave,omitzero"`
}

type TextDocumentSyncOptions struct {
	// Open and close notifications are sent to the server. If omitted open close notification should not
	// be sent.
	OpenClose Optional[bool] `json:"openClose,omitzero"`
	// Change notifications are sent to the server. See TextDocumentSyncKind.None, TextDocumentSyncKind.Full
	// and TextDocumentSyncKind.Incremental. If omitted it defaults to TextDocumentSyncKind.None.
	Change Optional[TextDocumentSyncKind] `json:"change,omitzero"`
	// If present will save notifications are sent to the server. If omitted the notification should not be
	// sent.
	WillSave Optional[bool] `json:"willSave,omitzero"`
	// If present will save wait until requests are sent to the server. If omitted the request should not be
	// sent.
	WillSaveWaitUntil Optional[bool] `json:"willSaveWaitUntil,omitzero"`
	// If present save notifications are sent to the server. If omitted the notification should not be
	// sent.
	Save Optional[OrTextDocumentSyncOptionsSave] `json:"save,omitzero"`
}

// A text edit applicable to a text document.
type TextEdit struct {
	// The range of the text document to be manipulated. To insert
	// text into a document create a range where start === end.
	Range Range `json:"range"`
	// The string to be inserted. For delete operations use an
	// empty string.
	NewText string `json:"newText"`
}

// Since 3.6.0
type TypeDefinitionClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `TypeDefinitionRegistrationOptions` return value
	// for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// The client supports additional metadata in the form of definition links.
	// Since 3.14.0
	LinkSupport Optional[bool] `json:"linkSupport,omitzero"`
}

type TypeDefinitionOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type TypeDefinitionParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
}

type TypeDefinitionRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// @since 3.17.0
// @since 3.17.0
type TypeHierarchyClientCapabilities struct {
	// Whether implementation supports dynamic registration. If this is set to `true`
	// the client supports the new `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	// return value for the corresponding server capability as well.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
}

// @since 3.17.0
// @since 3.17.0
type TypeHierarchyItem struct {
	// The name of this item.
	Name string `json:"name"`
	// The kind of this item.
	Kind SymbolKind `json:"kind"`
	// Tags for this item.
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// More detail for this item, e.g. the signature of a function.
	Detail Optional[string] `json:"detail,omitzero"`
	// The resource identifier of this item.
	URI DocumentURI `json:"uri"`
	// The range enclosing this symbol not including leading/trailing whitespace
	// but everything else, e.g. comments and code.
	Range Range `json:"range"`
	// The range that should be selected and revealed when this symbol is being
	// picked, e.g. the name of a function. Must be contained by the
	// TypeHierarchyItem.range `range`.
	SelectionRange Range `json:"selectionRange"`
	// A data entry field that is preserved between a type hierarchy prepare and
	// supertypes or subtypes requests. It could also be used to identify the
	// type hierarchy in the server, helping improve the performance on
	// resolving supertypes and subtypes.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Type hierarchy options used during static registration.
// @since 3.17.0
// @since 3.17.0
type TypeHierarchyOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

// The parameter of a `textDocument/prepareTypeHierarchy` request.
// @since 3.17.0
// @since 3.17.0
type TypeHierarchyPrepareParams struct {
	// The text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The position inside the text document.
	Position Position `json:"position"`
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

// Type hierarchy options used during static or dynamic registration.
// @since 3.17.0
// @since 3.17.0
type TypeHierarchyRegistrationOptions struct {
	// A document selector to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector Nullable[OrTextDocumentRegistrationOptionsDocumentSelector] `json:"documentSelector"`
	WorkDoneProgress Optional[bool]                                              `json:"workDoneProgress,omitzero"`
	// The id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID Optional[string] `json:"id,omitzero"`
}

// The parameter of a `typeHierarchy/subtypes` request.
// @since 3.17.0
// @since 3.17.0
type TypeHierarchySubtypesParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	Item               TypeHierarchyItem       `json:"item"`
}

// The parameter of a `typeHierarchy/supertypes` request.
// @since 3.17.0
// @since 3.17.0
type TypeHierarchySupertypesParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	Item               TypeHierarchyItem       `json:"item"`
}

// A diagnostic report indicating that the last returned
// report is still accurate.
// @since 3.17.0
// @since 3.17.0
type UnchangedDocumentDiagnosticReport struct {
	// A document diagnostic report indicating
	// no changes to the last result. A server can
	// only return `unchanged` if result ids are
	// provided.
	Kind string `json:"kind"`
	// A result id which will be sent on the next
	// diagnostic request for the same document.
	ResultID string `json:"resultId"`
}

// General parameters to unregister a request or notification.
type Unregistration struct {
	// The id used to unregister the request or notification. Usually an id
	// provided during the register request.
	ID string `json:"id"`
	// The method to unregister for.
	Method string `json:"method"`
}

type UnregistrationParams struct {
	Unregisterations []Unregistration `json:"unregisterations"`
}

// A versioned notebook document identifier.
// @since 3.17.0
// @since 3.17.0
type VersionedNotebookDocumentIdentifier struct {
	// The version number of this notebook document.
	Version int32 `json:"version"`
	// The notebook document's uri.
	URI string `json:"uri"`
}

// A text document identifier to denote a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	// The text document's uri.
	URI DocumentURI `json:"uri"`
	// The version number of this document.
	Version int32 `json:"version"`
}

// The parameters sent in a will save text document notification.
type WillSaveTextDocumentParams struct {
	// The document that will be saved.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	// The 'TextDocumentSaveReason'.
	Reason TextDocumentSaveReason `json:"reason"`
}

type WindowClientCapabilities struct {
	// It indicates whether the client supports server initiated
	// progress using the `window/workDoneProgress/create` request.
	// The capability also controls Whether client supports handling
	// of progress notifications. If set servers are allowed to report a
	// `workDoneProgress` property in the request specific server
	// capabilities.
	// @since 3.15.0
	// @since 3.15.0
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// Capabilities specific to the showMessage request.
	// @since 3.16.0
	// @since 3.16.0
	ShowMessage Optional[ShowMessageRequestClientCapabilities] `json:"showMessage,omitzero"`
	// Capabilities specific to the showDocument request.
	// @since 3.16.0
	// @since 3.16.0
	ShowDocument Optional[ShowDocumentClientCapabilities] `json:"showDocument,omitzero"`
}

type WorkDoneProgressBegin struct {
	Kind string `json:"kind"`
	// Mandatory title of the progress operation. Used to briefly inform about
	// the kind of operation being performed.
	// Examples: "Indexing" or "Linking dependencies".
	Title string `json:"title"`
	// Controls if a cancel button should show to allow the user to cancel the
	// long running operation. Clients that don't support cancellation are allowed
	// to ignore the setting.
	Cancellable Optional[bool] `json:"cancellable,omitzero"`
	// Optional, more detailed associated progress message. Contains
	// complementary information to the `title`.
	// Examples: "3/25 files", "project/src/module2", "node_modules/some_dep".
	// If unset, the previous progress message (if any) is still valid.
	Message Optional[string] `json:"message,omitzero"`
	// Optional progress percentage to display (value 100 is considered 100%).
	// If not provided infinite progress is assumed and clients are allowed
	// to ignore the `percentage` value in subsequent in report notifications.
	// The value should be steadily rising. Clients are free to ignore values
	// that are not following this rule. The value range is [0, 100].
	Percentage Optional[uint32] `json:"percentage,omitzero"`
}

type WorkDoneProgressCancelParams struct {
	// The token to be used to report progress.
	Token ProgressToken `json:"token"`
}

type WorkDoneProgressCreateParams struct {
	// The token to be used to report progress.
	Token ProgressToken `json:"token"`
}

type WorkDoneProgressEnd struct {
	Kind string `json:"kind"`
	// Optional, a final message indicating to for example indicate the outcome
	// of the operation.
	Message Optional[string] `json:"message,omitzero"`
}

type WorkDoneProgressOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
}

type WorkDoneProgressParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
}

type WorkDoneProgressReport struct {
	Kind string `json:"kind"`
	// Controls enablement state of a cancel button.
	// Clients that don't support cancellation or don't support controlling the button's
	// enablement state are allowed to ignore the property.
	Cancellable Optional[bool] `json:"cancellable,omitzero"`
	// Optional, more detailed associated progress message. Contains
	// complementary information to the `title`.
	// Examples: "3/25 files", "project/src/module2", "node_modules/some_dep".
	// If unset, the previous progress message (if any) is still valid.
	Message Optional[string] `json:"message,omitzero"`
	// Optional progress percentage to display (value 100 is considered 100%).
	// If not provided infinite progress is assumed and clients are allowed
	// to ignore the `percentage` value in subsequent in report notifications.
	// The value should be steadily rising. Clients are free to ignore values
	// that are not following this rule. The value range is [0, 100]
	Percentage Optional[uint32] `json:"percentage,omitzero"`
}

// Workspace specific client capabilities.
type WorkspaceClientCapabilities struct {
	// The client supports applying batch edits
	// to the workspace by supporting the request
	// 'workspace/applyEdit'
	ApplyEdit Optional[bool] `json:"applyEdit,omitzero"`
	// Capabilities specific to `WorkspaceEdit`s.
	WorkspaceEdit Optional[WorkspaceEditClientCapabilities] `json:"workspaceEdit,omitzero"`
	// Capabilities specific to the `workspace/didChangeConfiguration` notification.
	DidChangeConfiguration Optional[DidChangeConfigurationClientCapabilities] `json:"didChangeConfiguration,omitzero"`
	// Capabilities specific to the `workspace/didChangeWatchedFiles` notification.
	DidChangeWatchedFiles Optional[DidChangeWatchedFilesClientCapabilities] `json:"didChangeWatchedFiles,omitzero"`
	// Capabilities specific to the `workspace/symbol` request.
	Symbol Optional[WorkspaceSymbolClientCapabilities] `json:"symbol,omitzero"`
	// Capabilities specific to the `workspace/executeCommand` request.
	ExecuteCommand Optional[ExecuteCommandClientCapabilities] `json:"executeCommand,omitzero"`
	// The client has support for workspace folders.
	// @since 3.6.0
	// @since 3.6.0
	WorkspaceFolders Optional[bool] `json:"workspaceFolders,omitzero"`
	// The client supports `workspace/configuration` requests.
	// @since 3.6.0
	// @since 3.6.0
	Configuration Optional[bool] `json:"configuration,omitzero"`
	// Capabilities specific to the semantic token requests scoped to the
	// workspace.
	// @since 3.16.0.
	// @since 3.16.0.
	SemanticTokens Optional[SemanticTokensWorkspaceClientCapabilities] `json:"semanticTokens,omitzero"`
	// Capabilities specific to the code lens requests scoped to the
	// workspace.
	// @since 3.16.0.
	// @since 3.16.0.
	CodeLens Optional[CodeLensWorkspaceClientCapabilities] `json:"codeLens,omitzero"`
	// The client has support for file notifications/requests for user operations on files.
	// Since 3.16.0
	FileOperations Optional[FileOperationClientCapabilities] `json:"fileOperations,omitzero"`
	// Capabilities specific to the inline values requests scoped to the
	// workspace.
	// @since 3.17.0.
	// @since 3.17.0.
	InlineValue Optional[InlineValueWorkspaceClientCapabilities] `json:"inlineValue,omitzero"`
	// Capabilities specific to the inlay hint requests scoped to the
	// workspace.
	// @since 3.17.0.
	// @since 3.17.0.
	InlayHint Optional[InlayHintWorkspaceClientCapabilities] `json:"inlayHint,omitzero"`
	// Capabilities specific to the diagnostic requests scoped to the
	// workspace.
	// @since 3.17.0.
	// @since 3.17.0.
	Diagnostics Optional[DiagnosticWorkspaceClientCapabilities] `json:"diagnostics,omitzero"`
	// Capabilities specific to the folding range requests scoped to the workspace.
	// @since 3.18.0
	// @since 3.18.0
	FoldingRange Optional[FoldingRangeWorkspaceClientCapabilities] `json:"foldingRange,omitzero"`
	// Capabilities specific to the `workspace/textDocumentContent` request.
	// @since 3.18.0
	// @since 3.18.0
	TextDocumentContent Optional[TextDocumentContentClientCapabilities] `json:"textDocumentContent,omitzero"`
}

// Parameters of the workspace diagnostic request.
// @since 3.17.0
// @since 3.17.0
type WorkspaceDiagnosticParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// The additional identifier provided during registration.
	Identifier Optional[string] `json:"identifier,omitzero"`
	// The currently known diagnostic reports with their
	// previous result ids.
	PreviousResultIds []PreviousResultID `json:"previousResultIds"`
}

// A workspace diagnostic report.
// @since 3.17.0
// @since 3.17.0
type WorkspaceDiagnosticReport struct {
	Items []WorkspaceDocumentDiagnosticReport `json:"items"`
}

// A partial result for a workspace diagnostic report.
// @since 3.17.0
// @since 3.17.0
type WorkspaceDiagnosticReportPartialResult struct {
	Items []WorkspaceDocumentDiagnosticReport `json:"items"`
}

// A workspace edit represents changes to many resources managed in the workspace. The edit
// should either provide `changes` or `documentChanges`. If documentChanges are present
// they are preferred over `changes` if the client can handle versioned document edits.
// Since version 3.13.0 a workspace edit can contain resource operations as well. If resource
// operations are present clients need to execute the operations in the order in which they
// are provided. So a workspace edit for example can consist of the following two changes:
// (1) a create file a.txt and (2) a text document edit which insert text into file a.txt.
// An invalid sequence (e.g. (1) delete file a.txt and (2) insert text into file a.txt) will
// cause failure of the operation. How the client recovers from the failure is described by
// the client capability: `workspace.workspaceEdit.failureHandling`
type WorkspaceEdit struct {
	// Holds changes to existing resources.
	Changes Optional[map[DocumentURI]any] `json:"changes,omitzero"`
	// Depending on the client capability `workspace.workspaceEdit.resourceOperations` document changes
	// are either an array of `TextDocumentEdit`s to express changes to n different text documents
	// where each text document edit addresses a specific version of a text document. Or it can contain
	// above `TextDocumentEdit`s mixed with create, rename and delete file / folder operations.
	// Whether a client supports versioned document edits is expressed via
	// `workspace.workspaceEdit.documentChanges` client capability.
	// If a client neither supports `documentChanges` nor `workspace.workspaceEdit.resourceOperations` then
	// only plain `TextEdit`s using the `changes` property are supported.
	DocumentChanges Optional[[]OrWorkspaceEditDocumentChangesElem] `json:"documentChanges,omitzero"`
	// A map of change annotations that can be referenced in `AnnotatedTextEdit`s or create, rename and
	// delete file / folder operations.
	// Whether clients honor this property depends on the client capability `workspace.changeAnnotationSupport`.
	// @since 3.16.0
	// @since 3.16.0
	ChangeAnnotations Optional[map[ChangeAnnotationIdentifier]any] `json:"changeAnnotations,omitzero"`
}

type WorkspaceEditClientCapabilities struct {
	// The client supports versioned document changes in `WorkspaceEdit`s
	DocumentChanges Optional[bool] `json:"documentChanges,omitzero"`
	// The resource operations the client supports. Clients should at least
	// support 'create', 'rename' and 'delete' files and folders.
	// @since 3.13.0
	// @since 3.13.0
	ResourceOperations Optional[[]ResourceOperationKind] `json:"resourceOperations,omitzero"`
	// The failure handling strategy of a client if applying the workspace edit
	// fails.
	// @since 3.13.0
	// @since 3.13.0
	FailureHandling Optional[FailureHandlingKind] `json:"failureHandling,omitzero"`
	// Whether the client normalizes line endings to the client specific
	// setting.
	// If set to `true` the client will normalize line ending characters
	// in a workspace edit to the client-specified new line
	// character.
	// @since 3.16.0
	// @since 3.16.0
	NormalizesLineEndings Optional[bool] `json:"normalizesLineEndings,omitzero"`
	// Whether the client in general supports change annotations on text edits,
	// create file, rename file and delete file changes.
	// @since 3.16.0
	// @since 3.16.0
	ChangeAnnotationSupport Optional[ChangeAnnotationsSupportOptions] `json:"changeAnnotationSupport,omitzero"`
	// Whether the client supports `WorkspaceEditMetadata` in `WorkspaceEdit`s.
	// @since 3.18.0
	// @since 3.18.0
	MetadataSupport Optional[bool] `json:"metadataSupport,omitzero"`
	// Whether the client supports snippets as text edits.
	// @since 3.18.0
	// @since 3.18.0
	SnippetEditSupport Optional[bool] `json:"snippetEditSupport,omitzero"`
}

// Additional data about a workspace edit.
// @since 3.18.0
// @since 3.18.0
type WorkspaceEditMetadata struct {
	// Signal to the editor that this edit is a refactoring.
	IsRefactoring Optional[bool] `json:"isRefactoring,omitzero"`
}

// A workspace folder inside a client.
type WorkspaceFolder struct {
	// The associated URI for this workspace folder.
	URI string `json:"uri"`
	// The name of the workspace folder. Used to refer to this
	// workspace folder in the user interface.
	Name string `json:"name"`
}

// The workspace folder change event.
type WorkspaceFoldersChangeEvent struct {
	// The array of added workspace folders
	Added []WorkspaceFolder `json:"added"`
	// The array of the removed workspace folders
	Removed []WorkspaceFolder `json:"removed"`
}

type WorkspaceFoldersInitializeParams struct {
	// The workspace folders configured in the client when the server starts.
	// This property is only available if the client supports workspace folders.
	// It can be `null` if the client supports workspace folders but none are
	// configured.
	// @since 3.6.0
	// @since 3.6.0
	WorkspaceFolders OptionalNullable[OrWorkspaceFoldersInitializeParamsWorkspaceFolders] `json:"workspaceFolders,omitzero"`
}

type WorkspaceFoldersServerCapabilities struct {
	// The server has support for workspace folders
	Supported Optional[bool] `json:"supported,omitzero"`
	// Whether the server wants to receive workspace folder
	// change notifications.
	// If a string is provided the string is treated as an ID
	// under which the notification is registered on the client
	// side. The ID can be used to unregister for these events
	// using the `client/unregisterCapability` request.
	ChangeNotifications Optional[OrWorkspaceFoldersServerCapabilitiesChangeNotifications] `json:"changeNotifications,omitzero"`
}

// A full document diagnostic report for a workspace diagnostic result.
// @since 3.17.0
// @since 3.17.0
type WorkspaceFullDocumentDiagnosticReport struct {
	// A full document diagnostic report.
	Kind string `json:"kind"`
	// An optional result id. If provided it will
	// be sent on the next diagnostic request for the
	// same document.
	ResultID Optional[string] `json:"resultId,omitzero"`
	// The actual items.
	Items []Diagnostic `json:"items"`
	// The URI for which diagnostic information is reported.
	URI DocumentURI `json:"uri"`
	// The version number for which the diagnostics are reported.
	// If the document is not marked as open `null` can be provided.
	Version Nullable[OrWorkspaceFullDocumentDiagnosticReportVersion] `json:"version"`
}

// Defines workspace specific capabilities of the server.
// @since 3.18.0
// @since 3.18.0
type WorkspaceOptions struct {
	// The server supports workspace folder.
	// @since 3.6.0
	// @since 3.6.0
	WorkspaceFolders Optional[WorkspaceFoldersServerCapabilities] `json:"workspaceFolders,omitzero"`
	// The server is interested in notifications/requests for operations on files.
	// @since 3.16.0
	// @since 3.16.0
	FileOperations Optional[FileOperationOptions] `json:"fileOperations,omitzero"`
	// The server supports the `workspace/textDocumentContent` request.
	// @since 3.18.0
	// @since 3.18.0
	TextDocumentContent Optional[OrWorkspaceOptionsTextDocumentContent] `json:"textDocumentContent,omitzero"`
}

// A special workspace symbol that supports locations without a range.
// See also SymbolInformation.
// @since 3.17.0
// @since 3.17.0
type WorkspaceSymbol struct {
	// The name of this symbol.
	Name string `json:"name"`
	// The kind of this symbol.
	Kind SymbolKind `json:"kind"`
	// Tags for this symbol.
	// @since 3.16.0
	// @since 3.16.0
	Tags Optional[[]SymbolTag] `json:"tags,omitzero"`
	// The name of the symbol containing this symbol. This information is for
	// user interface purposes (e.g. to render a qualifier in the user interface
	// if necessary). It can't be used to re-infer a hierarchy for the document
	// symbols.
	ContainerName Optional[string] `json:"containerName,omitzero"`
	// The location of the symbol. Whether a server is allowed to
	// return a location without a range depends on the client
	// capability `workspace.symbol.resolveSupport`.
	// See SymbolInformation#location for more details.
	Location OrWorkspaceSymbolLocation `json:"location"`
	// A data entry field that is preserved on a workspace symbol between a
	// workspace symbol request and a workspace symbol resolve request.
	Data Optional[LSPAny] `json:"data,omitzero"`
}

// Client capabilities for a WorkspaceSymbolRequest.
type WorkspaceSymbolClientCapabilities struct {
	// Symbol request supports dynamic registration.
	DynamicRegistration Optional[bool] `json:"dynamicRegistration,omitzero"`
	// Specific capabilities for the `SymbolKind` in the `workspace/symbol` request.
	SymbolKind Optional[ClientSymbolKindOptions] `json:"symbolKind,omitzero"`
	// The client supports tags on `SymbolInformation`.
	// Clients supporting tags have to handle unknown tags gracefully.
	// @since 3.16.0
	// @since 3.16.0
	TagSupport Optional[ClientSymbolTagOptions] `json:"tagSupport,omitzero"`
	// The client support partial workspace symbols. The client will send the
	// request `workspaceSymbol/resolve` to the server to resolve additional
	// properties.
	// @since 3.17.0
	// @since 3.17.0
	ResolveSupport Optional[ClientSymbolResolveOptions] `json:"resolveSupport,omitzero"`
}

// Server capabilities for a WorkspaceSymbolRequest.
type WorkspaceSymbolOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The server provides support to resolve additional
	// information for a workspace symbol.
	// @since 3.17.0
	// @since 3.17.0
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// The parameters of a WorkspaceSymbolRequest.
type WorkspaceSymbolParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// An optional token that a server can use to report partial results (e.g. streaming) to
	// the client.
	PartialResultToken Optional[ProgressToken] `json:"partialResultToken,omitzero"`
	// A query string to filter symbols by. Clients may send an empty
	// string here to request all symbols.
	// The `query`-parameter should be interpreted in a *relaxed way* as editors
	// will apply their own highlighting and scoring on the results. A good rule
	// of thumb is to match case-insensitive and to simply check that the
	// characters of *query* appear in their order in a candidate symbol.
	// Servers shouldn't use prefix, substring, or similar strict matching.
	Query string `json:"query"`
}

// Registration options for a WorkspaceSymbolRequest.
type WorkspaceSymbolRegistrationOptions struct {
	WorkDoneProgress Optional[bool] `json:"workDoneProgress,omitzero"`
	// The server provides support to resolve additional
	// information for a workspace symbol.
	// @since 3.17.0
	// @since 3.17.0
	ResolveProvider Optional[bool] `json:"resolveProvider,omitzero"`
}

// An unchanged document diagnostic report for a workspace diagnostic result.
// @since 3.17.0
// @since 3.17.0
type WorkspaceUnchangedDocumentDiagnosticReport struct {
	// A document diagnostic report indicating
	// no changes to the last result. A server can
	// only return `unchanged` if result ids are
	// provided.
	Kind string `json:"kind"`
	// A result id which will be sent on the next
	// diagnostic request for the same document.
	ResultID string `json:"resultId"`
	// The URI for which diagnostic information is reported.
	URI DocumentURI `json:"uri"`
	// The version number for which the diagnostics are reported.
	// If the document is not marked as open `null` can be provided.
	Version Nullable[OrWorkspaceUnchangedDocumentDiagnosticReportVersion] `json:"version"`
}

// The initialize parameters
type _InitializeParams struct {
	// An optional token that a server can use to report work done progress.
	WorkDoneToken Optional[ProgressToken] `json:"workDoneToken,omitzero"`
	// The process Id of the parent process that started
	// the server.
	// Is `null` if the process has not been started by another process.
	// If the parent process is not alive then the server should exit.
	ProcessID Nullable[OrInitializeParamsProcessId] `json:"processId"`
	// Information about the client
	// @since 3.15.0
	// @since 3.15.0
	ClientInfo Optional[ClientInfo] `json:"clientInfo,omitzero"`
	// The locale the client is currently showing the user interface
	// in. This must not necessarily be the locale of the operating
	// system.
	// Uses IETF language tags as the value's syntax
	// (See https://en.wikipedia.org/wiki/IETF_language_tag)
	// @since 3.16.0
	// @since 3.16.0
	Locale Optional[string] `json:"locale,omitzero"`
	// The rootPath of the workspace. Is null
	// if no folder is open.
	// @deprecated in favour of rootUri.
	// Deprecated: in favour of rootUri.
	RootPath OptionalNullable[OrInitializeParamsRootPath] `json:"rootPath,omitzero"`
	// The rootUri of the workspace. Is null if no
	// folder is open. If both `rootPath` and `rootUri` are set
	// `rootUri` wins.
	// @deprecated in favour of workspaceFolders.
	// Deprecated: in favour of workspaceFolders.
	RootURI Nullable[OrInitializeParamsRootUri] `json:"rootUri"`
	// The capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities"`
	// User provided initialization options.
	InitializationOptions Optional[LSPAny] `json:"initializationOptions,omitzero"`
	// The initial trace setting. If omitted trace is disabled ('off').
	Trace Optional[TraceValue] `json:"trace,omitzero"`
}
