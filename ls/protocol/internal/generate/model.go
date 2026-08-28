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

package generate

// Model is the parsed LSP metamodel with source line numbers attached.
type Model struct {
	Version       Metadata       `json:"metaData"`
	Requests      []*Request     `json:"requests"`
	Notifications []*Notification `json:"notifications"`
	Structures    []*Structure   `json:"structures"`
	Enumerations  []*Enumeration `json:"enumerations"`
	TypeAliases   []*TypeAlias   `json:"typeAliases"`
}

// Metadata describes the metamodel version.
type Metadata struct {
	Version string `json:"version"`
	Line    int    `json:"line"`
}

// Request is a parsed LSP request definition.
type Request struct {
	Documentation       string `json:"documentation"`
	ErrorData           *Type  `json:"errorData"`
	Direction           string `json:"messageDirection"`
	Method              string `json:"method"`
	Params              *Type  `json:"params"`
	PartialResult       *Type  `json:"partialResult"`
	Proposed            bool   `json:"proposed"`
	RegistrationMethod  string `json:"registrationMethod"`
	RegistrationOptions *Type  `json:"registrationOptions"`
	Result              *Type  `json:"result"`
	Since               string `json:"since"`
	Line                int    `json:"line"`
	ClientCapability    string `json:"clientCapability"`
	ServerCapability    string `json:"serverCapability"`
	TypeName            string `json:"typeName"`
}

// Notification is a parsed LSP notification definition.
type Notification struct {
	Documentation       string `json:"documentation"`
	Direction           string `json:"messageDirection"`
	Method              string `json:"method"`
	Params              *Type  `json:"params"`
	Proposed            bool   `json:"proposed"`
	RegistrationMethod  string `json:"registrationMethod"`
	RegistrationOptions *Type  `json:"registrationOptions"`
	Since               string `json:"since"`
	Line                int    `json:"line"`
	ClientCapability    string `json:"clientCapability"`
	ServerCapability    string `json:"serverCapability"`
	TypeName            string `json:"typeName"`
}

// Structure is a parsed LSP structure.
type Structure struct {
	Documentation string     `json:"documentation"`
	Extends       []*Type    `json:"extends"`
	Mixins        []*Type    `json:"mixins"`
	Name          string     `json:"name"`
	Properties    []Property `json:"properties"`
	Proposed      bool       `json:"proposed"`
	Since         string     `json:"since"`
	Deprecated    string     `json:"deprecated"`
	Line          int        `json:"line"`
}

// Enumeration is a parsed LSP enumeration.
type Enumeration struct {
	Documentation        string           `json:"documentation"`
	Name                 string           `json:"name"`
	Proposed             bool             `json:"proposed"`
	Since                string           `json:"since"`
	Deprecated           string           `json:"deprecated"`
	SupportsCustomValues bool             `json:"supportsCustomValues"`
	Type                 *Type            `json:"type"`
	Values               []EnumerationEntry `json:"values"`
	Line                 int              `json:"line"`
}

// EnumerationEntry is a single enum value.
type EnumerationEntry struct {
	Documentation string `json:"documentation"`
	Name          string `json:"name"`
	Proposed      bool   `json:"proposed"`
	Since         string `json:"since"`
	Deprecated    string `json:"deprecated"`
	Value         any    `json:"value"`
	Line          int    `json:"line"`
}

// TypeAlias is a parsed LSP type alias.
type TypeAlias struct {
	Documentation string `json:"documentation"`
	Deprecated    string `json:"deprecated"`
	Name          string `json:"name"`
	Proposed      bool   `json:"proposed"`
	Since         string `json:"since"`
	Type          *Type  `json:"type"`
	Line          int    `json:"line"`
}

// Property is a structure field.
type Property struct {
	Name          string `json:"name"`
	Type          *Type  `json:"type"`
	Optional      bool   `json:"optional"`
	Documentation string `json:"documentation"`
	Deprecated    string `json:"deprecated"`
	Since         string `json:"since"`
	Proposed      bool   `json:"proposed"`
	Line          int    `json:"line"`
}

// Type is a parsed LSP type expression.
type Type struct {
	Kind    string  `json:"kind"`
	Items   []*Type `json:"items"`   // and, or, tuple
	Element *Type   `json:"element"` // array
	Name    string  `json:"name"`    // base, reference
	Key     *Type   `json:"key"`     // map
	Value   any     `json:"value"`   // map (value type), stringLiteral, integerLiteral (raw value), literal (map)
	Line    int     `json:"line"`
}

// TypeKind enumerates the known LSP type kinds.
const (
	KindAnd            = "and"
	KindArray          = "array"
	KindBase           = "base"
	KindBooleanLiteral = "booleanLiteral"
	KindIntegerLiteral = "integerLiteral"
	KindLiteral        = "literal"
	KindMap            = "map"
	KindOr             = "or"
	KindReference      = "reference"
	KindStringLiteral  = "stringLiteral"
	KindTuple          = "tuple"
)

// BaseTypes enumerates the primitive names allowed in the metamodel.
var BaseTypes = map[string]bool{
	"URI":          true,
	"DocumentUri":  true,
	"integer":      true,
	"uinteger":     true,
	"decimal":      true,
	"RegExp":       true,
	"string":       true,
	"boolean":      true,
	"null":         true,
}
