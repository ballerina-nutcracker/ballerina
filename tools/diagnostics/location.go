// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

import "fmt"

// Location represents a position in a source file using byte offsets.
// File identity is stored as an integer index into a DiagnosticEnv.
// To get the file name or line/column information, use a DiagnosticEnv.
type Location struct {
	fileIndex   int
	startOffset int
	endOffset   int
}

// NewLocation creates a new Location. The DiagnosticEnv maps the fileName
// to an integer index for compact storage. The file must already be
// registered via DiagnosticEnv.RegisterFile.
func NewLocation(de *DiagnosticEnv, fileName string, startOffset, endOffset int) Location {
	return NewLocationForIdentity(de, fileName, startOffset, endOffset)
}

// NewLocationForIdentity creates a location using a registered internal file identity.
func NewLocationForIdentity(de *DiagnosticEnv, identity string, startOffset, endOffset int) Location {
	return Location{
		fileIndex:   de.FileIndex(identity),
		startOffset: startOffset,
		endOffset:   endOffset,
	}
}

// NewBuiltinLocation returns a Location for a compiler-synthesized source
// element that has no backing file.
func NewBuiltinLocation() Location {
	return Location{fileIndex: BuiltinFileIndex, startOffset: -1, endOffset: -1}
}

// NewBallerinaTomlLocation returns a Location in the project's Ballerina.toml.
// Offsets are stored as given; line/column resolution is not supported.
func NewBallerinaTomlLocation(startOffset, endOffset int) Location {
	return Location{fileIndex: BallerinaTomlFileIndex, startOffset: startOffset, endOffset: endOffset}
}

// IsLocationEmpty returns true if the Location has no associated source.
// Built-in and Ballerina.toml sentinel locations are not considered empty
// because they still carry a meaningful file name for diagnostics.
func IsLocationEmpty(loc Location) bool {
	return loc.fileIndex == UnknownFileIndex
}

// LocationHasSource returns true if the Location refers to a registered
// source document whose line/column positions can be resolved. Built-in
// and Ballerina.toml sentinel locations return false even though they
// carry a file name.
func LocationHasSource(loc Location) bool {
	return loc.fileIndex > 0
}

// FileIndex returns the file index of the Location.
func (loc *Location) FileIndex() int {
	return loc.fileIndex
}

// StartOffset returns the start byte offset of the Location.
func (loc *Location) StartOffset() int {
	return loc.startOffset
}

// EndOffset returns the end byte offset of the Location.
func (loc *Location) EndOffset() int {
	return loc.endOffset
}

// String returns a string representation of the Location.
func (loc Location) String() string {
	return fmt.Sprintf("(%d:%d,%d)", loc.fileIndex, loc.startOffset, loc.endOffset)
}
