// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	stdcontext "context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/ast"
	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/cli/internal/nativeexec"
	compilercontext "github.com/ballerina-nutcracker/ballerina/context"
	"github.com/ballerina-nutcracker/ballerina/driver"
	"github.com/ballerina-nutcracker/ballerina/parser"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/projects"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/semantics"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

type runCompilation struct {
	context     *driver.Context
	resolver    *projects.DriverDependencyResolver
	birPackages []*bir.BIRPackage
	runtime     *runtime.Runtime
	loadFailed  bool
}

func compileRunInput(ctx stdcontext.Context, fsys fs.FS, input, packageDirName string,
	ballerinaEnvFS fs.FS, buildOptions projects.BuildOptions, parserDebug parser.DebugOptions,
	parserOutput io.Writer, nativeBaseDir, workspaceRoot string, platform pal.Platform,
) (*runCompilation, error) {
	resolver := projects.NewDriverDependencyResolver(fsys, projects.ProjectLoadConfig{
		BallerinaEnvFs: ballerinaEnvFS,
		BuildOptions:   &buildOptions,
	})
	if workspaceRoot != "" && input == "." {
		return &runCompilation{resolver: resolver},
			fmt.Errorf("cannot run a workspace project directly. Use 'bal run <package-path>' to run a specific package within the workspace")
	}
	dumps := &runDumpCollector{}
	var pipeline *driver.Pipeline
	hooks := driver.LifecycleHooks{
		OnParse: func(event driver.ParseEvent) bool {
			if event.Module.Package == event.Root && len(event.Document.ParserDebugOutput) > 0 {
				dumps.add(&dumps.parser, event.Module, event.SourcePath, string(event.Document.ParserDebugOutput))
			}
			return false
		},
		OnAST: func(event driver.ASTEvent) bool {
			if event.Module.Package != event.Root || (!event.Recovered && !buildOptions.DumpAST()) ||
				(event.Recovered && !buildOptions.DumpRecoveredAST()) {
				return false
			}
			printer := ast.PrettyPrinter{ShowNodeLocations: event.Recovered, DiagnosticEnv: pipeline.Context().DiagnosticEnv()}
			dumps.add(&dumps.ast, event.Module, event.SourcePath, printer.Print(event.Document.CompilationUnit)+"\n")
			return false
		},
		OnCFG: func(event driver.CFGEvent) bool {
			if !buildOptions.DumpCFG() || event.Module.Package != event.Root {
				return false
			}
			cfgContext := compilercontext.NewCompilerContext(resolver.CompilerEnvironment())
			rendered := semantics.PrettyPrintCFG(cfgContext, event.CFG)
			if buildOptions.DumpCFGFormat() == projects.CFGFormatDot {
				rendered = semantics.PrintCFGDot(cfgContext, event.CFG)
			}
			dumps.add(&dumps.cfg, event.Module, "", rendered)
			return false
		},
		OnBIR: func(event driver.BIREvent) bool {
			if !buildOptions.DumpBIR() || event.Package.PackageID.OrgName.Value() != event.Root.Org ||
				event.Package.PackageID.PkgName.Value() != event.Root.Name {
				return false
			}
			printer := bir.PrettyPrinter{}
			rendered := printer.Print(semtypes.ContextFrom(resolver.CompilerEnvironment().GetTypeEnv()), *event.Package)
			dumps.add(&dumps.bir, event.Module, "", rendered)
			return false
		},
	}
	if !buildOptions.DumpAST() && !buildOptions.DumpRecoveredAST() {
		hooks.OnAST = nil
	}
	pipeline = driver.NewPipeline(ctx, fsys, input, packageDirName, driver.NewEnv(resolver.CompilerEnvironment()),
		resolver, platform, parserDebug, hooks)
	rt, ok := pipeline.Run()
	dumps.flushParser(parserOutput)
	dumps.flush(os.Stderr)
	result := &runCompilation{
		context: pipeline.Context(), resolver: resolver, birPackages: pipeline.BIRPackages(), runtime: rt,
	}
	if result.context != nil && result.context.HasErrors() && pipeline.ParsedPackage() == nil {
		result.loadFailed = true
	}
	if !ok {
		if err := pipeline.Err(); err != nil {
			switch err.Error() {
			case "driver: cannot run a workspace directly":
				return result, fmt.Errorf("cannot run a workspace project directly. Use 'bal run <package-path>' to run a specific package within the workspace")
			case "driver: input is not a workspace member":
				return result, fmt.Errorf("no package found at path %s within workspace %s", nativeBaseDir, workspaceRoot)
			default:
				return result, err
			}
		}
		return result, nil
	}
	if !nativeexec.InNativeMode() {
		if err := execWithNativeProjects(resolver.NativeProjects(moduleDescriptors(pipeline.ParsedPackage())), nativeBaseDir); err != nil {
			return result, err
		}
	}
	return result, nil
}

type runDump struct {
	module driver.ModuleDescriptor
	path   string
	text   string
}

type runDumpCollector struct {
	mu     sync.Mutex
	parser []runDump
	ast    []runDump
	cfg    []runDump
	bir    []runDump
}

func (c *runDumpCollector) add(target *[]runDump, module driver.ModuleDescriptor, sourcePath, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	*target = append(*target, runDump{module: module, path: sourcePath, text: text})
}

func (c *runDumpCollector) flushParser(w io.Writer) {
	if w == nil {
		return
	}
	for _, item := range orderedRunDumps(c.parser) {
		_, _ = io.WriteString(w, item.text)
	}
}

func (c *runDumpCollector) flush(w io.Writer) {
	for _, item := range orderedRunDumps(c.ast) {
		_, _ = io.WriteString(w, item.text)
	}
	for _, item := range orderedRunDumps(c.cfg) {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "==================BEGIN CFG==================")
		_, _ = fmt.Fprintln(w, strings.TrimSpace(item.text))
		_, _ = fmt.Fprintln(w, "===================END CFG===================")
	}
	for _, item := range orderedRunDumps(c.bir) {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "==================BEGIN BIR==================")
		_, _ = fmt.Fprintln(w, strings.TrimSpace(item.text))
		_, _ = fmt.Fprintln(w, "===================END BIR===================")
	}
}

func orderedRunDumps(values []runDump) []runDump {
	result := append([]runDump(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].module.Name != result[j].module.Name {
			return result[i].module.Name < result[j].module.Name
		}
		return result[i].path < result[j].path
	})
	return result
}

func moduleDescriptors(parsed *driver.ParsedPackage) []driver.ModuleDescriptor {
	result := make([]driver.ModuleDescriptor, len(parsed.Modules))
	for index, module := range parsed.Modules {
		result[index] = module.ID
	}
	return result
}
