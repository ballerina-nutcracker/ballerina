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
	"io/fs"

	"ballerina/context"
	"ballerina/lib/langlibs"
	"ballerina/semtypes"
)

type ProjectEnvironmentBuilder struct {
	fsys         fs.FS
	repositories []Repository
	buildOptions BuildOptions
}

func NewProjectEnvironmentBuilder(fsys fs.FS) *ProjectEnvironmentBuilder {
	return &ProjectEnvironmentBuilder{fsys: fsys}
}

// WithRepositories sets the repositories to be used for package resolution.
// Repositories will be bound to the Environment during Build().
func (b *ProjectEnvironmentBuilder) WithRepositories(repos []Repository) *ProjectEnvironmentBuilder {
	b.repositories = repos
	return b
}

// WithBuildOptions sets the build options to be used during package resolution.
func (b *ProjectEnvironmentBuilder) WithBuildOptions(options BuildOptions) *ProjectEnvironmentBuilder {
	b.buildOptions = options
	return b
}

func (b *ProjectEnvironmentBuilder) Build() *Environment {
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), b.buildOptions.Stats())
	projEnv := NewEnvironment(b.fsys, env)

	// ResolutionOptions are a runtime-facing subset of BuildOptions; derive
	// them so callers configure a single source of truth (BuildOptions) and
	// the resolver still gets the offline/sticky flags it consults.
	projEnv.resolutionOptions = ResolutionOptions{}.
		WithOffline(b.buildOptions.Offline()).
		WithSticky(b.buildOptions.Sticky())

	// Bind and add repositories. Bundled migrated lang libraries must always be
	// available because they are resolved implicitly by the compiler.
	repositories := append([]Repository{NewFileSystemRepository(langlibs.FS, ".")}, b.repositories...)
	for _, repo := range repositories {
		if repo != nil {
			// Bind the repository to the environment if it supports late binding
			if bindable, ok := repo.(bindableRepository); ok {
				bindable.bind(projEnv)
			}
			projEnv.addRepository(repo)
		}
	}

	return projEnv
}
