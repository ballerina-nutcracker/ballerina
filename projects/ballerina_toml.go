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

// tomlDocumentContext holds internal state for a TOML document.
type tomlDocumentContext struct {
	name    string
	content string
}

// newTomlDocumentContext creates a tomlDocumentContext from a DocumentConfig.
func newTomlDocumentContext(docConfig DocumentConfig) *tomlDocumentContext {
	if docConfig == nil {
		return nil
	}
	return &tomlDocumentContext{
		name:    docConfig.Name(),
		content: docConfig.Content(),
	}
}

type packageToml struct {
	context         *tomlDocumentContext
	packageInstance *Package
}

func newPackageToml(ctx *tomlDocumentContext, pkg *Package) *packageToml {
	if ctx == nil {
		return nil
	}
	return &packageToml{context: ctx, packageInstance: pkg}
}

func (t *packageToml) Content() string {
	if t == nil || t.context == nil {
		return ""
	}
	return t.context.content
}

func (t *packageToml) PackageInstance() *Package {
	if t == nil {
		return nil
	}
	return t.packageInstance
}

// BallerinaToml represents the 'Ballerina.toml' file in a package.
type BallerinaToml struct {
	*packageToml
}

// newBallerinaToml creates a BallerinaToml from a tomlDocumentContext and Package.
func newBallerinaToml(ctx *tomlDocumentContext, pkg *Package) *BallerinaToml {
	toml := newPackageToml(ctx, pkg)
	if toml == nil {
		return nil
	}
	return &BallerinaToml{packageToml: toml}
}

// Name returns the name of the TOML file.
func (b *BallerinaToml) Name() string {
	return BallerinaTomlFile
}
