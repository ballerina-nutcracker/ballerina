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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"io/fs"
	"slices"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/lib/langlibs"
	"github.com/ballerina-nutcracker/ballerina/lib/stdlibs"
	"github.com/ballerina-nutcracker/ballerina/ls/core/completion"
)

var predeclaredLangLibs = []string{
	"lang.boolean", "lang.decimal", "lang.error", "lang.float", "lang.function", "lang.future",
	"lang.int", "lang.map", "lang.natural", "lang.object", "lang.stream", "lang.string",
	"lang.table", "lang.transaction", "lang.typedesc", "lang.xml",
}

// bundledPackages lists the lang and standard library packages embedded in the
// binary. Every project resolves them, so completion can offer them without a
// package repository. Predeclared lang libraries are left out.
func bundledPackages() []completion.AvailablePackage {
	var packages []completion.AvailablePackage
	for _, fsys := range []fs.FS{langlibs.FS, stdlibs.FS} {
		orgs, _ := fs.ReadDir(fsys, ".")
		for _, org := range orgs {
			names, _ := fs.ReadDir(fsys, org.Name())
			for _, name := range names {
				if !name.IsDir() || strings.Contains(name.Name(), ".__") || isPredeclaredLangLib(org.Name(), name.Name()) {
					continue
				}
				packages = append(packages, completion.AvailablePackage{Organization: org.Name(), Name: name.Name()})
			}
		}
	}
	return packages
}

func isPredeclaredLangLib(org, name string) bool {
	return org == "ballerina" && slices.Contains(predeclaredLangLibs, name)
}
