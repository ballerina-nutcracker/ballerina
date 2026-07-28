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

package ast

// AST-owned enums. These tag concrete AST node types and are consumed by
// walk/semantic/desugar stages. They live in ast (not model) because they
// describe AST structure, not language-model concepts.

type TypeKind uint8

const (
	TypeKindNone TypeKind = iota
	TypeKindInt
	TypeKindByte
	TypeKindFloat
	TypeKindDecimal
	TypeKindString
	TypeKindBoolean
	TypeKindBlob
	TypeKindTypeDesc
	TypeKindTypeRefDesc
	TypeKindStream
	TypeKindTable
	TypeKindJSON
	TypeKindXML
	TypeKindAny
	TypeKindAnyData
	TypeKindMap
	TypeKindFuture
	TypeKindPackage
	TypeKindService
	TypeKindConnector
	TypeKindEndpoint
	TypeKindFunction
	TypeKindAnnotation
	TypeKindArray
	TypeKindUnion
	TypeKindIntersection
	TypeKindVoid
	TypeKindNil
	TypeKindNever
	TypeKindOther
	TypeKindError
	TypeKindTuple
	TypeKindObject
	TypeKindRecord
	TypeKindFinite
	TypeKindChannel
	TypeKindHandle
	TypeKindReadOnly
	TypeKindTypeParam
	TypeKindParameterized
	TypeKindRegexp
)

func (kind TypeKind) String() string {
	switch kind {
	case TypeKindNone, TypeKindVoid:
		return ""
	case TypeKindInt:
		return "int"
	case TypeKindByte:
		return "byte"
	case TypeKindFloat:
		return "float"
	case TypeKindDecimal:
		return "decimal"
	case TypeKindString:
		return "string"
	case TypeKindBoolean:
		return "boolean"
	case TypeKindBlob:
		return "blob"
	case TypeKindTypeDesc:
		return "typedesc"
	case TypeKindTypeRefDesc:
		return "typerefdesc"
	case TypeKindStream:
		return "stream"
	case TypeKindTable:
		return "table"
	case TypeKindJSON:
		return "json"
	case TypeKindXML:
		return "xml"
	case TypeKindAny:
		return "any"
	case TypeKindAnyData:
		return "anydata"
	case TypeKindMap:
		return "map"
	case TypeKindFuture:
		return "future"
	case TypeKindPackage:
		return "package"
	case TypeKindService:
		return "service"
	case TypeKindConnector:
		return "connector"
	case TypeKindEndpoint:
		return "endpoint"
	case TypeKindFunction:
		return "function"
	case TypeKindAnnotation:
		return "annotation"
	case TypeKindArray:
		return "[]"
	case TypeKindUnion:
		return "|"
	case TypeKindIntersection:
		return "&"
	case TypeKindNil:
		return "null"
	case TypeKindNever:
		return "never"
	case TypeKindOther:
		return "other"
	case TypeKindError:
		return "error"
	case TypeKindTuple:
		return "tuple"
	case TypeKindObject:
		return "object"
	case TypeKindRecord:
		return "record"
	case TypeKindFinite:
		return "finite"
	case TypeKindChannel:
		return "channel"
	case TypeKindHandle:
		return "handle"
	case TypeKindReadOnly:
		return "readonly"
	case TypeKindTypeParam:
		return "typeparam"
	case TypeKindParameterized:
		return "parameterized"
	case TypeKindRegexp:
		return "regexp"
	default:
		panic("unexpected TypeKind")
	}
}

type DocumentationReferenceType string
