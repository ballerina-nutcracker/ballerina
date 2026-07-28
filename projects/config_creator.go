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
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"ballerina/common/tomlparser"
)

// balaPackageJSON represents the package.json structure in a .bala package.
type balaPackageJSON struct {
	Organization     string   `json:"organization"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	BallerinaVersion string   `json:"ballerina_version"`
	Platform         string   `json:"platform"`
	Export           []string `json:"export"`
}

// balaDependencyGraph represents the dependency-graph.json structure in a .bala package.
type balaDependencyGraph struct {
	Packages []balaDependencyPackage `json:"packages"`
}

// balaDependencyPackage represents a package entry in dependency-graph.json.
type balaDependencyPackage struct {
	Org          string                  `json:"org"`
	Name         string                  `json:"name"`
	Version      string                  `json:"version"`
	Dependencies []balaDependencyPackage `json:"dependencies"`
}

// balaProjectConfigResult contains the result of creating a bala project config.
type balaProjectConfigResult struct {
	PackageConfig PackageConfig
	Platform      string
	SchemaVersion int // bala schema version; 3 for legacy (package.json), 4+ for Bala.toml
}

// createBalaProjectConfig creates a PackageConfig by scanning an extracted
// .bala directory. Loads the TOML form when Bala.toml is present; otherwise
// falls back to createBalaProjectConfigLegacy for the legacy package.json
// layout.
func createBalaProjectConfig(fsys fs.FS, balaPath string) (balaProjectConfigResult, error) {
	info, err := fs.Stat(fsys, balaPath)
	if err != nil {
		return balaProjectConfigResult{}, err
	}
	if !info.IsDir() {
		return balaProjectConfigResult{}, &ProjectError{
			Message: "bala path must be a directory: " + balaPath,
		}
	}

	if _, err := fs.Stat(fsys, path.Join(balaPath, BalaTomlFile)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return createBalaProjectConfigLegacy(fsys, balaPath)
		}
		return balaProjectConfigResult{}, &ProjectError{
			Message: "failed to stat " + BalaTomlFile + ": " + err.Error(),
		}
	}

	tomlPath := path.Join(balaPath, BallerinaTomlFile)
	toml, err := tomlparser.Read(fsys, tomlPath)
	if err != nil {
		return balaProjectConfigResult{}, &ProjectError{
			Message: "failed to read " + BallerinaTomlFile + " from bala: " + err.Error(),
		}
	}
	manifest := newManifestBuilder(toml, balaPath).Build()

	deps, err := readDependenciesTomlDeps(fsys, balaPath, manifest.PackageDescriptor())
	if err != nil {
		return balaProjectConfigResult{}, err
	}
	manifest = NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc:      manifest.PackageDescriptor(),
		Dependencies:     deps,
		BuildOptions:     manifest.BuildOptions(),
		Diagnostics:      manifest.Diagnostics(),
		License:          manifest.License(),
		Authors:          manifest.Authors(),
		Keywords:         manifest.Keywords(),
		ExportedModules:  manifest.ExportedModules(),
		Repository:       manifest.Repository(),
		BallerinaVersion: manifest.BallerinaVersion(),
		Visibility:       manifest.Visibility(),
		Icon:             manifest.Icon(),
		Readme:           manifest.Readme(),
		Description:      manifest.Description(),
	})

	balaToml, err := readBalaToml(fsys, balaPath)
	if err != nil {
		return balaProjectConfigResult{}, err
	}

	packageDesc := manifest.PackageDescriptor()
	packageID := NewPackageID(packageDesc.Name().Value())

	defaultModuleConfig, err := createDefaultModuleConfig(fsys, balaPath, packageDesc, packageID)
	if err != nil {
		return balaProjectConfigResult{}, err
	}
	otherModules, err := createOtherModuleConfigs(fsys, balaPath, packageDesc, packageID)
	if err != nil {
		return balaProjectConfigResult{}, err
	}

	ballerinaTomlContent, err := fs.ReadFile(fsys, tomlPath)
	if err != nil {
		return balaProjectConfigResult{}, err
	}
	ballerinaTomlDoc := NewDocumentConfig(
		NewDocumentID(BallerinaTomlFile, defaultModuleConfig.ModuleID()),
		BallerinaTomlFile,
		string(ballerinaTomlContent),
	)

	config := NewPackageConfig(PackageConfigParams{
		PackageID:       packageID,
		PackageManifest: manifest,
		PackagePath:     balaPath,
		DefaultModule:   defaultModuleConfig,
		OtherModules:    otherModules,
		BallerinaToml:   ballerinaTomlDoc,
		ReadmeMd:        nil, // TODO: read from docs/
	})

	return balaProjectConfigResult{
		PackageConfig: config,
		Platform:      balaToml.platform,
		SchemaVersion: balaToml.schemaVersion,
	}, nil
}

// createBalaProjectConfigLegacy loads the v3 bala format (package.json + dependency-graph.json).
func createBalaProjectConfigLegacy(fsys fs.FS, balaPath string) (balaProjectConfigResult, error) {
	packageJSONPath := path.Join(balaPath, "package.json")
	pkgJSON, err := readBalaPackageJSON(fsys, packageJSONPath)
	if err != nil {
		return balaProjectConfigResult{}, err
	}

	pkgVersion, err := NewPackageVersionFromString(pkgJSON.Version)
	if err != nil {
		return balaProjectConfigResult{}, &ProjectError{
			Message: "invalid version in package.json: " + pkgJSON.Version,
		}
	}

	packageDesc := NewPackageDescriptor(
		NewPackageOrg(pkgJSON.Organization),
		NewPackageName(pkgJSON.Name),
		pkgVersion,
	)

	depGraph, err := readBalaDependencyGraph(fsys, balaPath)
	if err != nil {
		return balaProjectConfigResult{}, err
	}

	dependencies := extractDependencies(depGraph, pkgJSON.Organization, pkgJSON.Name, pkgJSON.Version)

	manifest := NewPackageManifestFromParams(PackageManifestParams{
		PackageDesc:     packageDesc,
		ExportedModules: pkgJSON.Export,
		Dependencies:    dependencies,
	})

	packageID := NewPackageID(pkgJSON.Name)

	modulesPath := path.Join(balaPath, ModulesDir)
	moduleConfigs, defaultModuleConfig, err := scanBalaModules(fsys, modulesPath, packageDesc, packageID, pkgJSON.Name)
	if err != nil {
		return balaProjectConfigResult{}, err
	}

	config := NewPackageConfig(PackageConfigParams{
		PackageID:       packageID,
		PackageManifest: manifest,
		PackagePath:     balaPath,
		DefaultModule:   defaultModuleConfig,
		OtherModules:    moduleConfigs,
		BallerinaToml:   nil,
		ReadmeMd:        nil, // TODO: read from docs/
	})

	return balaProjectConfigResult{
		PackageConfig: config,
		Platform:      pkgJSON.Platform,
		SchemaVersion: 3,
	}, nil
}

// balaTomlData holds the fields extracted from a single parse of Bala.toml.
type balaTomlData struct {
	platform      string
	schemaVersion int
}

// readBalaToml parses Bala.toml once and returns platform and schema version.
// platform defaults to "any" and schemaVersion defaults to 4 when absent.
func readBalaToml(fsys fs.FS, balaPath string) (balaTomlData, error) {
	tomlPath := path.Join(balaPath, BalaTomlFile)
	t, err := tomlparser.Read(fsys, tomlPath)
	if err != nil {
		return balaTomlData{}, &ProjectError{
			Message: "failed to read " + BalaTomlFile + ": " + err.Error(),
		}
	}

	platform := BalaPlatformAny
	if v, ok := t.GetString("build.platform"); ok && v != "" {
		platform = v
	}

	schemaVersion := 4
	if v, ok := t.GetString("bala.schema_version"); ok && v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return balaTomlData{}, &ProjectError{
				Message: "invalid bala.schema_version in " + BalaTomlFile + ": " + v,
			}
		}
		schemaVersion = n
	}

	return balaTomlData{platform: platform, schemaVersion: schemaVersion}, nil
}

// readDependenciesTomlDeps reads Dependencies.toml from a TOML-format bala and
// returns the direct dependencies of desc. Returns an error if the file is absent
// since Dependencies.toml is required in every valid TOML-format bala.
func readDependenciesTomlDeps(fsys fs.FS, balaPath string, desc PackageDescriptor) ([]Dependency, error) {
	depsPath := path.Join(balaPath, DependenciesTomlFile)
	t, err := tomlparser.Read(fsys, depsPath)
	if err != nil {
		return nil, &ProjectError{Message: "failed to read " + DependenciesTomlFile + ": " + err.Error()}
	}

	packages, ok := t.GetTables("package")
	if !ok {
		return nil, nil
	}

	// Build version lookup: "org/name" → version string
	versions := make(map[string]string, len(packages))
	for _, pkg := range packages {
		o, _ := pkg.GetString("org")
		n, _ := pkg.GetString("name")
		v, _ := pkg.GetString("version")
		if o != "" && n != "" && v != "" {
			versions[o+"/"+n] = v
		}
	}

	// Find the entry for desc and extract its direct deps
	wantOrg := desc.Org().Value()
	wantName := desc.Name().Value()
	wantVer := desc.Version().String()
	for _, pkg := range packages {
		o, _ := pkg.GetString("org")
		n, _ := pkg.GetString("name")
		v, _ := pkg.GetString("version")
		if o != wantOrg || n != wantName || v != wantVer {
			continue
		}
		rawDeps, ok := pkg.GetArray("dependencies")
		if !ok {
			return nil, nil
		}
		var deps []Dependency
		for _, raw := range rawDeps {
			entry, ok := raw.(map[string]any)
			if !ok {
				return nil, &ProjectError{Message: "malformed dependency entry in " + DependenciesTomlFile}
			}
			depOrg, _ := entry["org"].(string)
			depName, _ := entry["name"].(string)
			depVer := versions[depOrg+"/"+depName]
			if depOrg == "" || depName == "" || depVer == "" {
				return nil, &ProjectError{Message: "incomplete dependency entry in " + DependenciesTomlFile}
			}
			ver, err := NewPackageVersionFromString(depVer)
			if err != nil {
				return nil, &ProjectError{Message: "invalid dependency version in " + DependenciesTomlFile + ": " + err.Error()}
			}
			deps = append(deps, NewDependency(NewPackageOrg(depOrg), NewPackageName(depName), ver))
		}
		return deps, nil
	}
	return nil, nil
}

// readBalaPackageJSON reads and parses the package.json file.
func readBalaPackageJSON(fsys fs.FS, path string) (*balaPackageJSON, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, &ProjectError{
			Message: "failed to read package.json: " + err.Error(),
		}
	}

	var pkg balaPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, &ProjectError{
			Message: "failed to parse package.json: " + err.Error(),
		}
	}

	return &pkg, nil
}

// readBalaDependencyGraph reads and parses the dependency-graph.json file.
// Returns nil if the file doesn't exist (optional file).
func readBalaDependencyGraph(fsys fs.FS, balaPath string) (*balaDependencyGraph, error) {
	depGraphPath := path.Join(balaPath, "dependency-graph.json")
	data, err := fs.ReadFile(fsys, depGraphPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // File is optional
		}
		return nil, &ProjectError{
			Message: "failed to read dependency-graph.json: " + err.Error(),
		}
	}

	var graph balaDependencyGraph
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, &ProjectError{
			Message: "failed to parse dependency-graph.json: " + err.Error(),
		}
	}

	return &graph, nil
}

// extractDependencies extracts direct dependencies for the given package from dependency-graph.json.
func extractDependencies(graph *balaDependencyGraph, org, name, version string) []Dependency {
	if graph == nil {
		return nil
	}

	for _, pkg := range graph.Packages {
		if pkg.Org == org && pkg.Name == name && pkg.Version == version {
			var deps []Dependency
			for _, dep := range pkg.Dependencies {
				depVersion, err := NewPackageVersionFromString(dep.Version)
				if err != nil {
					continue // Skip invalid versions
				}
				deps = append(deps, NewDependency(
					NewPackageOrg(dep.Org),
					NewPackageName(dep.Name),
					depVersion,
				))
			}
			return deps
		}
	}

	return nil
}

// scanBalaModules scans the modules directory in a bala package.
// Returns other modules and the default module separately.
func scanBalaModules(fsys fs.FS, modulesPath string, packageDesc PackageDescriptor, packageID PackageID, pkgName string) ([]ModuleConfig, ModuleConfig, error) {
	var otherModules []ModuleConfig
	var defaultModule ModuleConfig

	// Check if modules directory exists
	info, err := fs.Stat(fsys, modulesPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No modules directory - create empty default module
			moduleDesc := NewModuleDescriptorForDefaultModule(packageDesc)
			moduleID := NewModuleID(moduleDesc.Name().String(), packageID)
			return nil, NewModuleConfig(moduleID, moduleDesc, nil, nil, nil, nil), nil
		}
		// Propagate other errors (permission, I/O, etc.)
		return nil, ModuleConfig{}, err
	}
	if !info.IsDir() {
		return nil, ModuleConfig{}, &ProjectError{
			Message: "modules path is not a directory: " + modulesPath,
		}
	}

	// List module directories
	entries, err := fs.ReadDir(fsys, modulesPath)
	if err != nil {
		return nil, ModuleConfig{}, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleDirName := entry.Name()
		modulePath := path.Join(modulesPath, moduleDirName)

		// Determine if this is the default module.
		// Default module has the same name as the package
		isDefault := moduleDirName == pkgName

		var moduleNamePart string
		if isDefault {
			moduleNamePart = ""
		} else if strings.HasPrefix(moduleDirName, pkgName+".") {
			// Sub-module: extract the part after "pkgName."
			moduleNamePart = strings.TrimPrefix(moduleDirName, pkgName+".")
		} else {
			// Module name doesn't match expected pattern, use as-is
			moduleNamePart = moduleDirName
		}

		moduleConfig, err := createBalaModuleConfig(fsys, modulePath, moduleNamePart, packageDesc, packageID, isDefault)
		if err != nil {
			return nil, ModuleConfig{}, err
		}

		if isDefault {
			defaultModule = moduleConfig
		} else {
			otherModules = append(otherModules, moduleConfig)
		}
	}

	// If no default module was found, create an empty one
	if defaultModule.ModuleID() == (ModuleID{}) {
		moduleDesc := NewModuleDescriptorForDefaultModule(packageDesc)
		moduleID := NewModuleID(moduleDesc.Name().String(), packageID)
		defaultModule = NewModuleConfig(moduleID, moduleDesc, nil, nil, nil, nil)
	}

	return otherModules, defaultModule, nil
}

// createBalaModuleConfig creates a ModuleConfig for a module in a bala package.
func createBalaModuleConfig(fsys fs.FS, modulePath string, moduleNamePart string, packageDesc PackageDescriptor, packageID PackageID, isDefault bool) (ModuleConfig, error) {
	var moduleDesc ModuleDescriptor
	if isDefault {
		moduleDesc = NewModuleDescriptorForDefaultModule(packageDesc)
	} else {
		moduleName := NewModuleName(packageDesc.Name(), moduleNamePart)
		moduleDesc = NewModuleDescriptor(packageDesc, moduleName)
	}

	moduleID := NewModuleID(moduleDesc.Name().String(), packageID)

	// Scan for .bal files in module directory
	sourceDocs, err := scanBalaBalFiles(fsys, modulePath, moduleID)
	if err != nil {
		return ModuleConfig{}, err
	}

	// Bala packages don't have test files (they're stripped during packaging)
	return NewModuleConfig(
		moduleID,
		moduleDesc,
		sourceDocs,
		nil, // no test docs in bala
		nil, // no readme
		nil, // dependencies
	), nil
}

// scanBalaBalFiles scans a bala module directory for .bal files.
func scanBalaBalFiles(fsys fs.FS, dirPath string, moduleID ModuleID) ([]DocumentConfig, error) {
	entries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		return nil, err
	}

	var docs []DocumentConfig
	var fileNames []string

	// Collect .bal file names
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), BalFileExtension) {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}

	// Sort for deterministic ordering
	sort.Strings(fileNames)

	// Create DocumentConfigs
	for _, fileName := range fileNames {
		filePath := path.Join(dirPath, fileName)
		content, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return nil, err
		}

		docID := NewDocumentID(fileName, moduleID)
		doc := NewDocumentConfig(docID, fileName, string(content))
		docs = append(docs, doc)
	}

	return docs, nil
}

// createBuildProjectConfig creates a PackageConfig by scanning the project directory.
// This is the main entry point for loading build projects (projects with Ballerina.toml).
func createBuildProjectConfig(fsys fs.FS, projectDirPath string) (PackageConfig, error) {
	// Verify project directory exists
	info, err := fs.Stat(fsys, projectDirPath)
	if err != nil {
		return PackageConfig{}, err
	}
	if !info.IsDir() {
		return PackageConfig{}, &ProjectError{
			Message: "project path must be a directory: " + projectDirPath,
		}
	}

	// Verify Ballerina.toml exists
	ballerinaTomlPath := path.Join(projectDirPath, BallerinaTomlFile)
	if _, err := fs.Stat(fsys, ballerinaTomlPath); os.IsNotExist(err) {
		return PackageConfig{}, &ProjectError{
			Message: "Ballerina.toml not found in: " + projectDirPath,
		}
	}

	// Parse Ballerina.toml
	toml, err := tomlparser.Read(fsys, ballerinaTomlPath)
	if err != nil {
		return PackageConfig{}, err
	}

	// Build manifest from TOML
	manifestBuilder := newManifestBuilder(toml, projectDirPath)
	manifest := manifestBuilder.Build()

	// Create package ID with package name from manifest
	packageID := NewPackageID(manifest.PackageDescriptor().Name().Value())

	// Create package descriptor from manifest
	packageDesc := manifest.PackageDescriptor()

	// Scan and create default module config
	defaultModuleConfig, err := createDefaultModuleConfig(fsys, projectDirPath, packageDesc, packageID)
	if err != nil {
		return PackageConfig{}, err
	}

	// Scan and create other module configs
	otherModules, err := createOtherModuleConfigs(fsys, projectDirPath, packageDesc, packageID)
	if err != nil {
		return PackageConfig{}, err
	}

	// Use the default module's ID for package-level documents
	defaultModuleID := defaultModuleConfig.ModuleID()

	// Create Ballerina.toml document config
	ballerinaTomlContent, err := fs.ReadFile(fsys, ballerinaTomlPath)
	if err != nil {
		return PackageConfig{}, err
	}
	ballerinaTomlDocID := NewDocumentID(BallerinaTomlFile, defaultModuleID)
	ballerinaTomlDoc := NewDocumentConfig(ballerinaTomlDocID, BallerinaTomlFile, string(ballerinaTomlContent))

	// Check for README.md
	var readmeMdDoc DocumentConfig
	readmeMdPath := path.Join(projectDirPath, ReadmeMdFile)
	if _, err := fs.Stat(fsys, readmeMdPath); err == nil {
		readmeMdContent, err := fs.ReadFile(fsys, readmeMdPath)
		if err == nil {
			readmeMdDocID := NewDocumentID(ReadmeMdFile, defaultModuleID)
			readmeMdDoc = NewDocumentConfig(readmeMdDocID, ReadmeMdFile, string(readmeMdContent))
		}
	}

	// Build PackageConfig
	return NewPackageConfig(PackageConfigParams{
		PackageID:       packageID,
		PackageManifest: manifest,
		PackagePath:     projectDirPath,
		DefaultModule:   defaultModuleConfig,
		OtherModules:    otherModules,
		BallerinaToml:   ballerinaTomlDoc,
		ReadmeMd:        readmeMdDoc,
	}), nil
}

// createDefaultModuleConfig creates a ModuleConfig for the default module.
// The default module contains .bal files in the project root directory.
func createDefaultModuleConfig(fsys fs.FS, projectPath string, packageDesc PackageDescriptor, packageID PackageID) (ModuleConfig, error) {
	moduleDesc := NewModuleDescriptorForDefaultModule(packageDesc)
	moduleID := NewModuleID(moduleDesc.Name().String(), packageID)

	// Scan for .bal files in root directory
	sourceDocs, err := scanBalFiles(fsys, projectPath, moduleID)
	if err != nil {
		return ModuleConfig{}, err
	}

	// Scan for test files in tests/ directory
	testsPath := path.Join(projectPath, TestsDir)
	var testDocs []DocumentConfig
	if info, err := fs.Stat(fsys, testsPath); err == nil && info.IsDir() {
		testDocs, err = scanBalFiles(fsys, testsPath, moduleID)
		if err != nil {
			return ModuleConfig{}, err
		}
	}

	// Check for README.md in module
	var readmeMd DocumentConfig
	readmeMdPath := path.Join(projectPath, ReadmeMdFile)
	if _, err := fs.Stat(fsys, readmeMdPath); err == nil {
		content, err := fs.ReadFile(fsys, readmeMdPath)
		if err == nil {
			readmeMd = NewDocumentConfig(NewDocumentID(ReadmeMdFile, moduleID), ReadmeMdFile, string(content))
		}
	}

	return NewModuleConfig(
		moduleID,
		moduleDesc,
		sourceDocs,
		testDocs,
		readmeMd,
		nil, // dependencies - populated later during resolution
	), nil
}

// createOtherModuleConfigs scans the modules/ directory for named modules.
func createOtherModuleConfigs(fsys fs.FS, projectPath string, packageDesc PackageDescriptor, packageID PackageID) ([]ModuleConfig, error) {
	modulesDir := path.Join(projectPath, ModulesDir)

	// Check if modules/ directory exists
	info, err := fs.Stat(fsys, modulesDir)
	if os.IsNotExist(err) {
		return nil, nil // No named modules
	}

	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, &ProjectError{
			Message: "modules path exists but is not a directory: " + modulesDir,
		}
	}

	// List subdirectories in modules/
	entries, err := fs.ReadDir(fsys, modulesDir)
	if err != nil {
		return nil, err
	}

	var moduleConfigs []ModuleConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleName := entry.Name()
		modulePath := path.Join(modulesDir, moduleName)

		moduleConfig, err := createModuleConfig(fsys, modulePath, moduleName, packageDesc, packageID)
		if err != nil {
			return nil, err
		}

		moduleConfigs = append(moduleConfigs, moduleConfig)
	}

	return moduleConfigs, nil
}

// createModuleConfig creates a ModuleConfig for a named module.
func createModuleConfig(fsys fs.FS, modulePath string, moduleNamePart string, packageDesc PackageDescriptor, packageID PackageID) (ModuleConfig, error) {
	moduleName := NewModuleName(packageDesc.Name(), moduleNamePart)
	moduleDesc := NewModuleDescriptor(packageDesc, moduleName)
	moduleID := NewModuleID(moduleDesc.Name().String(), packageID)

	// Scan for .bal files in module directory
	sourceDocs, err := scanBalFiles(fsys, modulePath, moduleID)
	if err != nil {
		return ModuleConfig{}, err
	}

	// Scan for test files in module's tests/ directory
	testsPath := path.Join(modulePath, TestsDir)
	var testDocs []DocumentConfig
	if info, err := fs.Stat(fsys, testsPath); err == nil && info.IsDir() {
		testDocs, err = scanBalFiles(fsys, testsPath, moduleID)
		if err != nil {
			return ModuleConfig{}, err
		}
	}

	// Check for README.md in module
	var readmeMd DocumentConfig
	readmeMdPath := path.Join(modulePath, ReadmeMdFile)
	if _, err := fs.Stat(fsys, readmeMdPath); err == nil {
		content, err := fs.ReadFile(fsys, readmeMdPath)
		if err == nil {
			readmeMd = NewDocumentConfig(NewDocumentID(ReadmeMdFile, moduleID), ReadmeMdFile, string(content))
		}
	}

	return NewModuleConfig(
		moduleID,
		moduleDesc,
		sourceDocs,
		testDocs,
		readmeMd,
		nil, // dependencies - populated later during resolution
	), nil
}

// scanBalFiles scans a directory for .bal files and creates DocumentConfigs.
func scanBalFiles(fsys fs.FS, dirPath string, moduleID ModuleID) ([]DocumentConfig, error) {
	entries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		return nil, err
	}

	var docs []DocumentConfig
	var fileNames []string

	// Collect .bal file names
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), BalFileExtension) {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}

	// Sort by name for deterministic ordering
	sort.Strings(fileNames)

	// Create DocumentConfigs
	for _, fileName := range fileNames {
		filePath := path.Join(dirPath, fileName)
		content, err := fs.ReadFile(fsys, filePath)
		if err != nil {
			return nil, err
		}

		docID := NewDocumentID(fileName, moduleID)
		doc := NewDocumentConfig(docID, fileName, string(content))
		docs = append(docs, doc)
	}

	return docs, nil
}
