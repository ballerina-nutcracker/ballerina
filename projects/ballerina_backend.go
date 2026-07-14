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

import (
	"ballerina-lang-go/bir"
	"ballerina-lang-go/model"
	"ballerina-lang-go/semantics"
)

// BallerinaBackendTarget is the target platform for the Go/BIR backend.
const BallerinaBackendTarget TargetPlatform = "native"

// BallerinaBackend generates BIR from compiled modules.
type BallerinaBackend struct {
	packageCompilation *PackageCompilation
	packageContext     *packageContext
}

// NewBallerinaBackend creates a BallerinaBackend from a PackageCompilation
// and triggers BIR generation for all compiled modules.
func NewBallerinaBackend(compilation *PackageCompilation) *BallerinaBackend {
	backend := compilation.getCompilerBackend(BallerinaBackendTarget, func(tp TargetPlatform) CompilerBackend {
		b := &BallerinaBackend{
			packageCompilation: compilation,
			packageContext:     compilation.getPackageContext(),
		}
		b.performCodeGen()
		return b
	})
	return backend.(*BallerinaBackend)
}

// performCodeGen generates BIR for all modules in topological order.
func (b *BallerinaBackend) performCodeGen() {
	for _, moduleCtx := range b.packageCompilation.Resolution().topologicallySortedModuleList {
		if moduleCtx.getCompilationState() == moduleCompilationStateCompiled {
			generateCodeInternal(moduleCtx)
		}
	}
}

// TargetPlatform returns the target platform for this backend.
func (b *BallerinaBackend) TargetPlatform() TargetPlatform {
	return BallerinaBackendTarget
}

// BIR returns the BIR package for the default module of the root package.
// BIR is generated during performCodeGen() which runs when the backend is created.
func (b *BallerinaBackend) BIR() *bir.BIRPackage {
	return b.packageContext.getDefaultModuleContext().getBIRPackage()
}

// ExportedSymbols returns the exported symbol spaces for all compiled modules.
func (b *BallerinaBackend) ExportedSymbols() map[semantics.PackageIdentifier]model.ExportedSymbolSpace {
	return b.packageCompilation.rootPackageContext.getProject().Environment().publicSymbols
}

// BIRPackages returns all BIR packages in topological order.
func (b *BallerinaBackend) BIRPackages() []*bir.BIRPackage {
	var pkgs []*bir.BIRPackage
	for _, moduleCtx := range b.packageCompilation.Resolution().topologicallySortedModuleList {
		if birPkg := moduleCtx.getBIRPackage(); birPkg != nil {
			pkgs = append(pkgs, birPkg)
		}
	}
	return pkgs
}

// EmitBala writes the package's `.bala` archive into outputDir and returns
// the archive's path. The output filename follows the
// <org>-<name>-<platform>-<version>.bala convention.
//
// The archive layout mirrors a build project on disk plus a Bala.toml:
//
//	Bala.toml                  // bala schema version + build metadata + modules list
//	Ballerina.toml
//	Dependencies.toml
//	<file>.bal                 // default-module sources at the root
//	modules/<sub>/<file>.bal   // sub-module sources, one folder per sub-module
//
// docs/ and resources/ are deferred until a consumer needs them. outputDir
// is created if it does not exist; an existing archive at the computed path
// is overwritten.
func (b *BallerinaBackend) EmitBala(outputDir string) (string, error) {
	pkg := b.packageContext.getProject().CurrentPackage()
	return writeBala(pkg, b.packageCompilation.Resolution(), outputDir)
}
