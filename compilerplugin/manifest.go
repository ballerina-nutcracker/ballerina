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

package compilerplugin

import (
	"errors"
	"fmt"
	goast "go/ast"
	"go/token"

	"github.com/ballerina-nutcracker/ballerina/common/tomlparser"
)

const afterSemanticsName = "after-semantics"

// ManifestEntry is a single [[plugin]] entry of a CompilerPlugin.toml.
type ManifestEntry struct {
	After    Stage
	Function string
}

// ParseManifest parses CompilerPlugin.toml content and returns its plugin
// entries in declaration order. It reports an error if the manifest has no
// [[plugin]] entry, if an entry has an unsupported stage or a function that is
// not an exported Go identifier, or if an entry is declared more than once.
func ParseManifest(content string) ([]ManifestEntry, error) {
	doc, err := tomlparser.ReadString(content)
	if err != nil {
		return nil, err
	}
	tables, ok := doc.GetTables("plugin")
	if !ok || len(tables) == 0 {
		return nil, errors.New("CompilerPlugin.toml must contain at least one [[plugin]] entry")
	}
	entries := make([]ManifestEntry, 0, len(tables))
	seen := make(map[ManifestEntry]struct{}, len(tables))
	for i, table := range tables {
		stageName, ok := table.GetString("stage")
		if !ok {
			return nil, fmt.Errorf("plugin entry %d must define a string stage", i+1)
		}
		if stageName != afterSemanticsName {
			return nil, fmt.Errorf("plugin entry %d has unsupported stage %q", i+1, stageName)
		}
		function, ok := table.GetString("function")
		if !ok {
			return nil, fmt.Errorf("plugin entry %d must define a string function", i+1)
		}
		if !token.IsIdentifier(function) || !goast.IsExported(function) {
			return nil, fmt.Errorf("plugin entry %d has invalid exported Go function %q", i+1, function)
		}
		entry := ManifestEntry{After: AfterSemantics, Function: function}
		if _, exists := seen[entry]; exists {
			return nil, fmt.Errorf("plugin entry %d duplicates %s at %s", i+1, function, entry.After)
		}
		seen[entry] = struct{}{}
		entries = append(entries, entry)
	}
	return entries, nil
}

// String returns the stage name used in CompilerPlugin.toml.
func (s Stage) String() string {
	switch s {
	case AfterSemantics:
		return afterSemanticsName
	default:
		return fmt.Sprintf("unknown-stage-%d", uint8(s))
	}
}
