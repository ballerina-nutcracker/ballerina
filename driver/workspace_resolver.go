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

package driver

import (
	stdcontext "context"
	"errors"
	"io/fs"

	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

type workspaceDependencyResolver struct {
	fsys       fs.FS
	workspace  *WorkspaceSources
	fallback   DependencyResolver
	resolution ResolutionOptions
}

func NewWorkspaceDependencyResolver(fsys fs.FS, workspace *WorkspaceSources,
	fallback DependencyResolver, resolution ResolutionOptions,
) DependencyResolver {
	return &workspaceDependencyResolver{fsys: fsys, workspace: workspace, fallback: fallback, resolution: resolution}
}

func (r *workspaceDependencyResolver) Resolve(ctx stdcontext.Context, request DependencyRequest) (
	fs.FS, string, *PackageSources, []diagnostics.Diagnostic, error,
) {
	if r.fsys == nil || r.workspace == nil || r.fallback == nil {
		return nil, "", nil, nil, errors.New("driver: invalid workspace dependency resolver")
	}
	if err := ctx.Err(); err != nil {
		return nil, "", nil, nil, err
	}
	if request.Repository == "" {
		for _, member := range r.workspace.Members {
			if member == nil || member.Sources == nil {
				continue
			}
			descriptor := member.Sources.Descriptor
			if descriptor.Org != request.Descriptor.Org || descriptor.Name != request.Descriptor.Name {
				continue
			}
			if request.Descriptor.Version != "" && descriptor.Version != request.Descriptor.Version {
				continue
			}
			if request.ModuleName != "" && findModuleSources(member.Sources, request.ModuleName) == nil {
				return nil, "", nil, nil, nil
			}
			return r.fsys, member.Root, member.Sources, nil, nil
		}
	}
	request.Resolution = r.resolution
	return r.fallback.Resolve(ctx, request)
}

var _ DependencyResolver = (*workspaceDependencyResolver)(nil)
