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

type TypeTags int

const (
	TypeTags_INT     TypeTags = iota + 1
	TypeTags_BYTE             = TypeTags_INT + 1
	TypeTags_FLOAT            = TypeTags_BYTE + 1
	TypeTags_DECIMAL          = TypeTags_FLOAT + 1
	TypeTags_STRING           = TypeTags_DECIMAL + 1
	TypeTags_BOOLEAN          = TypeTags_STRING + 1
	// All the above types are values type
	TypeTags_JSON        = TypeTags_BOOLEAN + 1
	TypeTags_XML         = TypeTags_JSON + 1
	TypeTags_TABLE       = TypeTags_XML + 1
	TypeTags_NIL         = TypeTags_TABLE + 1
	TypeTags_ANYDATA     = TypeTags_NIL + 1
	TypeTags_RECORD      = TypeTags_ANYDATA + 1
	TypeTags_TYPEDESC    = TypeTags_RECORD + 1
	TypeTags_TYPEREFDESC = TypeTags_TYPEDESC + 1
	TypeTags_STREAM      = TypeTags_TYPEREFDESC + 1
	TypeTags_MAP         = TypeTags_STREAM + 1
	TypeTags_INVOKABLE   = TypeTags_MAP + 1
	// All the above types are branded types
	TypeTags_ANY              = TypeTags_INVOKABLE + 1
	TypeTags_ENDPOINT         = TypeTags_ANY + 1
	TypeTags_ARRAY            = TypeTags_ENDPOINT + 1
	TypeTags_UNION            = TypeTags_ARRAY + 1
	TypeTags_INTERSECTION     = TypeTags_UNION + 1
	TypeTags_PACKAGE          = TypeTags_INTERSECTION + 1
	TypeTags_NONE             = TypeTags_PACKAGE + 1
	TypeTags_VOID             = TypeTags_NONE + 1
	TypeTags_XMLNS            = TypeTags_VOID + 1
	TypeTags_ANNOTATION       = TypeTags_XMLNS + 1
	TypeTags_SEMANTIC_ERROR   = TypeTags_ANNOTATION + 1
	TypeTags_ERROR            = TypeTags_SEMANTIC_ERROR + 1
	TypeTags_ITERATOR         = TypeTags_ERROR + 1
	TypeTags_TUPLE            = TypeTags_ITERATOR + 1
	TypeTags_FUTURE           = TypeTags_TUPLE + 1
	TypeTags_FINITE           = TypeTags_FUTURE + 1
	TypeTags_OBJECT           = TypeTags_FINITE + 1
	TypeTags_BYTE_ARRAY       = TypeTags_OBJECT + 1
	TypeTags_FUNCTION_POINTER = TypeTags_BYTE_ARRAY + 1
	TypeTags_HANDLE           = TypeTags_FUNCTION_POINTER + 1
	TypeTags_READONLY         = TypeTags_HANDLE + 1

	// Subtypes
	TypeTags_SIGNED32_INT   = TypeTags_READONLY + 1
	TypeTags_SIGNED16_INT   = TypeTags_SIGNED32_INT + 1
	TypeTags_SIGNED8_INT    = TypeTags_SIGNED16_INT + 1
	TypeTags_UNSIGNED32_INT = TypeTags_SIGNED8_INT + 1
	TypeTags_UNSIGNED16_INT = TypeTags_UNSIGNED32_INT + 1
	TypeTags_UNSIGNED8_INT  = TypeTags_UNSIGNED16_INT + 1
	TypeTags_CHAR_STRING    = TypeTags_UNSIGNED8_INT + 1
	TypeTags_XML_ELEMENT    = TypeTags_CHAR_STRING + 1
	TypeTags_XML_PI         = TypeTags_XML_ELEMENT + 1
	TypeTags_XML_COMMENT    = TypeTags_XML_PI + 1
	TypeTags_XML_TEXT       = TypeTags_XML_COMMENT + 1
	TypeTags_NEVER          = TypeTags_XML_TEXT + 1

	TypeTags_NULL_SET           = TypeTags_NEVER + 1
	TypeTags_PARAMETERIZED_TYPE = TypeTags_NULL_SET + 1
	TypeTags_REGEXP             = TypeTags_PARAMETERIZED_TYPE + 1
	TypeTags_EMPTY              = TypeTags_REGEXP + 1

	TypeTags_SEQUENCE = TypeTags_EMPTY + 1
)
