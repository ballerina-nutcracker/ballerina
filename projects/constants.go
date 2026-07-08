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

// File and directory constants for Ballerina projects.
// Java source: io.ballerina.projects.util.ProjectConstants
const (
	// BallerinaTomlFile is the name of the package manifest file.
	BallerinaTomlFile = "Ballerina.toml"

	// DependenciesTomlFile is the name of the dependency lock file.
	DependenciesTomlFile = "Dependencies.toml"

	// CloudTomlFile is the name of the cloud configuration file.
	CloudTomlFile = "Cloud.toml"

	// CompilerPluginTomlFile is the name of the compiler plugin manifest.
	CompilerPluginTomlFile = "CompilerPlugin.toml"

	// BalToolTomlFile is the name of the bal tool manifest.
	BalToolTomlFile = "BalTool.toml"

	// PackageMdFile is the package documentation file (deprecated).
	PackageMdFile = "Package.md"

	// ReadmeMdFile is the readme file.
	ReadmeMdFile = "README.md"

	// ModuleMdFile is the module documentation file (deprecated).
	ModuleMdFile = "Module.md"

	// ModulesDir is the directory containing named modules.
	ModulesDir = "modules"

	// TestsDir is the directory containing test sources.
	TestsDir = "tests"

	// ResourcesDir is the directory containing package resources.
	ResourcesDir = "resources"

	// TargetDir is the build output directory.
	TargetDir = "target"

	// BalFileExtension is the extension for Ballerina source files.
	BalFileExtension = ".bal"

	// BalaFileExtension is the extension for compiled package archives.
	BalaFileExtension = ".bala"

	// BalaTomlFile carries the bala-format schema version, build metadata
	// (compiler version, vendor, language spec) and the module list.
	BalaTomlFile = "Bala.toml"

	// BalaPlatformAny indicates a platform-independent bala. Java-platform
	// targets are not migrated to the Go port.
	BalaPlatformAny = "any"

	// TomlFileExtension is the extension for TOML files.
	TomlFileExtension = ".toml"

	// CacheDir is the cache directory under target.
	CacheDir = "cache"

	// RepoBIRCacheName is the directory name for cached BIR files.
	RepoBIRCacheName = "bir"

	// BIRFileExtension is the extension for BIR files.
	BIRFileExtension = ".bir"
)

// Package metadata constants.
const (
	// DefaultOrg is the default organization for unnamed packages.
	DefaultOrg = "$anon"

	// DefaultVersion is the default version string for packages.
	DefaultVersion = "0.0.0"

	// BallerinaOrg is the Ballerina organization name.
	BallerinaOrg = "ballerina"

	// BallerinaInternalOrg is the internal Ballerina organization.
	BallerinaInternalOrg = "ballerinai"
)

// DefaultPackageVersion is the pre-parsed default version for packages.
// This avoids repeated parsing of the default version string.
var DefaultPackageVersion = mustParseVersion(DefaultVersion)

func mustParseVersion(s string) PackageVersion {
	v, err := NewPackageVersionFromString(s)
	if err != nil {
		panic("invalid default version: " + s)
	}
	return v
}
