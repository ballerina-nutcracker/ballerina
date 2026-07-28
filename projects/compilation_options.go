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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package projects

// optionalBool is a tri-state boolean: unknown (unset), true, or false.
// It uses a single byte instead of *bool (pointer + bool = 9+ bytes with heap allocation).
type optionalBool byte

const (
	// optionalBoolUnknown is the zero value representing an unset boolean.
	optionalBoolUnknown optionalBool = iota
	// optionalBoolTrue represents an explicitly set true value.
	optionalBoolTrue
	// optionalBoolFalse represents an explicitly set false value.
	optionalBoolFalse
)

// optionalBoolOf converts a bool to an optionalBool.
func optionalBoolOf(value bool) optionalBool {
	if value {
		return optionalBoolTrue
	}
	return optionalBoolFalse
}

// isSet returns true if this optional has been explicitly set (true or false).
func (o optionalBool) isSet() bool {
	return o != optionalBoolUnknown
}

// valueOr returns the bool value, or the given default if unset.
func (o optionalBool) valueOr(def bool) bool {
	switch o {
	case optionalBoolTrue:
		return true
	case optionalBoolFalse:
		return false
	default:
		return def
	}
}

// PackageLockingMode represents the locking mode for package dependencies.
type PackageLockingMode int

const (
	// PackageLockingModeUnknown represents an unset locking mode.
	PackageLockingModeUnknown PackageLockingMode = iota
	// PackageLockingModeSoft allows minor and patch version updates.
	PackageLockingModeSoft
	// PackageLockingModeMedium allows only patch version updates.
	PackageLockingModeMedium
	// PackageLockingModeHard locks to exact versions.
	PackageLockingModeHard
)

// String returns the string representation of PackageLockingMode.
func (m PackageLockingMode) String() string {
	switch m {
	case PackageLockingModeSoft:
		return "soft"
	case PackageLockingModeMedium:
		return "medium"
	case PackageLockingModeHard:
		return "hard"
	default:
		return "unknown"
	}
}

// CFGFormat represents the output format for CFG (Control Flow Graph) dumps.
type CFGFormat int

const (
	// CFGFormatUnknown represents an unset format (zero value).
	// Callers should treat this as "use default" which is S-expression.
	CFGFormatUnknown CFGFormat = iota
	// CFGFormatSexp uses S-expression format (the default).
	CFGFormatSexp
	// CFGFormatDot uses Graphviz DOT format.
	CFGFormatDot
)

// String returns the string representation of CFGFormat.
func (f CFGFormat) String() string {
	switch f {
	case CFGFormatSexp:
		return "sexp"
	case CFGFormatDot:
		return "dot"
	default:
		return ""
	}
}

// ParseCFGFormat parses a string into a CFGFormat value.
func ParseCFGFormat(s string) CFGFormat {
	switch s {
	case "dot":
		return CFGFormatDot
	case "sexp", "":
		return CFGFormatSexp
	default:
		return CFGFormatUnknown
	}
}

// CompilationOptions holds compilation-specific options.
// BuildOptions contains a CompilationOptions instance and delegates compilation-related methods to it.
type CompilationOptions struct {
	offlineBuild                  optionalBool
	experimental                  optionalBool
	observabilityIncluded         optionalBool
	dumpAST                       optionalBool
	dumpRecoveredAST              optionalBool
	dumpBIR                       optionalBool
	dumpBIRFile                   optionalBool
	dumpCFG                       optionalBool
	dumpGraph                     optionalBool
	dumpRawGraphs                 optionalBool
	dumpTokens                    optionalBool
	dumpST                        optionalBool
	listConflictedClasses         optionalBool
	sticky                        optionalBool
	withCodeGenerators            optionalBool
	withCodeModifiers             optionalBool
	configSchemaGen               optionalBool
	exportOpenAPI                 optionalBool
	exportComponentModel          optionalBool
	disableSyntaxTree             optionalBool
	remoteManagement              optionalBool
	optimizeDependencyCompilation optionalBool
	traceRecovery                 optionalBool
	stats                         optionalBool
	cloud                         *string
	dumpCFGFormat                 CFGFormat
	lockingMode                   PackageLockingMode
}

// NewCompilationOptions creates a new CompilationOptions with default values.
func NewCompilationOptions() CompilationOptions {
	return CompilationOptions{}
}

// OfflineBuild returns whether offline build mode is enabled.
func (c CompilationOptions) OfflineBuild() bool {
	return c.offlineBuild.valueOr(false)
}

// Experimental returns whether experimental features are enabled.
func (c CompilationOptions) Experimental() bool {
	return c.experimental.valueOr(false)
}

// ObservabilityIncluded returns whether observability is included.
func (c CompilationOptions) ObservabilityIncluded() bool {
	return c.observabilityIncluded.valueOr(false)
}

// DumpAST returns whether AST dumping is enabled.
func (c CompilationOptions) DumpAST() bool {
	return c.dumpAST.valueOr(false)
}

// DumpRecoveredAST returns whether recovered AST dumping is enabled.
func (c CompilationOptions) DumpRecoveredAST() bool {
	return c.dumpRecoveredAST.valueOr(false)
}

// DumpBIR returns whether BIR dumping is enabled.
func (c CompilationOptions) DumpBIR() bool {
	return c.dumpBIR.valueOr(false)
}

// DumpBIRFile returns whether BIR file dumping is enabled.
func (c CompilationOptions) DumpBIRFile() bool {
	return c.dumpBIRFile.valueOr(false)
}

// DumpCFG returns whether CFG dumping is enabled.
func (c CompilationOptions) DumpCFG() bool {
	return c.dumpCFG.valueOr(false)
}

// DumpCFGFormat returns the CFG dump format.
// Returns CFGFormatUnknown if not set (callers should treat as S-expression default).
func (c CompilationOptions) DumpCFGFormat() CFGFormat {
	return c.dumpCFGFormat
}

// DumpGraph returns whether graph dumping is enabled.
func (c CompilationOptions) DumpGraph() bool {
	return c.dumpGraph.valueOr(false)
}

// DumpRawGraphs returns whether raw graph dumping is enabled.
func (c CompilationOptions) DumpRawGraphs() bool {
	return c.dumpRawGraphs.valueOr(false)
}

// Cloud returns the cloud target.
// Returns empty string if not set or explicitly cleared.
func (c CompilationOptions) Cloud() string {
	if c.cloud == nil {
		return ""
	}
	return *c.cloud
}

// ListConflictedClasses returns whether conflicted classes should be listed.
func (c CompilationOptions) ListConflictedClasses() bool {
	return c.listConflictedClasses.valueOr(false)
}

// Sticky returns whether sticky mode is enabled.
// Deprecated: Use LockingMode() instead.
func (c CompilationOptions) Sticky() bool {
	return c.sticky.valueOr(false)
}

// WithCodeGenerators returns whether code generators should be run.
func (c CompilationOptions) WithCodeGenerators() bool {
	return c.withCodeGenerators.valueOr(false)
}

// WithCodeModifiers returns whether code modifiers should be run.
func (c CompilationOptions) WithCodeModifiers() bool {
	return c.withCodeModifiers.valueOr(false)
}

// ConfigSchemaGen returns whether config schema generation is enabled.
func (c CompilationOptions) ConfigSchemaGen() bool {
	return c.configSchemaGen.valueOr(false)
}

// ExportOpenAPI returns whether OpenAPI export is enabled.
func (c CompilationOptions) ExportOpenAPI() bool {
	return c.exportOpenAPI.valueOr(false)
}

// ExportComponentModel returns whether component model export is enabled.
func (c CompilationOptions) ExportComponentModel() bool {
	return c.exportComponentModel.valueOr(false)
}

// DisableSyntaxTree returns whether syntax tree caching is disabled.
func (c CompilationOptions) DisableSyntaxTree() bool {
	return c.disableSyntaxTree.valueOr(false)
}

// RemoteManagement returns whether remote management is enabled.
func (c CompilationOptions) RemoteManagement() bool {
	return c.remoteManagement.valueOr(false)
}

// OptimizeDependencyCompilation returns whether dependency compilation optimization is enabled.
func (c CompilationOptions) OptimizeDependencyCompilation() bool {
	return c.optimizeDependencyCompilation.valueOr(false)
}

// DumpTokens returns whether lexer token dumping is enabled.
func (c CompilationOptions) DumpTokens() bool {
	return c.dumpTokens.valueOr(false)
}

// DumpST returns whether syntax tree dumping is enabled.
func (c CompilationOptions) DumpST() bool {
	return c.dumpST.valueOr(false)
}

// TraceRecovery returns whether error recovery tracing is enabled.
func (c CompilationOptions) TraceRecovery() bool {
	return c.traceRecovery.valueOr(false)
}

// Stats returns whether compilation stats collection is enabled.
func (c CompilationOptions) Stats() bool {
	return c.stats.valueOr(false)
}

// LockingMode returns the package locking mode.
// Returns PackageLockingModeUnknown if not explicitly set.
func (c CompilationOptions) LockingMode() PackageLockingMode {
	return c.lockingMode
}

// acceptOptionalBool returns theirs if set, else ours.
func acceptOptionalBool(ours, theirs optionalBool) optionalBool {
	if theirs.isSet() {
		return theirs
	}
	return ours
}

// AcceptTheirs merges the given compilation options by favoring theirs if there are conflicts.
func (c CompilationOptions) AcceptTheirs(theirs CompilationOptions) CompilationOptions {
	merged := CompilationOptions{
		offlineBuild:                  acceptOptionalBool(c.offlineBuild, theirs.offlineBuild),
		experimental:                  acceptOptionalBool(c.experimental, theirs.experimental),
		observabilityIncluded:         acceptOptionalBool(c.observabilityIncluded, theirs.observabilityIncluded),
		dumpAST:                       acceptOptionalBool(c.dumpAST, theirs.dumpAST),
		dumpRecoveredAST:              acceptOptionalBool(c.dumpRecoveredAST, theirs.dumpRecoveredAST),
		dumpBIR:                       acceptOptionalBool(c.dumpBIR, theirs.dumpBIR),
		dumpBIRFile:                   acceptOptionalBool(c.dumpBIRFile, theirs.dumpBIRFile),
		dumpCFG:                       acceptOptionalBool(c.dumpCFG, theirs.dumpCFG),
		dumpGraph:                     acceptOptionalBool(c.dumpGraph, theirs.dumpGraph),
		dumpRawGraphs:                 acceptOptionalBool(c.dumpRawGraphs, theirs.dumpRawGraphs),
		dumpTokens:                    acceptOptionalBool(c.dumpTokens, theirs.dumpTokens),
		dumpST:                        acceptOptionalBool(c.dumpST, theirs.dumpST),
		listConflictedClasses:         acceptOptionalBool(c.listConflictedClasses, theirs.listConflictedClasses),
		sticky:                        acceptOptionalBool(c.sticky, theirs.sticky),
		withCodeGenerators:            acceptOptionalBool(c.withCodeGenerators, theirs.withCodeGenerators),
		withCodeModifiers:             acceptOptionalBool(c.withCodeModifiers, theirs.withCodeModifiers),
		configSchemaGen:               acceptOptionalBool(c.configSchemaGen, theirs.configSchemaGen),
		exportOpenAPI:                 acceptOptionalBool(c.exportOpenAPI, theirs.exportOpenAPI),
		exportComponentModel:          acceptOptionalBool(c.exportComponentModel, theirs.exportComponentModel),
		disableSyntaxTree:             acceptOptionalBool(c.disableSyntaxTree, theirs.disableSyntaxTree),
		remoteManagement:              acceptOptionalBool(c.remoteManagement, theirs.remoteManagement),
		optimizeDependencyCompilation: acceptOptionalBool(c.optimizeDependencyCompilation, theirs.optimizeDependencyCompilation),
		traceRecovery:                 acceptOptionalBool(c.traceRecovery, theirs.traceRecovery),
		stats:                         acceptOptionalBool(c.stats, theirs.stats),
	}

	// Cloud (*string)
	if theirs.cloud != nil {
		merged.cloud = theirs.cloud
	} else {
		merged.cloud = c.cloud
	}

	// CFG format
	if theirs.dumpCFGFormat != CFGFormatUnknown {
		merged.dumpCFGFormat = theirs.dumpCFGFormat
	} else {
		merged.dumpCFGFormat = c.dumpCFGFormat
	}

	// Locking mode: explicit settings take precedence over deprecated sticky flag
	if theirs.lockingMode != PackageLockingModeUnknown {
		merged.lockingMode = theirs.lockingMode
	} else if c.lockingMode != PackageLockingModeUnknown {
		merged.lockingMode = c.lockingMode
	} else if merged.sticky == optionalBoolTrue {
		// Only use sticky as fallback when no explicit lockingMode is set
		merged.lockingMode = PackageLockingModeHard
	}

	return merged
}

// CompilationOptionsBuilder provides a builder pattern for CompilationOptions.
type CompilationOptionsBuilder struct {
	options CompilationOptions
}

// NewCompilationOptionsBuilder creates a new CompilationOptionsBuilder.
func NewCompilationOptionsBuilder() *CompilationOptionsBuilder {
	return &CompilationOptionsBuilder{
		options: NewCompilationOptions(),
	}
}

// WithOfflineBuild sets offline build mode.
func (b *CompilationOptionsBuilder) WithOfflineBuild(value bool) *CompilationOptionsBuilder {
	b.options.offlineBuild = optionalBoolOf(value)
	return b
}

// WithExperimental sets experimental features flag.
func (b *CompilationOptionsBuilder) WithExperimental(value bool) *CompilationOptionsBuilder {
	b.options.experimental = optionalBoolOf(value)
	return b
}

// WithObservabilityIncluded sets observability inclusion.
func (b *CompilationOptionsBuilder) WithObservabilityIncluded(value bool) *CompilationOptionsBuilder {
	b.options.observabilityIncluded = optionalBoolOf(value)
	return b
}

// WithDumpAST sets AST dumping flag.
func (b *CompilationOptionsBuilder) WithDumpAST(value bool) *CompilationOptionsBuilder {
	b.options.dumpAST = optionalBoolOf(value)
	return b
}

// WithDumpRecoveredAST sets recovered AST dumping flag.
func (b *CompilationOptionsBuilder) WithDumpRecoveredAST(value bool) *CompilationOptionsBuilder {
	b.options.dumpRecoveredAST = optionalBoolOf(value)
	return b
}

// WithDumpBIR sets BIR dumping flag.
func (b *CompilationOptionsBuilder) WithDumpBIR(value bool) *CompilationOptionsBuilder {
	b.options.dumpBIR = optionalBoolOf(value)
	return b
}

// WithDumpBIRFile sets BIR file dumping flag.
func (b *CompilationOptionsBuilder) WithDumpBIRFile(value bool) *CompilationOptionsBuilder {
	b.options.dumpBIRFile = optionalBoolOf(value)
	return b
}

// WithDumpCFG sets CFG dumping flag.
func (b *CompilationOptionsBuilder) WithDumpCFG(value bool) *CompilationOptionsBuilder {
	b.options.dumpCFG = optionalBoolOf(value)
	return b
}

// WithDumpCFGFormat sets CFG dump format.
func (b *CompilationOptionsBuilder) WithDumpCFGFormat(value CFGFormat) *CompilationOptionsBuilder {
	b.options.dumpCFGFormat = value
	return b
}

// WithDumpGraph sets graph dumping flag.
func (b *CompilationOptionsBuilder) WithDumpGraph(value bool) *CompilationOptionsBuilder {
	b.options.dumpGraph = optionalBoolOf(value)
	return b
}

// WithDumpRawGraphs sets raw graph dumping flag.
func (b *CompilationOptionsBuilder) WithDumpRawGraphs(value bool) *CompilationOptionsBuilder {
	b.options.dumpRawGraphs = optionalBoolOf(value)
	return b
}

// WithCloud sets the cloud target.
func (b *CompilationOptionsBuilder) WithCloud(value string) *CompilationOptionsBuilder {
	b.options.cloud = &value
	return b
}

// WithListConflictedClasses sets conflicted classes listing flag.
func (b *CompilationOptionsBuilder) WithListConflictedClasses(value bool) *CompilationOptionsBuilder {
	b.options.listConflictedClasses = optionalBoolOf(value)
	return b
}

// WithSticky sets sticky mode.
// Deprecated: Use WithLockingMode() instead.
func (b *CompilationOptionsBuilder) WithSticky(value bool) *CompilationOptionsBuilder {
	b.options.sticky = optionalBoolOf(value)
	return b
}

// WithCodeGeneratorsEnabled sets whether code generators should run.
func (b *CompilationOptionsBuilder) WithCodeGeneratorsEnabled(value bool) *CompilationOptionsBuilder {
	b.options.withCodeGenerators = optionalBoolOf(value)
	return b
}

// WithCodeModifiersEnabled sets whether code modifiers should run.
func (b *CompilationOptionsBuilder) WithCodeModifiersEnabled(value bool) *CompilationOptionsBuilder {
	b.options.withCodeModifiers = optionalBoolOf(value)
	return b
}

// WithConfigSchemaGen sets config schema generation flag.
func (b *CompilationOptionsBuilder) WithConfigSchemaGen(value bool) *CompilationOptionsBuilder {
	b.options.configSchemaGen = optionalBoolOf(value)
	return b
}

// WithExportOpenAPI sets OpenAPI export flag.
func (b *CompilationOptionsBuilder) WithExportOpenAPI(value bool) *CompilationOptionsBuilder {
	b.options.exportOpenAPI = optionalBoolOf(value)
	return b
}

// WithExportComponentModel sets component model export flag.
func (b *CompilationOptionsBuilder) WithExportComponentModel(value bool) *CompilationOptionsBuilder {
	b.options.exportComponentModel = optionalBoolOf(value)
	return b
}

// WithDisableSyntaxTree sets syntax tree caching disabled flag.
func (b *CompilationOptionsBuilder) WithDisableSyntaxTree(value bool) *CompilationOptionsBuilder {
	b.options.disableSyntaxTree = optionalBoolOf(value)
	return b
}

// WithRemoteManagement sets remote management flag.
func (b *CompilationOptionsBuilder) WithRemoteManagement(value bool) *CompilationOptionsBuilder {
	b.options.remoteManagement = optionalBoolOf(value)
	return b
}

// WithOptimizeDependencyCompilation sets dependency compilation optimization flag.
func (b *CompilationOptionsBuilder) WithOptimizeDependencyCompilation(value bool) *CompilationOptionsBuilder {
	b.options.optimizeDependencyCompilation = optionalBoolOf(value)
	return b
}

// WithDumpTokens sets lexer token dumping flag.
func (b *CompilationOptionsBuilder) WithDumpTokens(value bool) *CompilationOptionsBuilder {
	b.options.dumpTokens = optionalBoolOf(value)
	return b
}

// WithDumpST sets syntax tree dumping flag.
func (b *CompilationOptionsBuilder) WithDumpST(value bool) *CompilationOptionsBuilder {
	b.options.dumpST = optionalBoolOf(value)
	return b
}

// WithTraceRecovery sets error recovery tracing flag.
func (b *CompilationOptionsBuilder) WithTraceRecovery(value bool) *CompilationOptionsBuilder {
	b.options.traceRecovery = optionalBoolOf(value)
	return b
}

// WithStats sets compilation stats collection flag.
func (b *CompilationOptionsBuilder) WithStats(value bool) *CompilationOptionsBuilder {
	b.options.stats = optionalBoolOf(value)
	return b
}

// WithLockingMode sets the package locking mode.
func (b *CompilationOptionsBuilder) WithLockingMode(mode PackageLockingMode) *CompilationOptionsBuilder {
	b.options.lockingMode = mode
	return b
}

// Build creates the CompilationOptions instance.
func (b *CompilationOptionsBuilder) Build() CompilationOptions {
	return b.options
}
