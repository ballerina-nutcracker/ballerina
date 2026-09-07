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

package driver

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"sync"

	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/st"
	"github.com/ballerina-nutcracker/ballerina/tools/text"
)

// ParsedDocument holds the syntax tree and text of a single parsed source file,
// together with any parser debug output captured for it.
type ParsedDocument struct {
	SourcePath        string
	TextDocument      text.TextDocument
	SyntaxTree        *st.SyntaxTree
	ParserDebugOutput []byte
}

// ParsedModule holds the parsed regular and test documents of one module,
// indexed positionally as they appeared in its ModuleSources.
type ParsedModule struct {
	ID            ModuleDescriptor
	Documents     []*ParsedDocument
	TestDocuments []*ParsedDocument
	dependency    bool
	implicitName  string
}

type packageData struct {
	manifest Manifest
}

// ParsedPackage holds the parsed modules of the root package and of every
// dependency package pulled in later by dependency resolution.
type ParsedPackage struct {
	Root     PackageDescriptor
	Modules  []*ParsedModule
	packages map[PackageDescriptor]packageData
}

// Parse parses every document of the root package described by sources, rooted at
// packageRoot within fsys, and records the root descriptor and discovered modules on ctx.
// It returns an error if ctx is already cancelled, if the inputs fail validation, or if any
// document cannot be read or parsed; syntax diagnostics are collected on ctx instead.
func Parse(ctx *Context, fsys fs.FS, packageRoot string, sources *PackageSources,
	debug parser.DebugOptions,
) (*ParsedPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateParseInputs(packageRoot, sources); err != nil {
		return nil, err
	}
	ctx.setRoot(sources.Descriptor)
	parsed, err := parsePackage(ctx, fsys, packageRoot, sources, debug, false)
	if err != nil {
		return nil, err
	}
	parsed.packages = map[PackageDescriptor]packageData{sources.Descriptor: {manifest: cloneManifest(sources.Manifest)}}
	return parsed, nil
}

// parsePackage parses all modules of one package, marking them as dependency modules when
// dependency is true, and publishes the resulting module identifiers to ctx as the discovery set.
// It returns the first module error, or the cancellation error if ctx was cancelled while parsing.
func parsePackage(ctx *Context, fsys fs.FS, packageRoot string, sources *PackageSources,
	debug parser.DebugOptions, dependency bool,
) (*ParsedPackage, error) {
	modules := make([]*ParsedModule, len(sources.Modules))
	errs := make([]error, len(sources.Modules))
	schedulingErr := runModuleParsers(sources.Modules, ctx.Err, func(index int, module *ModuleSources) {
		modules[index], errs[index] = parseModule(ctx, fsys, packageRoot, module, debug, dependency)
	})
	if schedulingErr != nil {
		return nil, schedulingErr
	}
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ids := make([]ModuleDescriptor, len(modules))
	for i, module := range modules {
		ids[i] = module.ID
	}
	ctx.setDiscoveryModules(ids)
	return &ParsedPackage{Root: sources.Descriptor, Modules: modules}, nil
}

// runModuleParsers runs parse for each module concurrently, one goroutine per module, and waits
// for all of them. Cancellation is observed only between scheduling steps: schedulingCheck is
// consulted before each goroutine is started and its first error stops further scheduling and is
// returned, while goroutines already started still run to completion.
func runModuleParsers(modules []*ModuleSources, schedulingCheck func() error,
	parse func(int, *ModuleSources),
) error {
	var wg sync.WaitGroup
	var schedulingErr error
	for index, module := range modules {
		if err := schedulingCheck(); err != nil {
			schedulingErr = err
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			parse(index, module)
		}()
	}
	wg.Wait()
	return schedulingErr
}

// parseModule reads and parses the module's regular and test documents concurrently, one
// goroutine per document, storing each result in the slot matching its source index. Each
// goroutine checks ctx for cancellation before reading and before parsing, and drains the
// per-document compiler context into ctx. Debug output is suppressed for dependency modules.
// The first error from any document is returned.
func parseModule(ctx *Context, fsys fs.FS, root string, sources *ModuleSources,
	debug parser.DebugOptions, dependency bool,
) (*ParsedModule, error) {
	result := &ParsedModule{ID: sources.ID, Documents: make([]*ParsedDocument, len(sources.Documents)),
		TestDocuments: make([]*ParsedDocument, len(sources.TestDocuments)), dependency: dependency}
	total := len(sources.Documents) + len(sources.TestDocuments)
	errs := make([]error, total)
	var wg sync.WaitGroup
	parseOne := func(test bool, index int, sourcePath string, slot **ParsedDocument, errorIndex int) {
		defer wg.Done()
		if err := ctx.Err(); err != nil {
			errs[errorIndex] = err
			return
		}
		content, err := fs.ReadFile(fsys, path.Join(root, sourcePath))
		if err != nil {
			errs[errorIndex] = err
			return
		}
		if err := ctx.Err(); err != nil {
			errs[errorIndex] = err
			return
		}
		cx := ctx.newCompilerContext(sources.ID)
		cx.StartStage(compilercontext.StageParse)
		var buffer bytes.Buffer
		debugWriter := io.Writer(&buffer)
		debugOptions := debug
		if dependency {
			debugWriter = io.Discard
			debugOptions = parser.DebugOptions{}
		}
		identity := diagnosticIdentity(dependency, sources.ID, sourcePath)
		tree, err := parser.GetSyntaxTreeWithIdentity(cx, sourcePath, identity, string(content), debugOptions, debugWriter)
		cx.EndStage()
		ctx.drainModule(cx, sources.ID, compilercontext.StageParse, test, index)
		if err != nil {
			errs[errorIndex] = err
			return
		}
		*slot = &ParsedDocument{SourcePath: sourcePath, TextDocument: tree.TextDocument(), SyntaxTree: tree,
			ParserDebugOutput: append([]byte(nil), buffer.Bytes()...)}
	}
	for index, sourcePath := range sources.Documents {
		wg.Add(1)
		go parseOne(false, index, sourcePath, &result.Documents[index], index)
	}
	for index, sourcePath := range sources.TestDocuments {
		wg.Add(1)
		go parseOne(true, index, sourcePath, &result.TestDocuments[index], len(sources.Documents)+index)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// validateParseInputs rejects an unusable package root or sources: a root that is not a valid
// relative path, nil or incompletely described sources, modules whose package descriptor does
// not match, duplicate or wrongly prefixed module names, duplicate or escaping source paths,
// and any module set that does not contain exactly one default module.
func validateParseInputs(root string, sources *PackageSources) error {
	if root == "" || (root != "." && !fs.ValidPath(root)) || strings.HasPrefix(root, "/") || containsParent(root) {
		return fmt.Errorf("driver: invalid package root %q", root)
	}
	if sources == nil {
		return errors.New("driver: nil package sources")
	}
	if sources.Descriptor.Org == "" || sources.Descriptor.Name == "" || sources.Descriptor.Version == "" || !validVersion(sources.Descriptor.Version) {
		return fmt.Errorf("driver: invalid package descriptor %+v", sources.Descriptor)
	}
	seenModules := make(map[string]bool)
	seenDocuments := make(map[string]bool)
	defaultCount := 0
	for _, module := range sources.Modules {
		if module == nil {
			return errors.New("driver: nil module sources")
		}
		if module.ID.Package != sources.Descriptor {
			return fmt.Errorf("driver: module %q has mismatched package descriptor", module.ID.Name)
		}
		if seenModules[module.ID.Name] {
			return fmt.Errorf("driver: duplicate module %q", module.ID.Name)
		}
		seenModules[module.ID.Name] = true
		if module.ID.Name == sources.Descriptor.Name {
			defaultCount++
		} else if !strings.HasPrefix(module.ID.Name, sources.Descriptor.Name+".") || module.ID.Name == sources.Descriptor.Name+"." {
			return fmt.Errorf("driver: invalid named module %q", module.ID.Name)
		}
		for _, document := range append(append([]string(nil), module.Documents...), module.TestDocuments...) {
			if document == "" || !fs.ValidPath(document) || strings.HasPrefix(document, "/") || containsParent(document) {
				return fmt.Errorf("driver: invalid source path %q", document)
			}
			joined := path.Join(root, document)
			if root != "." && joined != root && !strings.HasPrefix(joined, root+"/") {
				return fmt.Errorf("driver: source path %q escapes root %q", document, root)
			}
			if seenDocuments[document] {
				return fmt.Errorf("driver: duplicate source path %q", document)
			}
			seenDocuments[document] = true
		}
	}
	if defaultCount != 1 {
		return fmt.Errorf("driver: expected exactly one default module, got %d", defaultCount)
	}
	return nil
}

// containsParent reports whether any slash-separated component of value is "..".
func containsParent(value string) bool {
	for _, component := range strings.Split(value, "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

// validVersion reports whether version is a one-to-three-part dotted sequence of non-negative
// integers, optionally followed by a non-empty pre-release or build-metadata suffix.
func validVersion(version string) bool {
	if version == "" {
		return false
	}
	core := version
	if index := strings.IndexByte(core, '+'); index >= 0 {
		if index == len(core)-1 {
			return false
		}
		core = core[:index]
	}
	if index := strings.IndexByte(core, '-'); index >= 0 {
		if index == len(core)-1 {
			return false
		}
		core = core[:index]
	}
	parts := strings.Split(core, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return false
		}
	}
	return true
}

// diagnosticIdentity builds the stable key used to attribute diagnostics to a document: an "R"
// or "D" namespace byte for root versus dependency, followed by the org, package name, version,
// module name and source path each length-prefixed. Unlike the display path, this encoding is
// unambiguous, so two documents in different packages or modules can never collide.
func diagnosticIdentity(dependency bool, module ModuleDescriptor, sourcePath string) string {
	namespace := "R"
	if dependency {
		namespace = "D"
	}
	values := []string{module.Package.Org, module.Package.Name, module.Package.Version, module.Name, sourcePath}
	var builder strings.Builder
	builder.WriteString(namespace)
	for _, value := range values {
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
	}
	return builder.String()
}

// cloneManifest returns a copy of manifest whose Dependencies slice is freshly allocated, so the
// caller cannot mutate the original's backing array.
func cloneManifest(manifest Manifest) Manifest {
	manifest.Dependencies = append([]Dependency(nil), manifest.Dependencies...)
	return manifest
}
