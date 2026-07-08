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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type targetMode byte

const (
	singleFileMode targetMode = iota
	packageMode
	multipleFilesMode
)

type benchmarkTarget struct {
	mode  targetMode
	label string
	root  string
	paths []string
}

func resolveTarget(path string) (*benchmarkTarget, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("input path %q not found: %w", path, err)
	}

	if !info.IsDir() {
		if !strings.HasSuffix(info.Name(), ".bal") {
			return nil, fmt.Errorf("file %q is not a .bal file", info.Name())
		}
		return &benchmarkTarget{
			mode:  singleFileMode,
			label: info.Name(),
			root:  "",
			paths: []string{path},
		}, nil
	}

	if _, err := os.Stat(filepath.Join(path, "Ballerina.toml")); err == nil {
		return &benchmarkTarget{
			mode:  packageMode,
			label: info.Name(),
			root:  path,
			paths: []string{path},
		}, nil
	}

	files, err := collectBalFiles(path)
	if err != nil {
		return nil, fmt.Errorf("failed to collect .bal files in directory %q: %w", path, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .bal files found in directory %q", path)
	}
	return &benchmarkTarget{
		mode:  multipleFilesMode,
		label: info.Name(),
		root:  path,
		paths: files,
	}, nil
}

func collectBalFiles(dir string) ([]string, error) {
	var files []string
	visited := make(map[string]struct{})

	var walk func(path string) error
	walk = func(path string) error {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("failed to resolve symlinks for %q: %w", path, err)
		}
		abs, err := filepath.Abs(canonical)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %q: %w", canonical, err)
		}

		if _, seen := visited[abs]; seen {
			return nil
		}
		visited[abs] = struct{}{}

		if _, err := os.Stat(filepath.Join(path, "Ballerina.toml")); err == nil {
			files = append(files, path)
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read directory %q: %w", path, err)
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())

			info, err := os.Stat(entryPath)
			if err != nil {
				return fmt.Errorf("failed to stat %q: %w", entryPath, err)
			}

			if info.IsDir() {
				if err := walk(entryPath); err != nil {
					return err
				}
			} else if strings.HasSuffix(entry.Name(), ".bal") {
				files = append(files, entryPath)
			}
		}
		return nil
	}

	if err := walk(dir); err != nil {
		return nil, fmt.Errorf("failed to collect .bal files: %w", err)
	}
	return files, nil
}
