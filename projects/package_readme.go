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

// mdDocumentContext holds internal state for a markdown document (e.g. a
// package's readme).
type mdDocumentContext struct {
	name    string
	content string
}

// newMdDocumentContext creates an mdDocumentContext from a DocumentConfig.
func newMdDocumentContext(docConfig DocumentConfig) *mdDocumentContext {
	if docConfig == nil {
		return nil
	}
	return &mdDocumentContext{
		name:    docConfig.Name(),
		content: docConfig.Content(),
	}
}

// PackageReadmeMd represents a package's readme document — the file named
// by PackageManifest.Readme() (default "README.md", or an explicit/custom
// path declared via `readme = "..."` in Ballerina.toml).
// Java source: io.ballerina.projects.PackageReadmeMd
type PackageReadmeMd struct {
	context         *mdDocumentContext
	packageInstance *Package
}

// newPackageReadmeMd creates a PackageReadmeMd from an mdDocumentContext and Package.
func newPackageReadmeMd(ctx *mdDocumentContext, pkg *Package) *PackageReadmeMd {
	if ctx == nil {
		return nil
	}
	return &PackageReadmeMd{
		context:         ctx,
		packageInstance: pkg,
	}
}

// Name returns the readme document's file name.
func (r *PackageReadmeMd) Name() string {
	if r.context == nil {
		return ""
	}
	return r.context.name
}

// Content returns the readme document's content.
func (r *PackageReadmeMd) Content() string {
	if r.context == nil {
		return ""
	}
	return r.context.content
}

// PackageInstance returns the package this readme belongs to.
func (r *PackageReadmeMd) PackageInstance() *Package {
	return r.packageInstance
}
