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
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type finding struct {
	position token.Position
	name     string
}

func (f finding) String() string {
	return fmt.Sprintf("%s: nodebase: do not expose Base type %s; keep base types private and expose concrete implementation types instead", f.position, f.name)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: nodebase <directory>...")
		os.Exit(1)
	}
	findings, err := scan(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, f := range findings {
		fmt.Fprintln(os.Stderr, f)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func scan(roots []string) ([]finding, error) {
	var findings []finding
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" || entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".") {
					if path != root {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			findings = append(findings, checkFile(fset, file)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return findings, nil
}

func checkFile(fset *token.FileSet, file *ast.File) []finding {
	var findings []finding
	for _, decl := range file.Decls {
		types, ok := decl.(*ast.GenDecl)
		if !ok || types.Tok != token.TYPE {
			continue
		}
		for _, spec := range types.Specs {
			name := spec.(*ast.TypeSpec).Name
			if ast.IsExported(name.Name) && strings.HasSuffix(name.Name, "Base") {
				findings = append(findings, finding{position: fset.Position(name.Pos()), name: name.Name})
			}
		}
	}
	return findings
}
