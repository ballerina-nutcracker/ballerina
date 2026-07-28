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
	"context"

	"ballerina/tools/diagnostics"
)

// blendedManifest is the per-resolution view of consumer dependency policy.
// Indexed by "<org>/<name>" for O(1) lookup.
type blendedManifest struct {
	index map[string]blendedDependency
}

// blendedDependency is the blended-view record for a single dep.
type blendedDependency struct {
	org        PackageOrg
	name       PackageName
	version    PackageVersion
	repository string
}

func (d blendedDependency) Org() PackageOrg         { return d.org }
func (d blendedDependency) Name() PackageName       { return d.name }
func (d blendedDependency) Version() PackageVersion { return d.version }

// Repository returns the user-specified repository name (empty if unset).
func (d blendedDependency) Repository() string { return d.repository }

// blendedManifestOption configures blendedManifest sources at construction time.
type blendedManifestOption func(*blendedManifestBuilder)

// blendedManifestBuilder accumulates the sources that feed into a blendedManifest.
type blendedManifestBuilder struct {
	fromManifest      PackageManifest
	packageResolver   PackageResolver // optional; nil skips repository validation
	resolutionOptions ResolutionOptions
	reportDiagnostic  func(diagnostics.Diagnostic) // optional; nil silently drops missed entries
}

// withPackageManifest sets the root PackageManifest source. Required.
func withPackageManifest(m PackageManifest) blendedManifestOption {
	return func(b *blendedManifestBuilder) { b.fromManifest = m }
}

// withPackageResolver enables upfront validation of user-specified repositories.
func withPackageResolver(resolver PackageResolver, opts ResolutionOptions) blendedManifestOption {
	return func(b *blendedManifestBuilder) {
		b.packageResolver = resolver
		b.resolutionOptions = opts
	}
}

// withDiagnosticReporter installs the callback for repository miss warnings.
func withDiagnosticReporter(report func(diagnostics.Diagnostic)) blendedManifestOption {
	return func(b *blendedManifestBuilder) { b.reportDiagnostic = report }
}

// newBlendedManifest builds the blended view from one or more sources.
func newBlendedManifest(opts ...blendedManifestOption) *blendedManifest {
	b := &blendedManifestBuilder{}
	for _, opt := range opts {
		opt(b)
	}
	return buildBlendedManifest(b)
}

// buildBlendedManifest materializes the blendedManifest. When a PackageResolver
// is configured, each `repository = "..."` entry is validated upfront; misses
// are dropped from the index (and reported if a diagnostic reporter is set).
func buildBlendedManifest(b *blendedManifestBuilder) *blendedManifest {
	bm := &blendedManifest{
		index: make(map[string]blendedDependency),
	}
	for _, dep := range b.fromManifest.Dependencies() {
		if dep.Repository() != "" && b.packageResolver != nil {
			desc := NewPackageDescriptor(dep.Org(), dep.Name(), dep.Version())
			req := newResolutionRequestWithRepository(desc, dep.Repository())
			responses := b.packageResolver.ResolvePackages(
				context.Background(),
				[]ResolutionRequest{req},
				b.resolutionOptions,
			)
			if len(responses) == 0 || !responses[0].IsResolved() {
				if b.reportDiagnostic != nil {
					emitLocalRepoMissWarning(
						b.reportDiagnostic,
						dep.Org().Value(),
						dep.Name().Value(),
						dep.Version().String(),
						dep.Repository(),
					)
				}
				continue
			}
		}

		key := dep.Org().Value() + "/" + dep.Name().Value()
		bm.index[key] = blendedDependency{
			org:        dep.Org(),
			name:       dep.Name(),
			version:    dep.Version(),
			repository: dep.Repository(),
		}
	}
	return bm
}

// emitLocalRepoMissWarning warns that a dep routed to a user-specified repository is
// missing there; the caller then drops it so resolution falls back to the chain.
func emitLocalRepoMissWarning(report func(diagnostics.Diagnostic), org, pkgName, version, repoName string) {
	info := diagnostics.NewDiagnosticInfo(
		nil,
		"dependency %s/%s:%s cannot be found in the '%s' repository. falling back to default repositories",
		diagnostics.Warning,
	)
	loc := diagnostics.NewBallerinaTomlLocation(0, 0)
	report(diagnostics.NewDefaultDiagnostic(info, loc, nil, org, pkgName, version, repoName))
}

// dependency returns the blended-view entry for (org, name). A nil receiver
// returns zero+false so single-file loads and tests don't have to construct one.
func (b *blendedManifest) dependency(org, name string) (blendedDependency, bool) {
	if b == nil {
		return blendedDependency{}, false
	}
	d, ok := b.index[org+"/"+name]
	return d, ok
}
