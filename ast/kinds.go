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

type NodeKind uint

const (
	NodeKind_ANNOTATION NodeKind = iota
	NodeKind_ANNOTATION_ATTACHMENT
	NodeKind_ANNOTATION_ATTRIBUTE
	NodeKind_COMPILATION_UNIT
	NodeKind_DEPRECATED
	NodeKind_DOCUMENTATION
	NodeKind_MARKDOWN_DOCUMENTATION
	NodeKind_ENDPOINT
	NodeKind_FUNCTION
	NodeKind_RESOURCE_FUNC
	NodeKind_BLOCK_FUNCTION_BODY
	NodeKind_EXPR_FUNCTION_BODY
	NodeKind_EXTERN_FUNCTION_BODY
	NodeKind_IDENTIFIER
	NodeKind_IMPORT
	NodeKind_PACKAGE
	NodeKind_PACKAGE_DECLARATION
	NodeKind_RECORD_LITERAL_KEY_VALUE
	NodeKind_RECORD_LITERAL_SPREAD_OP
	NodeKind_RESOURCE
	NodeKind_SERVICE
	NodeKind_TYPE_DEFINITION
	NodeKind_VARIABLE
	NodeKind_LET_VARIABLE
	NodeKind_TUPLE_VARIABLE
	NodeKind_RECORD_VARIABLE
	NodeKind_ERROR_VARIABLE
	NodeKind_WORKER
	NodeKind_XMLNS
	NodeKind_CHANNEL
	NodeKind_WAIT_LITERAL_KEY_VALUE
	NodeKind_TABLE_KEY_SPECIFIER
	NodeKind_TABLE_KEY_TYPE_CONSTRAINT
	NodeKind_RETRY_SPEC
	NodeKind_CLASS_DEFN
	NodeKind_DOCUMENTATION_ATTRIBUTE
	NodeKind_ARRAY_LITERAL_EXPR
	NodeKind_TUPLE_LITERAL_EXPR
	NodeKind_LIST_CONSTRUCTOR_EXPR
	NodeKind_LIST_CONSTRUCTOR_SPREAD_OP
	NodeKind_BINARY_EXPR
	NodeKind_QUERY_EXPR
	NodeKind_ELVIS_EXPR
	NodeKind_GROUP_EXPR
	NodeKind_TYPE_INIT_EXPR
	NodeKind_FIELD_BASED_ACCESS_EXPR
	NodeKind_INDEX_BASED_ACCESS_EXPR
	NodeKind_INT_RANGE_EXPR
	NodeKind_INVOCATION
	NodeKind_COLLECT_CONTEXT_INVOCATION
	NodeKind_LAMBDA
	NodeKind_ARROW_EXPR
	NodeKind_LITERAL
	NodeKind_NUMERIC_LITERAL
	NodeKind_HEX_FLOATING_POINT_LITERAL
	NodeKind_INTEGER_LITERAL
	NodeKind_DECIMAL_FLOATING_POINT_LITERAL
	NodeKind_CONSTANT
	NodeKind_RECORD_LITERAL_EXPR
	NodeKind_SIMPLE_VARIABLE_REF
	NodeKind_CONSTANT_REF
	NodeKind_TUPLE_VARIABLE_REF
	NodeKind_RECORD_VARIABLE_REF
	NodeKind_ERROR_VARIABLE_REF
	NodeKind_STRING_TEMPLATE_LITERAL
	NodeKind_RAW_TEMPLATE_LITERAL
	NodeKind_TERNARY_EXPR
	NodeKind_WAIT_EXPR
	NodeKind_TRAP_EXPR
	NodeKind_TYPEDESC_EXPRESSION
	NodeKind_ANNOT_ACCESS_EXPRESSION
	NodeKind_TYPE_CONVERSION_EXPR
	NodeKind_IS_ASSIGNABLE_EXPR
	NodeKind_UNARY_EXPR
	NodeKind_REST_ARGS_EXPR
	NodeKind_NAMED_ARGS_EXPR
	NodeKind_XML_QNAME
	NodeKind_XML_ATTRIBUTE
	NodeKind_XML_ATTRIBUTE_ACCESS_EXPR
	NodeKind_XML_QUOTED_STRING
	NodeKind_XML_ELEMENT_LITERAL
	NodeKind_XML_TEXT_LITERAL
	NodeKind_XML_COMMENT_LITERAL
	NodeKind_XML_PI_LITERAL
	NodeKind_XML_SEQUENCE_LITERAL
	NodeKind_XML_ELEMENT_FILTER_EXPR
	NodeKind_XML_ELEMENT_ACCESS
	NodeKind_XML_NAVIGATION
	NodeKind_XML_EXTENDED_NAVIGATION
	NodeKind_XML_STEP_INDEXED_EXTEND
	NodeKind_XML_STEP_FILTER_EXTEND
	NodeKind_XML_STEP_METHOD_CALL_EXTEND
	NodeKind_STATEMENT_EXPRESSION
	NodeKind_MATCH_EXPRESSION
	NodeKind_MATCH_EXPRESSION_PATTERN_CLAUSE
	NodeKind_CHECK_EXPR
	NodeKind_CHECK_PANIC_EXPR
	NodeKind_FAIL
	NodeKind_TYPE_TEST_EXPR
	NodeKind_IS_LIKE
	NodeKind_IGNORE_EXPR
	NodeKind_DOCUMENTATION_DESCRIPTION
	NodeKind_DOCUMENTATION_PARAMETER
	NodeKind_DOCUMENTATION_REFERENCE
	NodeKind_DOCUMENTATION_DEPRECATION
	NodeKind_DOCUMENTATION_DEPRECATED_PARAMETERS
	NodeKind_SERVICE_CONSTRUCTOR
	NodeKind_LET_EXPR
	NodeKind_TABLE_CONSTRUCTOR_EXPR
	NodeKind_TRANSACTIONAL_EXPRESSION
	NodeKind_OBJECT_CTOR_EXPRESSION
	NodeKind_ERROR_CONSTRUCTOR_EXPRESSION
	NodeKind_DYNAMIC_PARAM_EXPR
	NodeKind_INFER_TYPEDESC_EXPR
	NodeKind_REG_EXP_TEMPLATE_LITERAL
	NodeKind_REG_EXP_DISJUNCTION
	NodeKind_REG_EXP_SEQUENCE
	NodeKind_REG_EXP_ATOM_CHAR_ESCAPE
	NodeKind_REG_EXP_ATOM_QUANTIFIER
	NodeKind_REG_EXP_CHARACTER_CLASS
	NodeKind_REG_EXP_CHAR_SET
	NodeKind_REG_EXP_CHAR_SET_RANGE
	NodeKind_REG_EXP_QUANTIFIER
	NodeKind_REG_EXP_ASSERTION
	NodeKind_REG_EXP_CAPTURING_GROUP
	NodeKind_REG_EXP_FLAG_EXPR
	NodeKind_REG_EXP_FLAGS_ON_OFF
	NodeKind_NATURAL_EXPR
	NodeKind_ABORT
	NodeKind_DONE
	NodeKind_RETRY
	NodeKind_RETRY_TRANSACTION
	NodeKind_ASSIGNMENT
	NodeKind_COMPOUND_ASSIGNMENT
	NodeKind_POST_INCREMENT
	NodeKind_BLOCK
	NodeKind_BREAK
	NodeKind_NEXT
	NodeKind_EXPRESSION_STATEMENT
	NodeKind_FOREACH
	NodeKind_FORK_JOIN
	NodeKind_IF
	NodeKind_MATCH
	NodeKind_MATCH_STATEMENT
	NodeKind_MATCH_TYPED_PATTERN_CLAUSE
	NodeKind_MATCH_STATIC_PATTERN_CLAUSE
	NodeKind_MATCH_STRUCTURED_PATTERN_CLAUSE
	NodeKind_REPLY
	NodeKind_RETURN
	NodeKind_THROW
	NodeKind_PANIC
	NodeKind_TRANSACTION
	NodeKind_TRANSFORM
	NodeKind_TUPLE_DESTRUCTURE
	NodeKind_RECORD_DESTRUCTURE
	NodeKind_ERROR_DESTRUCTURE
	NodeKind_VARIABLE_DEF
	NodeKind_WHILE
	NodeKind_LOCK
	NodeKind_WORKER_RECEIVE
	NodeKind_ALTERNATE_WORKER_RECEIVE
	NodeKind_MULTIPLE_WORKER_RECEIVE
	NodeKind_WORKER_ASYNC_SEND
	NodeKind_WORKER_SYNC_SEND
	NodeKind_WORKER_FLUSH
	NodeKind_STREAM
	NodeKind_SCOPE
	NodeKind_COMPENSATE
	NodeKind_CHANNEL_RECEIVE
	NodeKind_CHANNEL_SEND
	NodeKind_DO_ACTION
	NodeKind_COMMIT
	NodeKind_ROLLBACK
	NodeKind_DO_STMT
	NodeKind_SELECT
	NodeKind_COLLECT
	NodeKind_FROM
	NodeKind_JOIN
	NodeKind_WHERE
	NodeKind_DO
	NodeKind_LET_CLAUSE
	NodeKind_ON_CONFLICT
	NodeKind_ON
	NodeKind_LIMIT
	NodeKind_ORDER_BY
	NodeKind_ORDER_KEY
	NodeKind_GROUP_BY
	NodeKind_GROUPING_KEY
	NodeKind_ON_FAIL
	NodeKind_MATCH_CLAUSE
	NodeKind_MATCH_GUARD
	NodeKind_CONST_MATCH_PATTERN
	NodeKind_WILDCARD_MATCH_PATTERN
	NodeKind_VAR_BINDING_PATTERN_MATCH_PATTERN
	NodeKind_LIST_MATCH_PATTERN
	NodeKind_REST_MATCH_PATTERN
	NodeKind_MAPPING_MATCH_PATTERN
	NodeKind_FIELD_MATCH_PATTERN
	NodeKind_ERROR_MATCH_PATTERN
	NodeKind_ERROR_MESSAGE_MATCH_PATTERN
	NodeKind_ERROR_CAUSE_MATCH_PATTERN
	NodeKind_ERROR_FIELD_MATCH_PATTERN
	NodeKind_NAMED_ARG_MATCH_PATTERN
	NodeKind_SIMPLE_MATCH_PATTERN
	NodeKind_WILDCARD_BINDING_PATTERN
	NodeKind_CAPTURE_BINDING_PATTERN
	NodeKind_LIST_BINDING_PATTERN
	NodeKind_REST_BINDING_PATTERN
	NodeKind_FIELD_BINDING_PATTERN
	NodeKind_MAPPING_BINDING_PATTERN
	NodeKind_ERROR_BINDING_PATTERN
	NodeKind_ERROR_MESSAGE_BINDING_PATTERN
	NodeKind_ERROR_CAUSE_BINDING_PATTERN
	NodeKind_ERROR_FIELD_BINDING_PATTERN
	NodeKind_NAMED_ARG_BINDING_PATTERN
	NodeKind_SIMPLE_BINDING_PATTERN
	NodeKind_ARRAY_TYPE
	NodeKind_MEMBER_TYPE_DESC
	NodeKind_UNION_TYPE_NODE
	NodeKind_INTERSECTION_TYPE_NODE
	NodeKind_FINITE_TYPE_NODE
	NodeKind_TUPLE_TYPE_NODE
	NodeKind_BUILT_IN_REF_TYPE
	NodeKind_CONSTRAINED_TYPE
	NodeKind_FUNCTION_TYPE
	NodeKind_USER_DEFINED_TYPE
	NodeKind_VALUE_TYPE
	NodeKind_RECORD_TYPE
	NodeKind_OBJECT_TYPE
	NodeKind_OBJECT_FIELD
	NodeKind_METHOD_DECL
	NodeKind_ERROR_TYPE
	NodeKind_STREAM_TYPE
	NodeKind_TABLE_TYPE
	NodeKind_NODE_ENTRY
	NodeKind_RESOURCE_PATH_IDENTIFIER_SEGMENT
	NodeKind_RESOURCE_PATH_PARAM_SEGMENT
	NodeKind_RESOURCE_PATH_REST_PARAM_SEGMENT
	NodeKind_RESOURCE_ROOT_PATH_SEGMENT
	NodeKind_INFERRED_TYPEDESC_DEFAULT
)

type TypeKind string

const (
	TypeKind_INT           TypeKind = "int"
	TypeKind_BYTE          TypeKind = "byte"
	TypeKind_FLOAT         TypeKind = "float"
	TypeKind_DECIMAL       TypeKind = "decimal"
	TypeKind_STRING        TypeKind = "string"
	TypeKind_BOOLEAN       TypeKind = "boolean"
	TypeKind_BLOB          TypeKind = "blob"
	TypeKind_TYPEDESC      TypeKind = "typedesc"
	TypeKind_TYPEREFDESC   TypeKind = "typerefdesc"
	TypeKind_STREAM        TypeKind = "stream"
	TypeKind_TABLE         TypeKind = "table"
	TypeKind_JSON          TypeKind = "json"
	TypeKind_XML           TypeKind = "xml"
	TypeKind_ANY           TypeKind = "any"
	TypeKind_ANYDATA       TypeKind = "anydata"
	TypeKind_MAP           TypeKind = "map"
	TypeKind_FUTURE        TypeKind = "future"
	TypeKind_PACKAGE       TypeKind = "package"
	TypeKind_SERVICE       TypeKind = "service"
	TypeKind_CONNECTOR     TypeKind = "connector"
	TypeKind_ENDPOINT      TypeKind = "endpoint"
	TypeKind_FUNCTION      TypeKind = "function"
	TypeKind_ANNOTATION    TypeKind = "annotation"
	TypeKind_ARRAY         TypeKind = "[]"
	TypeKind_UNION         TypeKind = "|"
	TypeKind_INTERSECTION  TypeKind = "&"
	TypeKind_VOID          TypeKind = ""
	TypeKind_NIL           TypeKind = "null"
	TypeKind_NEVER         TypeKind = "never"
	TypeKind_NONE          TypeKind = ""
	TypeKind_OTHER         TypeKind = "other"
	TypeKind_ERROR         TypeKind = "error"
	TypeKind_TUPLE         TypeKind = "tuple"
	TypeKind_OBJECT        TypeKind = "object"
	TypeKind_RECORD        TypeKind = "record"
	TypeKind_FINITE        TypeKind = "finite"
	TypeKind_CHANNEL       TypeKind = "channel"
	TypeKind_HANDLE        TypeKind = "handle"
	TypeKind_READONLY      TypeKind = "readonly"
	TypeKind_TYPEPARAM     TypeKind = "typeparam"
	TypeKind_PARAMETERIZED TypeKind = "parameterized"
	TypeKind_REGEXP        TypeKind = "regexp"
)

type DocumentationReferenceType string
