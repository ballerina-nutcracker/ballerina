// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package projects

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// FindProjectRoot discovers the nearest package manifest above an absolute
// native source path. Only a missing manifest permits searching a parent.
func FindProjectRoot(openDir func(dir string) fs.FS, file string) (root string, found bool, err error) {
	if !filepath.IsAbs(file) {
		return "", false, fmt.Errorf("project discovery: path is not absolute: %q", file)
	}
	for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
		info, err := fs.Stat(openDir(dir), BallerinaTomlFile)
		if err == nil {
			if info.IsDir() {
				return "", false, fmt.Errorf("project discovery: %s is a directory", filepath.Join(dir, BallerinaTomlFile))
			}
			return dir, true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false, fmt.Errorf("project discovery at %s: %w", dir, err)
		}
		if filepath.Dir(dir) == dir {
			return "", false, nil
		}
	}
}
