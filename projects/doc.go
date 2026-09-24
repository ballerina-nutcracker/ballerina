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

// Package projects provides the Ballerina Project API for loading, compiling,
// and managing Ballerina projects and packages.
//
// It implements the orchestration layer that loads projects from the filesystem,
// parses manifests, resolves dependencies, compiles modules, and generates BIR
// (Ballerina Intermediate Representation) for execution.
//
// # Type Hierarchy
//
// The core type hierarchy mirrors the Ballerina project model:
//
//	Project           Interface representing a Ballerina project (build, single-file, or bala)
//	  Package         A versioned package identified by org/name/version
//	    Module        A single module within a package (one default, zero or more named)
//	      Document    A single .bal source file within a module
//
// Each level has associated types:
//   - Identity:    PackageID, ModuleID, DocumentID (UUID-based)
//   - Descriptor:  PackageDescriptor, ModuleDescriptor (org + name + version)
//   - Config:      PackageConfig, ModuleConfig, DocumentConfig (filesystem layout)
//
// # Project Types
//
// Three concrete project implementations are provided:
//
//   - [BuildProject]: A standard project with a Ballerina.toml manifest
//   - [SingleFileProject]: A standalone .bal file without a manifest
//   - [BalaProject]: A pre-compiled package loaded from a .bala archive
//
// Projects are loaded using [Load] which auto-detects the
// project type based on the path:
//   - Directory with Ballerina.toml -> BuildProject
//   - Directory with package.json (bala) -> BalaProject
//   - Single .bal file -> SingleFileProject
//
// # Loading Projects
//
// Use [Load] to load projects from the filesystem. First, set up the required filesystems:
//
//	// Set up project filesystem (the directory containing the project)
//	fsys := os.DirFS("/path/to/project/parent")
//
//	// Set up Ballerina home filesystem (for langlib and cached packages)
//	// Determine home path from BAL_ENV env var or ~/.ballerina
//	ballerinaEnvPath := os.Getenv(projects.BallerinaEnvVar)
//	if ballerinaEnvPath == "" {
//	    userHome, _ := os.UserHomeDir()
//	    ballerinaEnvPath = filepath.Join(userHome, projects.UserHomeDirName)
//	}
//	ballerinaEnvFs := os.DirFS(ballerinaEnvPath)
//
//	// Load with default options
//	result, err := projects.Load(fsys, "myproject", projects.ProjectLoadConfig{
//	    BallerinaEnvFs: ballerinaEnvFs,
//	})
//
//	// Load with custom build options
//	buildOpts := projects.NewBuildOptionsBuilder().
//	    WithOffline(true).
//	    WithSkipTests(true).
//	    Build()
//	result, err := projects.Load(fsys, "./myproject", projects.ProjectLoadConfig{
//	    BallerinaEnvFs: ballerinaEnvFs,
//	    BuildOptions: &buildOpts,
//	})
//
//	// Load a single .bal file
//	result, err := projects.Load(fsys, "./main.bal", projects.ProjectLoadConfig{
//	    BallerinaEnvFs: ballerinaEnvFs,
//	})
//
// [ProjectLoadConfig] allows optional configuration:
//   - BuildOptions: Compilation settings (offline mode, skip tests, etc.)
//   - Repositories: Package repositories for dependency resolution
//   - BallerinaEnvFs: Filesystem for Ballerina home (defaults to <projectFs>/.ballerina/)
//
// # Compilation Pipeline
//
// The compilation pipeline follows a two-step design:
//
//  1. Load: Project loading creates PackageConfig from the filesystem
//  2. Compile: [PackageCompilation] parses, analyzes, type-checks, and (by default) generates
//     BIR for all modules. Parsing runs concurrently across the whole package; symbol and
//     top-level type resolution then run sequentially in topological order via
//     [PackageResolution] (each module needs its dependencies' published symbols first); the
//     remaining phases, including BIR generation, then run per module with no barrier between
//     them.
//
// BIR generation can be skipped via [CompilationOptionsBuilder.WithGenerateCode] for callers
// that only need diagnostics/symbols per compile (e.g. a language server). [BallerinaBackend],
// obtained via [NewBallerinaBackend], generates any missing BIR on demand.
//
// Complete example:
//
//	// Load project
//	result, err := projects.Load(fsys, "./myproject", projects.ProjectLoadConfig{
//	    BallerinaEnvFs: ballerinaEnvFs,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check for loading diagnostics
//	if result.Diagnostics().HasErrors() {
//	    // Handle errors
//	}
//
//	// Get project and package
//	project := result.Project()
//	pkg := project.CurrentPackage()
//
//	// Compile (parses, type-checks, analyzes, and generates BIR by default)
//	compilation := pkg.Compilation()
//	if compilation.DiagnosticResult().HasErrors() {
//	    // Handle compilation errors
//	}
//
//	// Access the generated BIR for execution
//	backend := projects.NewBallerinaBackend(compilation)
//	birPkg := backend.BIR()
//
// # Immutability and Modifiers
//
// Package, Module, and Document are immutable after creation. To create modified
// copies, use the modifier pattern:
//
//	// Modify a document
//	doc := module.Document(docID)
//	updatedDoc := doc.Modify().WithContent(newContent).Apply()
//
//	// Modify a package
//	modifier := pkg.Modify()
//	modifier.AddModule(moduleConfig)
//	newPkg := modifier.Apply()
//
// # Thread Safety
//
// Lazy-initialized fields (compilation, resolution, compiler backends) are
// protected by sync.Once or sync.Mutex for safe concurrent access. Document
// content supports both eager and lazy loading via the DocumentConfig interface.
package projects
