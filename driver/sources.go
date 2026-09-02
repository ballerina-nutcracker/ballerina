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
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/common/tomlparser"
	"github.com/ballerina-nutcracker/ballerina/tools/diagnostics"
)

const (
	ballerinaToml  = "Ballerina.toml"
	defaultOrg     = "$anon"
	defaultVersion = "0.0.0"
)

type Dependency struct {
	Org        string
	Name       string
	Version    string
	Repository string
}

type ResolutionOptions struct {
	Offline bool
	Sticky  bool
}

type Manifest struct {
	Dependencies []Dependency
	Resolution   ResolutionOptions
}

type PackageSources struct {
	Descriptor PackageDescriptor
	Manifest   Manifest
	Modules    []*ModuleSources
}

type ModuleSources struct {
	ID            ModuleDescriptor
	Documents     []string
	TestDocuments []string
}

type WorkspaceSources struct {
	Members    []*WorkspaceMemberSources
	Resolution ResolutionOptions
}

type WorkspaceMemberSources struct {
	Root    string
	Sources *PackageSources
}

func (w *WorkspaceSources) Member(root string) (*WorkspaceMemberSources, bool) {
	if w == nil {
		return nil, false
	}
	root = path.Clean(root)
	for _, member := range w.Members {
		if member != nil && member.Root == root {
			return member, true
		}
	}
	return nil, false
}

func FindWorkspaceSources(ctx *Context, fsys fs.FS, workspaceRoot string) (*WorkspaceSources, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if fsys == nil {
		return nil, errors.New("driver: nil filesystem")
	}
	if workspaceRoot == "" || (workspaceRoot != "." && !fs.ValidPath(workspaceRoot)) || containsParent(workspaceRoot) {
		return nil, fmt.Errorf("driver: invalid workspace root %q", workspaceRoot)
	}
	tomlPath := path.Join(workspaceRoot, ballerinaToml)
	toml, readErr := tomlparser.Read(fsys, tomlPath)
	if toml == nil {
		return nil, readErr
	}
	workspace, ok := toml.GetTable("workspace")
	if !ok {
		return nil, nil
	}
	ctx.addRootDiagnostics(convertTOMLDiagnostics(toml.Diagnostics()))
	if readErr != nil && len(toml.Diagnostics()) == 0 {
		return nil, readErr
	}
	packages, ok := workspace.GetArray("packages")
	if !ok || len(packages) == 0 {
		ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic("no packages found in the workspace Ballerina.toml file")})
		return &WorkspaceSources{}, nil
	}

	result := &WorkspaceSources{}
	resolutionSet := false
	for index, value := range packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		memberPath, ok := value.(string)
		if !ok {
			ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(
				fmt.Sprintf("workspace.packages[%d] must be a string, got %T", index, value))})
			continue
		}
		clean := path.Clean(memberPath)
		if path.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(
				"workspace package path must stay within the workspace root: '" + memberPath + "'")})
			continue
		}
		root := path.Join(workspaceRoot, clean)
		if _, err := fs.Stat(fsys, path.Join(root, ballerinaToml)); err != nil {
			ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(
				"could not locate the package path '" + memberPath + "'")})
			continue
		}
		memberToml, memberErr := tomlparser.Read(fsys, path.Join(root, ballerinaToml))
		if memberToml == nil {
			message := "unable to parse Ballerina.toml"
			if memberErr != nil {
				message = memberErr.Error()
			}
			ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(
				"failed to load package '" + memberPath + "': " + message)})
			continue
		}
		if memberDiagnostics := memberToml.Diagnostics(); len(memberDiagnostics) > 0 {
			converted := make([]diagnostics.Diagnostic, 0, len(memberDiagnostics))
			for _, diagnostic := range memberDiagnostics {
				converted = append(converted, manifestDiagnostic(
					"failed to load package '"+memberPath+"': TOML parse error: "+diagnostic.Message))
			}
			ctx.addRootDiagnostics(converted)
			continue
		}
		sources, err := FindSources(ctx, fsys, root, path.Base(clean))
		if err != nil {
			ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(
				"failed to load package '" + memberPath + "': " + err.Error())})
			continue
		}
		if !resolutionSet {
			result.Resolution = sources.Manifest.Resolution
			resolutionSet = true
		}
		result.Members = append(result.Members, &WorkspaceMemberSources{Root: root, Sources: sources})
	}
	return result, nil
}

func FindSources(ctx *Context, fsys fs.FS, inputPath string, packageDirName string) (*PackageSources, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if fsys == nil {
		return nil, errors.New("driver: nil filesystem")
	}
	if inputPath == "" || !fs.ValidPath(inputPath) {
		return nil, fmt.Errorf("driver: invalid input path %q", inputPath)
	}
	info, err := fs.Stat(fsys, inputPath)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if !strings.HasSuffix(path.Base(inputPath), ".bal") {
			return nil, fmt.Errorf("driver: source file must have .bal extension: %s", inputPath)
		}
		name := strings.TrimSuffix(path.Base(inputPath), ".bal")
		descriptor := PackageDescriptor{Org: defaultOrg, Name: name, Version: defaultVersion}
		module := &ModuleSources{ID: ModuleDescriptor{Package: descriptor, Name: name}, Documents: []string{path.Base(inputPath)}}
		return &PackageSources{Descriptor: descriptor, Modules: []*ModuleSources{module}}, nil
	}
	if packageDirName == "" || packageDirName == "." || path.Base(packageDirName) != packageDirName {
		return nil, fmt.Errorf("driver: invalid package directory name %q", packageDirName)
	}
	tomlPath := path.Join(inputPath, ballerinaToml)
	toml, readErr := tomlparser.Read(fsys, tomlPath)
	if toml == nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return nil, fmt.Errorf("ballerina.toml not found in: %s", inputPath)
		}
		return nil, readErr
	}
	ctx.addRootDiagnostics(convertTOMLDiagnostics(toml.Diagnostics()))
	if readErr != nil && len(toml.Diagnostics()) == 0 {
		return nil, readErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	descriptor := PackageDescriptor{Org: defaultOrg, Name: packageDirName, Version: defaultVersion}
	if value, ok := toml.GetString("package.org"); ok && value != "" {
		descriptor.Org = value
	}
	if value, ok := toml.GetString("package.name"); ok && value != "" {
		descriptor.Name = value
	}
	if value, ok := toml.GetString("package.version"); ok && value != "" {
		descriptor.Version = value
	}
	if !validVersion(descriptor.Version) {
		ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic(fmt.Sprintf("invalid version '%s'", descriptor.Version))})
		descriptor.Version = defaultVersion
	}
	manifest := Manifest{}
	if value, ok := toml.GetBool("build-options.offline"); ok {
		manifest.Resolution.Offline = value
	}
	if value, ok := toml.GetBool("build-options.sticky"); ok {
		manifest.Resolution.Sticky = value
	}
	if tables, ok := toml.GetTables("dependency"); ok {
		for _, table := range tables {
			dependency, depErr := dependencyFromTOML(table)
			if depErr != nil {
				ctx.addRootDiagnostics([]diagnostics.Diagnostic{manifestDiagnostic("invalid dependency: " + depErr.Error())})
				continue
			}
			manifest.Dependencies = append(manifest.Dependencies, dependency)
		}
	}
	modules, err := discoverModules(ctx, fsys, inputPath, descriptor)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &PackageSources{Descriptor: descriptor, Manifest: manifest, Modules: modules}, nil
}

func dependencyFromTOML(table *tomlparser.Toml) (Dependency, error) {
	org, ok := table.GetString("org")
	if !ok || org == "" {
		return Dependency{}, errors.New("missing required field 'org'")
	}
	name, ok := table.GetString("name")
	if !ok || name == "" {
		return Dependency{}, errors.New("missing required field 'name'")
	}
	version, ok := table.GetString("version")
	if !ok || version == "" {
		return Dependency{}, errors.New("missing required field 'version'")
	}
	if !validVersion(version) {
		return Dependency{}, fmt.Errorf("invalid version '%s'", version)
	}
	repository, _ := table.GetString("repository")
	return Dependency{Org: org, Name: name, Version: version, Repository: repository}, nil
}

func discoverModules(ctx *Context, fsys fs.FS, root string, descriptor PackageDescriptor) ([]*ModuleSources, error) {
	documents, err := discoverBalFiles(ctx, fsys, root, ".")
	if err != nil {
		return nil, err
	}
	tests, err := discoverOptionalBalFiles(ctx, fsys, root, "tests")
	if err != nil {
		return nil, err
	}
	modules := []*ModuleSources{{ID: ModuleDescriptor{Package: descriptor, Name: descriptor.Name}, Documents: documents, TestDocuments: tests}}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(fsys, path.Join(root, "modules"))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if errors.Is(err, fs.ErrNotExist) {
		return modules, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix := path.Join("modules", entry.Name())
		documents, err := discoverBalFiles(ctx, fsys, root, prefix)
		if err != nil {
			return nil, err
		}
		tests, err := discoverOptionalBalFiles(ctx, fsys, root, path.Join(prefix, "tests"))
		if err != nil {
			return nil, err
		}
		modules = append(modules, &ModuleSources{
			ID:        ModuleDescriptor{Package: descriptor, Name: descriptor.Name + "." + entry.Name()},
			Documents: documents, TestDocuments: tests,
		})
	}
	return modules, nil
}

func discoverOptionalBalFiles(ctx *Context, fsys fs.FS, root, relative string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := fs.Stat(fsys, path.Join(root, relative))
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	return discoverBalFiles(ctx, fsys, root, relative)
}

func discoverBalFiles(ctx *Context, fsys fs.FS, root, relative string) ([]string, error) {
	directory := root
	if relative != "." {
		directory = path.Join(root, relative)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(fsys, directory)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".bal") {
			if relative == "." {
				result = append(result, entry.Name())
			} else {
				result = append(result, path.Join(relative, entry.Name()))
			}
		}
	}
	sort.Strings(result)
	return result, nil
}

func convertTOMLDiagnostics(values []tomlparser.Diagnostic) []diagnostics.Diagnostic {
	result := make([]diagnostics.Diagnostic, 0, len(values))
	for _, value := range values {
		info := diagnostics.NewDiagnosticInfo(nil, value.Message, value.Severity)
		result = append(result, diagnostics.NewDefaultDiagnostic(info, diagnostics.NewBallerinaTomlLocation(0, 0), nil))
	}
	return result
}

func manifestDiagnostic(message string) diagnostics.Diagnostic {
	return diagnostics.NewDefaultDiagnostic(diagnostics.NewDiagnosticInfo(nil, message, diagnostics.Error), diagnostics.NewBallerinaTomlLocation(0, 0), nil)
}
