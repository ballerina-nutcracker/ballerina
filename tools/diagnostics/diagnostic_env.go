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

package diagnostics

import (
	"fmt"
	"sync"

	"ballerina-lang-go/tools/text"
)

// Sentinel fileIndex values for synthetic sources that carry no TextDocument
// and are not registered in the env's slice. Real files use positive indices
// (1-based) assigned at RegisterFile time so the zero-value Location maps to
// UnknownFileIndex.
const (
	// UnknownFileIndex marks a Location with no associated source file.
	// Value 0 so a zero-value Location is treated as unknown.
	UnknownFileIndex = 0
	// BuiltinFileIndex marks a Location in the compiler's synthetic "<built-in>"
	// source. Built-in locations cannot carry user-visible errors.
	BuiltinFileIndex = -1
	// BallerinaTomlFileIndex marks a Location in Ballerina.toml. Manifest
	// validation diagnostics use this without resolving offsets to line/column.
	BallerinaTomlFileIndex = -2
)

const (
	builtinFileName       = "<built-in>"
	ballerinaTomlFileName = "Ballerina.toml"
)

// DiagnosticEnv resolves byte-offset-based Locations to line/column numbers.
// It maps file names to integer indices for compact storage in Location.
// Thread-safe via RWMutex since it is shared across compilation phases.
type DiagnosticEnv struct {
	mu          sync.RWMutex
	fileNames   []string
	nameToIndex map[string]int
	docs        []text.TextDocument
}

// NewDiagnosticEnv creates an empty DiagnosticEnv.
func NewDiagnosticEnv() *DiagnosticEnv {
	return &DiagnosticEnv{
		nameToIndex: make(map[string]int),
	}
}

// RegisterFile adds or updates a file in the environment.
// Assigns 1-based indices so zero-value Location (fileIndex=0) is unknown.
func (de *DiagnosticEnv) RegisterFile(fileName string, doc text.TextDocument) {
	de.mu.Lock()
	defer de.mu.Unlock()
	if idx, ok := de.nameToIndex[fileName]; ok {
		// idx is 1-based, de.docs is 0-based.
		existing := de.docs[idx-1]
		// A dependency re-parsed for a second compile sharing this env may
		// produce a distinct TextDocument instance, so compare by content too.
		if existing == doc || existing.String() == doc.String() {
			return
		}
		panic(fmt.Sprintf("diagnostics: duplicte file declarations with same name: %q", fileName))
	}
	de.fileNames = append(de.fileNames, fileName)
	de.docs = append(de.docs, doc)
	idx := len(de.fileNames)
	de.nameToIndex[fileName] = idx
}

// FileIndex returns the index for a previously registered file name.
// Callers are expected to have invoked RegisterFile first; this panics
// otherwise so missing registrations surface immediately.
func (de *DiagnosticEnv) FileIndex(fileName string) int {
	de.mu.RLock()
	defer de.mu.RUnlock()
	idx, ok := de.nameToIndex[fileName]
	if !ok {
		panic(fmt.Sprintf("diagnostics: file not registered: %q", fileName))
	}
	return idx
}

// FileName returns the file name for a Location.
func (de *DiagnosticEnv) FileName(loc Location) string {
	switch loc.fileIndex {
	case UnknownFileIndex:
		return ""
	case BuiltinFileIndex:
		return builtinFileName
	case BallerinaTomlFileIndex:
		return ballerinaTomlFileName
	}
	de.mu.RLock()
	defer de.mu.RUnlock()
	slot := loc.fileIndex - 1
	if slot < 0 || slot >= len(de.fileNames) {
		panic(fmt.Sprintf("diagnostics: fileIndex %d out of range (have %d files)", loc.fileIndex, len(de.fileNames)))
	}
	return de.fileNames[slot]
}

func (de *DiagnosticEnv) getDoc(loc Location) text.TextDocument {
	de.mu.RLock()
	defer de.mu.RUnlock()
	slot := loc.fileIndex - 1
	return de.docs[slot]
}

// StartLine returns the 0-based start line for the given Location.
// Panics if the Location has no associated source. Callers must check
// IsLocationEmpty and synthetic sentinels (built-in, Ballerina.toml) first.
func (de *DiagnosticEnv) StartLine(loc Location) int {
	doc := de.requireDoc(loc, "StartLine")
	line, _, err := doc.LinePositionFromTextPosition(loc.startOffset)
	if err != nil {
		panic(fmt.Sprintf("diagnostics: StartLine: failed to resolve startOffset %d for loc %+v: %v", loc.startOffset, loc, err))
	}
	return line
}

// StartColumn returns the 0-based start column for the given Location.
// Panics if the Location has no associated source; see StartLine.
func (de *DiagnosticEnv) StartColumn(loc Location) int {
	doc := de.requireDoc(loc, "StartColumn")
	_, col, err := doc.LinePositionFromTextPosition(loc.startOffset)
	if err != nil {
		panic(fmt.Sprintf("diagnostics: StartColumn: failed to resolve startOffset %d for loc %+v: %v", loc.startOffset, loc, err))
	}
	return col
}

// EndLine returns the 0-based end line for the given Location.
// Panics if the Location has no associated source; see StartLine.
func (de *DiagnosticEnv) EndLine(loc Location) int {
	doc := de.requireDoc(loc, "EndLine")
	line, _, err := doc.LinePositionFromTextPosition(loc.endOffset)
	if err != nil {
		panic(fmt.Sprintf("diagnostics: EndLine: failed to resolve endOffset %d for loc %+v: %v", loc.endOffset, loc, err))
	}
	return line
}

// EndColumn returns the 0-based end column for the given Location.
// Panics if the Location has no associated source; see StartLine.
func (de *DiagnosticEnv) EndColumn(loc Location) int {
	doc := de.requireDoc(loc, "EndColumn")
	_, col, err := doc.LinePositionFromTextPosition(loc.endOffset)
	if err != nil {
		panic(fmt.Sprintf("diagnostics: EndColumn: failed to resolve endOffset %d for loc %+v: %v", loc.endOffset, loc, err))
	}
	return col
}

func (de *DiagnosticEnv) requireDoc(loc Location, caller string) text.TextDocument {
	doc := de.getDoc(loc)
	if doc == nil {
		panic(fmt.Sprintf("diagnostics: %s: no source for loc %+v (fileIndex=%d)", caller, loc, loc.fileIndex))
	}
	return doc
}
