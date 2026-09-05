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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	balaTomlHeaderRe        = regexp.MustCompile(`^\s*\[+[^\[\]]*\]+\s*$`)
	balaTomlPackageHeaderRe = regexp.MustCompile(`^\s*\[package\]\s*$`)
	balaTomlModuleHeaderRe  = regexp.MustCompile(`^\s*\[\[package\.modules\]\]\s*$`)
	balaTomlReadmeLineRe    = regexp.MustCompile(`^(\s*readme\s*=\s*)"[^"]*"(.*)$`)
	balaTomlIconLineRe      = regexp.MustCompile(`^(\s*icon\s*=\s*)"[^"]*"(.*)$`)
	balaTomlNameLineRe      = regexp.MustCompile(`^\s*name\s*=\s*"([^"]*)"`)
)

// rewriteBallerinaTomlForBala returns content (a Ballerina.toml's text) with
// its readme/icon/module-readme paths pointed at wherever addBalaDocs
// actually bundled that content inside the bala, instead of their original
// project-relative path (which no longer exists once packed). Existing
// readme/icon lines are rewritten in place; a readme resolved only via
// auto-discovery (no explicit line in the source text) is inserted.
// Auto-discovered undeclared modules have no [[package.modules]] block to
// edit or insert into and are left as-is — a lower-priority gap, since
// nothing currently re-derives a manifest from a repacked bala.
func rewriteBallerinaTomlForBala(content string, manifest PackageManifest) string {
	lines := strings.Split(content, "\n")

	type moduleBlock struct {
		start      int // index of the first line after the [[package.modules]] header
		name       string
		readmeLine int // -1 if no explicit readme line
	}

	packageStart, packageReadmeLine, packageIconLine := -1, -1, -1
	var modules []moduleBlock

	const (
		sectionNone = iota
		sectionPackage
		sectionModule
	)
	section := sectionNone
	var cur moduleBlock

	closeSection := func() {
		if section == sectionModule {
			modules = append(modules, cur)
		}
		section = sectionNone
	}

	for i, line := range lines {
		switch {
		case balaTomlPackageHeaderRe.MatchString(line):
			closeSection()
			section = sectionPackage
			packageStart = i + 1
			continue
		case balaTomlModuleHeaderRe.MatchString(line):
			closeSection()
			section = sectionModule
			cur = moduleBlock{start: i + 1, readmeLine: -1}
			continue
		case balaTomlHeaderRe.MatchString(line):
			closeSection()
			continue
		}

		switch section {
		case sectionPackage:
			if balaTomlReadmeLineRe.MatchString(line) {
				packageReadmeLine = i
			} else if balaTomlIconLineRe.MatchString(line) {
				packageIconLine = i
			}
		case sectionModule:
			if m := balaTomlNameLineRe.FindStringSubmatch(line); m != nil {
				cur.name = m[1]
			}
			if balaTomlReadmeLineRe.MatchString(line) {
				cur.readmeLine = i
			}
		}
	}
	closeSection()

	if packageReadmeLine >= 0 && manifest.Readme() != "" {
		lines[packageReadmeLine] = rewriteQuotedValue(lines[packageReadmeLine], balaDocPath(manifest.Readme()))
	}
	if packageIconLine >= 0 && manifest.Icon() != "" {
		lines[packageIconLine] = rewriteQuotedValue(lines[packageIconLine], balaDocPath(manifest.Icon()))
	}

	var insertions []struct {
		before int
		text   string
	}
	if packageReadmeLine < 0 && packageStart >= 0 && manifest.Readme() != "" {
		insertions = append(insertions, struct {
			before int
			text   string
		}{packageStart, readmeAssignment(balaDocPath(manifest.Readme()))})
	}

	for _, mod := range modules {
		if mod.name == "" {
			continue
		}
		modManifest := findManifestModule(manifest, mod.name)
		if modManifest == nil || modManifest.Readme() == "" {
			continue
		}
		zipPath := balaModuleDocPath(mod.name, modManifest.Readme())
		if mod.readmeLine >= 0 {
			lines[mod.readmeLine] = rewriteQuotedValue(lines[mod.readmeLine], zipPath)
			continue
		}
		insertions = append(insertions, struct {
			before int
			text   string
		}{mod.start, readmeAssignment(zipPath)})
	}

	if len(insertions) == 0 {
		return strings.Join(lines, "\n")
	}

	sort.Slice(insertions, func(i, j int) bool { return insertions[i].before < insertions[j].before })
	var out []string
	pos := 0
	for _, ins := range insertions {
		out = append(out, lines[pos:ins.before]...)
		out = append(out, ins.text)
		pos = ins.before
	}
	out = append(out, lines[pos:]...)
	return strings.Join(out, "\n")
}

func rewriteQuotedValue(line, newValue string) string {
	if m := balaTomlReadmeLineRe.FindStringSubmatch(line); m != nil {
		return m[1] + strconv.Quote(newValue) + m[2]
	}
	if m := balaTomlIconLineRe.FindStringSubmatch(line); m != nil {
		return m[1] + strconv.Quote(newValue) + m[2]
	}
	return line
}

func readmeAssignment(zipPath string) string {
	return fmt.Sprintf("readme = %s", strconv.Quote(zipPath))
}

func findManifestModule(manifest PackageManifest, name string) *ManifestModule {
	for _, mod := range manifest.Modules() {
		if mod.Name() == name {
			return &mod
		}
	}
	return nil
}
