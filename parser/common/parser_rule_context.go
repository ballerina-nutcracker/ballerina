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

package common

import (
	"ballerina/tools/diagnostics"
)

// FIXME: make this private
type ParserRuleContext struct {
	value string
	name  string
}

func (p ParserRuleContext) String() string {
	return p.value
}

var (
	// Productions
	PARSER_RULE_CONTEXT_COMP_UNIT                                  = ParserRuleContext{value: "comp-unit", name: "COMP_UNIT"}
	PARSER_RULE_CONTEXT_EOF                                        = ParserRuleContext{value: "eof", name: "EOF"}
	PARSER_RULE_CONTEXT_TOP_LEVEL_NODE                             = ParserRuleContext{value: "top-level-node", name: "TOP_LEVEL_NODE"}
	PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_METADATA            = ParserRuleContext{value: "top-level-node-without-metadata", name: "TOP_LEVEL_NODE_WITHOUT_METADATA"}
	PARSER_RULE_CONTEXT_TOP_LEVEL_NODE_WITHOUT_MODIFIER            = ParserRuleContext{value: "top-level-node-without-modifier", name: "TOP_LEVEL_NODE_WITHOUT_MODIFIER"}
	PARSER_RULE_CONTEXT_FUNC_DEF                                   = ParserRuleContext{value: "func-def", name: "FUNC_DEF"}
	PARSER_RULE_CONTEXT_FUNC_DEF_START                             = ParserRuleContext{value: "function-def-start", name: "FUNC_DEF_START"}
	PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE                      = ParserRuleContext{value: "func-def-or-func-type", name: "FUNC_DEF_OR_FUNC_TYPE"}
	PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER                   = ParserRuleContext{value: "func-def-first-qualifier", name: "FUNC_DEF_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER                  = ParserRuleContext{value: "func-def-second-qualifier", name: "FUNC_DEF_SECOND_QUALIFIER"}
	PARSER_RULE_CONTEXT_FUNC_DEF_WITHOUT_FIRST_QUALIFIER           = ParserRuleContext{value: "func-def-without-first-qualifier", name: "FUNC_DEF_WITHOUT_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_PARAM_LIST                                 = ParserRuleContext{value: "parameters", name: "PARAM_LIST"}
	PARSER_RULE_CONTEXT_PARAMETER_START                            = ParserRuleContext{value: "parameter-start", name: "PARAMETER_START"}
	PARSER_RULE_CONTEXT_PARAMETER_START_WITHOUT_ANNOTATION         = ParserRuleContext{value: "parameter-start-without-annotation", name: "PARAMETER_START_WITHOUT_ANNOTATION"}
	PARSER_RULE_CONTEXT_PARAM_END                                  = ParserRuleContext{value: "param-end", name: "PARAM_END"}
	PARSER_RULE_CONTEXT_REQUIRED_PARAM                             = ParserRuleContext{value: "required-parameter", name: "REQUIRED_PARAM"}
	PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM                          = ParserRuleContext{value: "defaultable-parameter", name: "DEFAULTABLE_PARAM"}
	PARSER_RULE_CONTEXT_REST_PARAM                                 = ParserRuleContext{value: "rest-parameter", name: "REST_PARAM"}
	PARSER_RULE_CONTEXT_PARAM_START                                = ParserRuleContext{value: "parameter-start", name: "PARAM_START"}
	PARSER_RULE_CONTEXT_PARAM_RHS                                  = ParserRuleContext{value: "param-rhs", name: "PARAM_RHS"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_PARAM_RHS                        = ParserRuleContext{value: "function-type-desc-param-rhs", name: "FUNC_TYPE_PARAM_RHS"}
	PARSER_RULE_CONTEXT_REST_PARAM_RHS                             = ParserRuleContext{value: "rest-param-rhs", name: "REST_PARAM_RHS"}
	PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE                       = ParserRuleContext{value: "after-parameter-type", name: "AFTER_PARAMETER_TYPE"}
	PARSER_RULE_CONTEXT_PARAMETER_NAME_RHS                         = ParserRuleContext{value: "parameter-name-rhs", name: "PARAMETER_NAME_RHS"}
	PARSER_RULE_CONTEXT_REQUIRED_PARAM_NAME_RHS                    = ParserRuleContext{value: "required-param-name-rhs", name: "REQUIRED_PARAM_NAME_RHS"}
	PARSER_RULE_CONTEXT_FUNC_OPTIONAL_RETURNS                      = ParserRuleContext{value: "func-optional-returns", name: "FUNC_OPTIONAL_RETURNS"}
	PARSER_RULE_CONTEXT_FUNC_BODY                                  = ParserRuleContext{value: "func-body", name: "FUNC_BODY"}
	PARSER_RULE_CONTEXT_FUNC_BODY_OR_TYPE_DESC_RHS                 = ParserRuleContext{value: "func-body-or-type-desc-rhs", name: "FUNC_BODY_OR_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_ANON_FUNC_BODY                             = ParserRuleContext{value: "annon-func-body", name: "ANON_FUNC_BODY"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_END                         = ParserRuleContext{value: "func-type-desc-end", name: "FUNC_TYPE_DESC_END"}
	PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY                         = ParserRuleContext{value: "external-func-body", name: "EXTERNAL_FUNC_BODY"}
	PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS         = ParserRuleContext{value: "external-func-body-optional-annots", name: "EXTERNAL_FUNC_BODY_OPTIONAL_ANNOTS"}
	PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK                            = ParserRuleContext{value: "func-body-block", name: "FUNC_BODY_BLOCK"}
	PARSER_RULE_CONTEXT_MODULE_TYPE_DEFINITION                     = ParserRuleContext{value: "type-definition", name: "MODULE_TYPE_DEFINITION"}
	PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION                    = ParserRuleContext{value: "class-definition", name: "MODULE_CLASS_DEFINITION"}
	PARSER_RULE_CONTEXT_MODULE_CLASS_DEFINITION_START              = ParserRuleContext{value: "class-definition-start", name: "MODULE_CLASS_DEFINITION_START"}
	PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER                 = ParserRuleContext{value: "first-class-type-qualifier", name: "FIRST_CLASS_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER                = ParserRuleContext{value: "second-class-type-qualifier", name: "SECOND_CLASS_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER                 = ParserRuleContext{value: "third-class-type-qualifier", name: "THIRD_CLASS_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER                = ParserRuleContext{value: "fourth-class-type-qualifier", name: "FOURTH_CLASS_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_FIRST_QUALIFIER          = ParserRuleContext{value: "class-def-without-first-qualifier", name: "CLASS_DEF_WITHOUT_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_SECOND_QUALIFIER         = ParserRuleContext{value: "class-def-without-second-qualifier", name: "CLASS_DEF_WITHOUT_SECOND_QUALIFIER"}
	PARSER_RULE_CONTEXT_CLASS_DEF_WITHOUT_THIRD_QUALIFIER          = ParserRuleContext{value: "class-def-without-third-qualifier", name: "CLASS_DEF_WITHOUT_THIRD_QUALIFIER"}
	PARSER_RULE_CONTEXT_FIELD_OR_REST_DESCIPTOR_RHS                = ParserRuleContext{value: "field-or-rest-descriptor-rhs", name: "FIELD_OR_REST_DESCIPTOR_RHS"}
	PARSER_RULE_CONTEXT_FIELD_DESCRIPTOR_RHS                       = ParserRuleContext{value: "field-descriptor-rhs", name: "FIELD_DESCRIPTOR_RHS"}
	PARSER_RULE_CONTEXT_RECORD_BODY_START                          = ParserRuleContext{value: "record-body-start", name: "RECORD_BODY_START"}
	PARSER_RULE_CONTEXT_RECORD_BODY_END                            = ParserRuleContext{value: "record-body-end", name: "RECORD_BODY_END"}
	PARSER_RULE_CONTEXT_RECORD_FIELD                               = ParserRuleContext{value: "record-field", name: "RECORD_FIELD"}
	PARSER_RULE_CONTEXT_RECORD_FIELD_OR_RECORD_END                 = ParserRuleContext{value: "record-field-orrecord-end", name: "RECORD_FIELD_OR_RECORD_END"}
	PARSER_RULE_CONTEXT_RECORD_FIELD_START                         = ParserRuleContext{value: "record-field-start", name: "RECORD_FIELD_START"}
	PARSER_RULE_CONTEXT_RECORD_FIELD_WITHOUT_METADATA              = ParserRuleContext{value: "record-field-without-metadata", name: "RECORD_FIELD_WITHOUT_METADATA"}
	PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR                            = ParserRuleContext{value: "type-descriptor", name: "TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_TYPE_DESC_WITHOUT_ISOLATED                 = ParserRuleContext{value: "type-desc-without-isolated", name: "TYPE_DESC_WITHOUT_ISOLATED"}
	PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR                           = ParserRuleContext{value: "class-descriptor", name: "CLASS_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_RECORD_TYPE_DESCRIPTOR                     = ParserRuleContext{value: "record-type-desc", name: "RECORD_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_TYPE_REFERENCE                             = ParserRuleContext{value: "type-reference", name: "TYPE_REFERENCE"}
	PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION           = ParserRuleContext{value: "type-reference-in-type-inclusion", name: "TYPE_REFERENCE_IN_TYPE_INCLUSION"}
	PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER                = ParserRuleContext{value: "simple-type-desc-identifier", name: "SIMPLE_TYPE_DESC_IDENTIFIER"}
	PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN                        = ParserRuleContext{value: "(", name: "ARG_LIST_OPEN_PAREN"}
	PARSER_RULE_CONTEXT_ARG_LIST                                   = ParserRuleContext{value: "arguments", name: "ARG_LIST"}
	PARSER_RULE_CONTEXT_ARG_START                                  = ParserRuleContext{value: "argument-start", name: "ARG_START"}
	PARSER_RULE_CONTEXT_ARG_END                                    = ParserRuleContext{value: "arg-end", name: "ARG_END"}
	PARSER_RULE_CONTEXT_ARG_LIST_END                               = ParserRuleContext{value: "argument-end", name: "ARG_LIST_END"}
	PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN                       = ParserRuleContext{value: ")", name: "ARG_LIST_CLOSE_PAREN"}
	PARSER_RULE_CONTEXT_ARG_START_OR_ARG_LIST_END                  = ParserRuleContext{value: "arg-start-or-args-list-end", name: "ARG_START_OR_ARG_LIST_END"}
	PARSER_RULE_CONTEXT_NAMED_OR_POSITIONAL_ARG_RHS                = ParserRuleContext{value: "named-or-positional-arg", name: "NAMED_OR_POSITIONAL_ARG_RHS"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR                     = ParserRuleContext{value: "object-type-desc", name: "OBJECT_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER                  = ParserRuleContext{value: "object-constructor-member", name: "OBJECT_CONSTRUCTOR_MEMBER"}
	PARSER_RULE_CONTEXT_CLASS_MEMBER                               = ParserRuleContext{value: "class-member", name: "CLASS_MEMBER"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_MEMBER                         = ParserRuleContext{value: "object-type-member", name: "OBJECT_TYPE_MEMBER"}
	PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_START        = ParserRuleContext{value: "class-member-or-object-member-start", name: "CLASS_MEMBER_OR_OBJECT_MEMBER_START"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_MEMBER_START            = ParserRuleContext{value: "object-constructor-member-start", name: "OBJECT_CONSTRUCTOR_MEMBER_START"}
	PARSER_RULE_CONTEXT_CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META = ParserRuleContext{value: "class-member-or-object-member-without-metadata", name: "CLASS_MEMBER_OR_OBJECT_MEMBER_WITHOUT_META"}
	PARSER_RULE_CONTEXT_OBJECT_CONS_MEMBER_WITHOUT_META            = ParserRuleContext{value: "object-constructor-member-without-metadata", name: "OBJECT_CONS_MEMBER_WITHOUT_META"}
	PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD                       = ParserRuleContext{value: "object-func-or-field", name: "OBJECT_FUNC_OR_FIELD"}
	PARSER_RULE_CONTEXT_OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY    = ParserRuleContext{value: "object-func-or-field-without-visibility", name: "OBJECT_FUNC_OR_FIELD_WITHOUT_VISIBILITY"}
	PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL              = ParserRuleContext{value: "object-member-visibility-qual", name: "OBJECT_MEMBER_VISIBILITY_QUAL"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_START                        = ParserRuleContext{value: "object-method-start", name: "OBJECT_METHOD_START"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER              = ParserRuleContext{value: "object-method-first-qualifier", name: "OBJECT_METHOD_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER             = ParserRuleContext{value: "object-method-second-qualifier", name: "OBJECT_METHOD_SECOND_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER              = ParserRuleContext{value: "object-method.third-qualifier", name: "OBJECT_METHOD_THIRD_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER             = ParserRuleContext{value: "object-method-fourth-qualifier", name: "OBJECT_METHOD_FOURTH_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER      = ParserRuleContext{value: "object.method.without.first.qualifier", name: "OBJECT_METHOD_WITHOUT_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER     = ParserRuleContext{value: "object.method.without.transactional", name: "OBJECT_METHOD_WITHOUT_SECOND_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER      = ParserRuleContext{value: "object.method.without.isolated", name: "OBJECT_METHOD_WITHOUT_THIRD_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_FIELD_START                         = ParserRuleContext{value: "object-field-start", name: "OBJECT_FIELD_START"}
	PARSER_RULE_CONTEXT_OBJECT_FIELD_QUALIFIER                     = ParserRuleContext{value: "object-field-qualifier", name: "OBJECT_FIELD_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS                           = ParserRuleContext{value: "object-field-rhs", name: "OBJECT_FIELD_RHS"}
	PARSER_RULE_CONTEXT_OPTIONAL_FIELD_INITIALIZER                 = ParserRuleContext{value: "optional-field-initializer", name: "OPTIONAL_FIELD_INITIALIZER"}
	PARSER_RULE_CONTEXT_ON_FAIL_OPTIONAL_BINDING_PATTERN           = ParserRuleContext{value: "on-fail-optional-binding-pattern", name: "ON_FAIL_OPTIONAL_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER                = ParserRuleContext{value: "first-object-type-qualifier", name: "FIRST_OBJECT_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER               = ParserRuleContext{value: "second-object-type-qualifier", name: "SECOND_OBJECT_TYPE_QUALIFIER"}
	PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER                = ParserRuleContext{value: "first-object-cons-qualifier", name: "FIRST_OBJECT_CONS_QUALIFIER"}
	PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER               = ParserRuleContext{value: "second-object-cons-qualifier", name: "SECOND_OBJECT_CONS_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_CONS_WITHOUT_FIRST_QUALIFIER        = ParserRuleContext{value: "object-cons-without-first-qualifier", name: "OBJECT_CONS_WITHOUT_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER        = ParserRuleContext{value: "object-type-without-first-qualifier", name: "OBJECT_TYPE_WITHOUT_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_START                          = ParserRuleContext{value: "object-type-start", name: "OBJECT_TYPE_START"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_START                   = ParserRuleContext{value: "object-constructor-start", name: "OBJECT_CONSTRUCTOR_START"}
	PARSER_RULE_CONTEXT_IMPORT_DECL                                = ParserRuleContext{value: "import-decl", name: "IMPORT_DECL"}
	PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME                  = ParserRuleContext{value: "import-org-or-module-name", name: "IMPORT_ORG_OR_MODULE_NAME"}
	PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME                         = ParserRuleContext{value: "module-name", name: "IMPORT_MODULE_NAME"}
	PARSER_RULE_CONTEXT_IMPORT_PREFIX                              = ParserRuleContext{value: "import-prefix", name: "IMPORT_PREFIX"}
	PARSER_RULE_CONTEXT_IMPORT_PREFIX_DECL                         = ParserRuleContext{value: "import-alias", name: "IMPORT_PREFIX_DECL"}
	PARSER_RULE_CONTEXT_IMPORT_DECL_ORG_OR_MODULE_NAME_RHS         = ParserRuleContext{value: "import-decl-org-or-module-name-rhs", name: "IMPORT_DECL_ORG_OR_MODULE_NAME_RHS"}
	PARSER_RULE_CONTEXT_AFTER_IMPORT_MODULE_NAME                   = ParserRuleContext{value: "after-import-module-name", name: "AFTER_IMPORT_MODULE_NAME"}
	PARSER_RULE_CONTEXT_SERVICE_DECL                               = ParserRuleContext{value: "service-decl", name: "SERVICE_DECL"}
	PARSER_RULE_CONTEXT_SERVICE_DECL_START                         = ParserRuleContext{value: "service-decl-start", name: "SERVICE_DECL_START"}
	PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER                     = ParserRuleContext{value: "service-decl-qualifier", name: "SERVICE_DECL_QUALIFIER"}
	PARSER_RULE_CONTEXT_SERVICE_DECL_OR_VAR_DECL                   = ParserRuleContext{value: "service-decl-or-var-decl", name: "SERVICE_DECL_OR_VAR_DECL"}
	PARSER_RULE_CONTEXT_SERVICE_VAR_DECL_RHS                       = ParserRuleContext{value: "service-var-decl-rhs", name: "SERVICE_VAR_DECL_RHS"}
	PARSER_RULE_CONTEXT_OPTIONAL_SERVICE_DECL_TYPE                 = ParserRuleContext{value: "optional-service-decl-type", name: "OPTIONAL_SERVICE_DECL_TYPE"}
	PARSER_RULE_CONTEXT_OPTIONAL_ABSOLUTE_PATH                     = ParserRuleContext{value: "optional-absolute-path", name: "OPTIONAL_ABSOLUTE_PATH"}
	PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH                     = ParserRuleContext{value: "absolute-resource-path", name: "ABSOLUTE_RESOURCE_PATH"}
	PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_START               = ParserRuleContext{value: "absolute-resource-path-start", name: "ABSOLUTE_RESOURCE_PATH_START"}
	PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH                 = ParserRuleContext{value: "absolute-path-single-slash", name: "ABSOLUTE_PATH_SINGLE_SLASH"}
	PARSER_RULE_CONTEXT_ABSOLUTE_RESOURCE_PATH_END                 = ParserRuleContext{value: "absolute-resource-path-end", name: "ABSOLUTE_RESOURCE_PATH_END"}
	PARSER_RULE_CONTEXT_SERVICE_DECL_RHS                           = ParserRuleContext{value: "service-decl-rhs", name: "SERVICE_DECL_RHS"}
	PARSER_RULE_CONTEXT_LISTENERS_LIST                             = ParserRuleContext{value: "listeners-list", name: "LISTENERS_LIST"}
	PARSER_RULE_CONTEXT_LISTENERS_LIST_END                         = ParserRuleContext{value: "listeners-list-end", name: "LISTENERS_LIST_END"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_BLOCK                   = ParserRuleContext{value: "object-constructor-block", name: "OBJECT_CONSTRUCTOR_BLOCK"}
	PARSER_RULE_CONTEXT_RESOURCE_KEYWORD_RHS                       = ParserRuleContext{value: "resource-keyword-rhs", name: "RESOURCE_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_OPTIONAL_RELATIVE_PATH                     = ParserRuleContext{value: "optional-relative-path", name: "OPTIONAL_RELATIVE_PATH"}
	PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH                     = ParserRuleContext{value: "relative-resource-path", name: "RELATIVE_RESOURCE_PATH"}
	PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_START               = ParserRuleContext{value: "relative-resource-path-start", name: "RELATIVE_RESOURCE_PATH_START"}
	PARSER_RULE_CONTEXT_RESOURCE_PATH_SEGMENT                      = ParserRuleContext{value: "resource-path-segment", name: "RESOURCE_PATH_SEGMENT"}
	PARSER_RULE_CONTEXT_RESOURCE_PATH_PARAM                        = ParserRuleContext{value: "resource-path-param", name: "RESOURCE_PATH_PARAM"}
	PARSER_RULE_CONTEXT_PATH_PARAM_OPTIONAL_ANNOTS                 = ParserRuleContext{value: "path-param-optional-annots", name: "PATH_PARAM_OPTIONAL_ANNOTS"}
	PARSER_RULE_CONTEXT_PATH_PARAM_ELLIPSIS                        = ParserRuleContext{value: "path-param-ellipsis", name: "PATH_PARAM_ELLIPSIS"}
	PARSER_RULE_CONTEXT_OPTIONAL_PATH_PARAM_NAME                   = ParserRuleContext{value: "optional-path-param-name", name: "OPTIONAL_PATH_PARAM_NAME"}
	PARSER_RULE_CONTEXT_RELATIVE_RESOURCE_PATH_END                 = ParserRuleContext{value: "relative-resource-path-end", name: "RELATIVE_RESOURCE_PATH_END"}
	PARSER_RULE_CONTEXT_RESOURCE_PATH_END                          = ParserRuleContext{value: "relative-resource-path-end", name: "RESOURCE_PATH_END"}
	PARSER_RULE_CONTEXT_RESOURCE_ACCESSOR_DEF_OR_DECL_RHS          = ParserRuleContext{value: "resource-accessor-def-or-decl-rhs", name: "RESOURCE_ACCESSOR_DEF_OR_DECL_RHS"}
	PARSER_RULE_CONTEXT_LISTENER_DECL                              = ParserRuleContext{value: "listener-decl", name: "LISTENER_DECL"}
	PARSER_RULE_CONTEXT_CONSTANT_DECL                              = ParserRuleContext{value: "const-decl", name: "CONSTANT_DECL"}
	PARSER_RULE_CONTEXT_CONST_DECL_TYPE                            = ParserRuleContext{value: "const-decl-type", name: "CONST_DECL_TYPE"}
	PARSER_RULE_CONTEXT_CONST_DECL_RHS                             = ParserRuleContext{value: "const-decl-rhs", name: "CONST_DECL_RHS"}
	PARSER_RULE_CONTEXT_NIL_TYPE_DESCRIPTOR                        = ParserRuleContext{value: "nil-type-descriptor", name: "NIL_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR                   = ParserRuleContext{value: "optional-type-descriptor", name: "OPTIONAL_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR                      = ParserRuleContext{value: "array-type-descriptor", name: "ARRAY_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_ARRAY_LENGTH                               = ParserRuleContext{value: "array-length", name: "ARRAY_LENGTH"}
	PARSER_RULE_CONTEXT_ARRAY_LENGTH_START                         = ParserRuleContext{value: "array-length-start", name: "ARRAY_LENGTH_START"}
	PARSER_RULE_CONTEXT_ANNOT_REFERENCE                            = ParserRuleContext{value: "annot-reference", name: "ANNOT_REFERENCE"}
	PARSER_RULE_CONTEXT_ANNOTATIONS                                = ParserRuleContext{value: "annots", name: "ANNOTATIONS"}
	PARSER_RULE_CONTEXT_ANNOTATION_END                             = ParserRuleContext{value: "annot-end", name: "ANNOTATION_END"}
	PARSER_RULE_CONTEXT_ANNOTATION_REF_RHS                         = ParserRuleContext{value: "annot-ref-rhs", name: "ANNOTATION_REF_RHS"}
	PARSER_RULE_CONTEXT_DOC_STRING                                 = ParserRuleContext{value: "doc-string", name: "DOC_STRING"}
	PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER                       = ParserRuleContext{value: "qualified-identifier", name: "QUALIFIED_IDENTIFIER"}
	PARSER_RULE_CONTEXT_EQUAL_OR_RIGHT_ARROW                       = ParserRuleContext{value: "equal-or-right-arrow", name: "EQUAL_OR_RIGHT_ARROW"}
	PARSER_RULE_CONTEXT_ANNOTATION_DECL                            = ParserRuleContext{value: "annotation-decl", name: "ANNOTATION_DECL"}
	PARSER_RULE_CONTEXT_ANNOT_DECL_OPTIONAL_TYPE                   = ParserRuleContext{value: "annot-decl-optional-type", name: "ANNOT_DECL_OPTIONAL_TYPE"}
	PARSER_RULE_CONTEXT_ANNOT_DECL_RHS                             = ParserRuleContext{value: "annot-decl-rhs", name: "ANNOT_DECL_RHS"}
	PARSER_RULE_CONTEXT_ANNOT_OPTIONAL_ATTACH_POINTS               = ParserRuleContext{value: "annot-optional-attach-points", name: "ANNOT_OPTIONAL_ATTACH_POINTS"}
	PARSER_RULE_CONTEXT_ANNOT_ATTACH_POINTS_LIST                   = ParserRuleContext{value: "annot-attach-points-list", name: "ANNOT_ATTACH_POINTS_LIST"}
	PARSER_RULE_CONTEXT_ATTACH_POINT                               = ParserRuleContext{value: "attach-point", name: "ATTACH_POINT"}
	PARSER_RULE_CONTEXT_ATTACH_POINT_IDENT                         = ParserRuleContext{value: "attach-point-ident", name: "ATTACH_POINT_IDENT"}
	PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT          = ParserRuleContext{value: "single-keyword-attach-point-ident", name: "SINGLE_KEYWORD_ATTACH_POINT_IDENT"}
	PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT                   = ParserRuleContext{value: "ident-after-object-ident", name: "IDENT_AFTER_OBJECT_IDENT"}
	PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION                  = ParserRuleContext{value: "xml-namespace-decl", name: "XML_NAMESPACE_DECLARATION"}
	PARSER_RULE_CONTEXT_XML_NAMESPACE_PREFIX_DECL                  = ParserRuleContext{value: "namespace-prefix-decl", name: "XML_NAMESPACE_PREFIX_DECL"}
	PARSER_RULE_CONTEXT_DEFAULT_WORKER_INIT                        = ParserRuleContext{value: "default-worker-init", name: "DEFAULT_WORKER_INIT"}
	PARSER_RULE_CONTEXT_NAMED_WORKERS                              = ParserRuleContext{value: "named-workers", name: "NAMED_WORKERS"}
	PARSER_RULE_CONTEXT_WORKER_NAME_RHS                            = ParserRuleContext{value: "worker-name-rhs", name: "WORKER_NAME_RHS"}
	PARSER_RULE_CONTEXT_DEFAULT_WORKER                             = ParserRuleContext{value: "default-worker", name: "DEFAULT_WORKER"}
	PARSER_RULE_CONTEXT_KEY_SPECIFIER                              = ParserRuleContext{value: "key-specifier", name: "KEY_SPECIFIER"}
	PARSER_RULE_CONTEXT_KEY_SPECIFIER_RHS                          = ParserRuleContext{value: "key-specifier-rhs", name: "KEY_SPECIFIER_RHS"}
	PARSER_RULE_CONTEXT_TABLE_KEY_RHS                              = ParserRuleContext{value: "table-key-rhs", name: "TABLE_KEY_RHS"}
	PARSER_RULE_CONTEXT_LET_EXPR_LET_VAR_DECL                      = ParserRuleContext{value: "let-expr-let-var-decl", name: "LET_EXPR_LET_VAR_DECL"}
	PARSER_RULE_CONTEXT_LET_CLAUSE_LET_VAR_DECL                    = ParserRuleContext{value: "let-clause-let-var-decl", name: "LET_CLAUSE_LET_VAR_DECL"}
	PARSER_RULE_CONTEXT_LET_VAR_DECL_START                         = ParserRuleContext{value: "let-var-decl-start", name: "LET_VAR_DECL_START"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC                             = ParserRuleContext{value: "func-type-desc", name: "FUNC_TYPE_DESC"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START                       = ParserRuleContext{value: "func-type-desc-start", name: "FUNC_TYPE_DESC_START"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER                  = ParserRuleContext{value: "func-type-first-qualifier", name: "FUNC_TYPE_FIRST_QUALIFIER"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER                 = ParserRuleContext{value: "func-type-second-qualifier", name: "FUNC_TYPE_SECOND_QUALIFIER"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL    = ParserRuleContext{value: "func-type-desc-start-without-first-qual", name: "FUNC_TYPE_DESC_START_WITHOUT_FIRST_QUAL"}
	PARSER_RULE_CONTEXT_FUNCTION_KEYWORD_RHS                       = ParserRuleContext{value: "func-keyword-rhs", name: "FUNCTION_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_END_OF_TYPE_DESC                           = ParserRuleContext{value: "end-of-type-desc", name: "END_OF_TYPE_DESC"}
	PARSER_RULE_CONTEXT_SELECT_CLAUSE                              = ParserRuleContext{value: "select-clause", name: "SELECT_CLAUSE"}
	PARSER_RULE_CONTEXT_COLLECT_CLAUSE                             = ParserRuleContext{value: "collect-clause", name: "COLLECT_CLAUSE"}
	PARSER_RULE_CONTEXT_RESULT_CLAUSE                              = ParserRuleContext{value: "result-clause", name: "RESULT_CLAUSE"}
	PARSER_RULE_CONTEXT_WHERE_CLAUSE                               = ParserRuleContext{value: "where-clause", name: "WHERE_CLAUSE"}
	PARSER_RULE_CONTEXT_FROM_CLAUSE                                = ParserRuleContext{value: "from-clause", name: "FROM_CLAUSE"}
	PARSER_RULE_CONTEXT_LET_CLAUSE                                 = ParserRuleContext{value: "let-clause", name: "LET_CLAUSE"}
	PARSER_RULE_CONTEXT_MODULE_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS  = ParserRuleContext{value: "module-level-func-type-desc-rhs", name: "MODULE_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_EXPLICIT_ANON_FUNC_EXPR_BODY_START         = ParserRuleContext{value: "explicit-anon-func-expr-body-start", name: "EXPLICIT_ANON_FUNC_EXPR_BODY_START"}
	PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAMS            = ParserRuleContext{value: "braced-expr-or-anon-func-params", name: "BRACED_EXPR_OR_ANON_FUNC_PARAMS"}
	PARSER_RULE_CONTEXT_BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS         = ParserRuleContext{value: "braced-expr-or-anon-func-param-rhs", name: "BRACED_EXPR_OR_ANON_FUNC_PARAM_RHS"}
	PARSER_RULE_CONTEXT_ANON_FUNC_PARAM_RHS                        = ParserRuleContext{value: "anon-func-param-rhs", name: "ANON_FUNC_PARAM_RHS"}
	PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM                   = ParserRuleContext{value: "implicit-anon-func-param", name: "IMPLICIT_ANON_FUNC_PARAM"}
	PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER                       = ParserRuleContext{value: "optional-peer-worker", name: "OPTIONAL_PEER_WORKER"}
	PARSER_RULE_CONTEXT_METHOD_NAME                                = ParserRuleContext{value: "method-name", name: "METHOD_NAME"}
	PARSER_RULE_CONTEXT_PEER_WORKER_NAME                           = ParserRuleContext{value: "peer-worker-name", name: "PEER_WORKER_NAME"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE_RHS                     = ParserRuleContext{value: "type-desc-in-tuple-rhs", name: "TYPE_DESC_IN_TUPLE_RHS"}
	PARSER_RULE_CONTEXT_TUPLE_TYPE_MEMBER_RHS                      = ParserRuleContext{value: "tuple-type-member-rhs", name: "TUPLE_TYPE_MEMBER_RHS"}
	PARSER_RULE_CONTEXT_NIL_OR_PARENTHESISED_TYPE_DESC_RHS         = ParserRuleContext{value: "nil-or-parenthesised-tpe-desc-rhs", name: "NIL_OR_PARENTHESISED_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS  = ParserRuleContext{value: "remote-or-resource-call-or-async-send-rhs", name: "REMOTE_OR_RESOURCE_CALL_OR_ASYNC_SEND_RHS"}
	PARSER_RULE_CONTEXT_REMOTE_CALL_OR_ASYNC_SEND_END              = ParserRuleContext{value: "remote-call-or-async-send-end", name: "REMOTE_CALL_OR_ASYNC_SEND_END"}
	PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND          = ParserRuleContext{value: "default-worker-name-in-async-send", name: "DEFAULT_WORKER_NAME_IN_ASYNC_SEND"}
	PARSER_RULE_CONTEXT_RECEIVE_WORKERS                            = ParserRuleContext{value: "receive-workers", name: "RECEIVE_WORKERS"}
	PARSER_RULE_CONTEXT_MULTI_RECEIVE_WORKERS                      = ParserRuleContext{value: "multi-receive-workers", name: "MULTI_RECEIVE_WORKERS"}
	PARSER_RULE_CONTEXT_RECEIVE_FIELD_END                          = ParserRuleContext{value: "receive-field-end", name: "RECEIVE_FIELD_END"}
	PARSER_RULE_CONTEXT_RECEIVE_FIELD                              = ParserRuleContext{value: "receive-field", name: "RECEIVE_FIELD"}
	PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME                         = ParserRuleContext{value: "receive-field-name", name: "RECEIVE_FIELD_NAME"}
	PARSER_RULE_CONTEXT_INFER_PARAM_END_OR_PARENTHESIS_END         = ParserRuleContext{value: "infer-param-end-or-parenthesis-end", name: "INFER_PARAM_END_OR_PARENTHESIS_END"}
	PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER_END                = ParserRuleContext{value: "list-constructor-member-end", name: "LIST_CONSTRUCTOR_MEMBER_END"}
	PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN                      = ParserRuleContext{value: "typed-binding-pattern", name: "TYPED_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_BINDING_PATTERN                            = ParserRuleContext{value: "binding-pattern", name: "BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_CAPTURE_BINDING_PATTERN                    = ParserRuleContext{value: "capture-binding-pattern", name: "CAPTURE_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_REST_BINDING_PATTERN                       = ParserRuleContext{value: "rest-binding-pattern", name: "REST_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN                       = ParserRuleContext{value: "list-binding-pattern", name: "LIST_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_LIST_BINDING_PATTERNS_START                = ParserRuleContext{value: "list-binding-patterns-start", name: "LIST_BINDING_PATTERNS_START"}
	PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER                = ParserRuleContext{value: "list-binding-pattern-member", name: "LIST_BINDING_PATTERN_MEMBER"}
	PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_MEMBER_END            = ParserRuleContext{value: "list-binding-pattern-member-end", name: "LIST_BINDING_PATTERN_MEMBER_END"}
	PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN                      = ParserRuleContext{value: "field-binding-pattern", name: "FIELD_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME                 = ParserRuleContext{value: "field-binding-pattern-name", name: "FIELD_BINDING_PATTERN_NAME"}
	PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN                    = ParserRuleContext{value: "mapping-binding-pattern", name: "MAPPING_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_MEMBER             = ParserRuleContext{value: "mapping-binding-pattern-member", name: "MAPPING_BINDING_PATTERN_MEMBER"}
	PARSER_RULE_CONTEXT_MAPPING_BINDING_PATTERN_END                = ParserRuleContext{value: "mapping-binding-pattern-end", name: "MAPPING_BINDING_PATTERN_END"}
	PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_END                  = ParserRuleContext{value: "field-binding-pattern-end-or-continue", name: "FIELD_BINDING_PATTERN_END"}
	PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN                      = ParserRuleContext{value: "error-binding-pattern", name: "ERROR_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS    = ParserRuleContext{value: "error-binding-pattern-error-keyword-rhs", name: "ERROR_BINDING_PATTERN_ERROR_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_ERROR_ARG_LIST_BINDING_PATTERN_START       = ParserRuleContext{value: "error-arg-list-binding-pattern-start", name: "ERROR_ARG_LIST_BINDING_PATTERN_START"}
	PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN                     = ParserRuleContext{value: "simple-binding-pattern", name: "SIMPLE_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END          = ParserRuleContext{value: "error-message-binding-pattern-end", name: "ERROR_MESSAGE_BINDING_PATTERN_END"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA    = ParserRuleContext{value: "error-message-binding-pattern-end-comma", name: "ERROR_MESSAGE_BINDING_PATTERN_END_COMMA"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_RHS          = ParserRuleContext{value: "error-message-binding-pattern-rhs", name: "ERROR_MESSAGE_BINDING_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN         = ParserRuleContext{value: "error-cause-simple-binding-pattern", name: "ERROR_CAUSE_SIMPLE_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN                = ParserRuleContext{value: "error-field-binding-pattern", name: "ERROR_FIELD_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN_END            = ParserRuleContext{value: "error-field-binding-pattern-end", name: "ERROR_FIELD_BINDING_PATTERN_END"}
	PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN                  = ParserRuleContext{value: "named-arg-binding-pattern", name: "NAMED_ARG_BINDING_PATTERN"}
	PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER        = ParserRuleContext{value: "binding-pattern-starting-indentifier", name: "BINDING_PATTERN_STARTING_IDENTIFIER"}
	PARSER_RULE_CONTEXT_WAIT_KEYWORD_RHS                           = ParserRuleContext{value: "wait-keyword-rhs", name: "WAIT_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_MULTI_WAIT_FIELDS                          = ParserRuleContext{value: "multi-wait-fields", name: "MULTI_WAIT_FIELDS"}
	PARSER_RULE_CONTEXT_WAIT_FIELD_NAME                            = ParserRuleContext{value: "wait-field-name", name: "WAIT_FIELD_NAME"}
	PARSER_RULE_CONTEXT_WAIT_FIELD_NAME_RHS                        = ParserRuleContext{value: "wait-field-name-rhs", name: "WAIT_FIELD_NAME_RHS"}
	PARSER_RULE_CONTEXT_WAIT_FIELD_END                             = ParserRuleContext{value: "wait-field-end", name: "WAIT_FIELD_END"}
	PARSER_RULE_CONTEXT_WAIT_FUTURE_EXPR_END                       = ParserRuleContext{value: "wait-future-expr-end", name: "WAIT_FUTURE_EXPR_END"}
	PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPRS                       = ParserRuleContext{value: "alternate-wait-exprs", name: "ALTERNATE_WAIT_EXPRS"}
	PARSER_RULE_CONTEXT_ALTERNATE_WAIT_EXPR_LIST_END               = ParserRuleContext{value: "alternate-wait-expr-lit-end", name: "ALTERNATE_WAIT_EXPR_LIST_END"}
	PARSER_RULE_CONTEXT_DO_CLAUSE                                  = ParserRuleContext{value: "do-clause", name: "DO_CLAUSE"}
	PARSER_RULE_CONTEXT_MODULE_ENUM_DECLARATION                    = ParserRuleContext{value: "module-enum-declaration", name: "MODULE_ENUM_DECLARATION"}
	PARSER_RULE_CONTEXT_MODULE_ENUM_NAME                           = ParserRuleContext{value: "module-enum-name", name: "MODULE_ENUM_NAME"}
	PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME                           = ParserRuleContext{value: "enum-member-name", name: "ENUM_MEMBER_NAME"}
	PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR_END                 = ParserRuleContext{value: "member-access-key-expr-end", name: "MEMBER_ACCESS_KEY_EXPR_END"}
	PARSER_RULE_CONTEXT_MEMBER_ACCESS_KEY_EXPR                     = ParserRuleContext{value: "member-access-key-expr", name: "MEMBER_ACCESS_KEY_EXPR"}
	PARSER_RULE_CONTEXT_RETRY_KEYWORD_RHS                          = ParserRuleContext{value: "retry-keyword-rhs", name: "RETRY_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_RETRY_TYPE_PARAM_RHS                       = ParserRuleContext{value: "retry-type-param-rhs", name: "RETRY_TYPE_PARAM_RHS"}
	PARSER_RULE_CONTEXT_RETRY_BODY                                 = ParserRuleContext{value: "retry-body", name: "RETRY_BODY"}
	PARSER_RULE_CONTEXT_ROLLBACK_RHS                               = ParserRuleContext{value: "rollback-rhs", name: "ROLLBACK_RHS"}
	PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST                  = ParserRuleContext{value: "stmt-start-bracketed-list", name: "STMT_START_BRACKETED_LIST"}
	PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_MEMBER           = ParserRuleContext{value: "stmt-start-bracketed-list-member", name: "STMT_START_BRACKETED_LIST_MEMBER"}
	PARSER_RULE_CONTEXT_STMT_START_BRACKETED_LIST_RHS              = ParserRuleContext{value: "stmt-start-bracketed-list-rhs", name: "STMT_START_BRACKETED_LIST_RHS"}
	PARSER_RULE_CONTEXT_BRACKETED_LIST                             = ParserRuleContext{value: "bracketed-list", name: "BRACKETED_LIST"}
	PARSER_RULE_CONTEXT_BRACKETED_LIST_RHS                         = ParserRuleContext{value: "bracketed-list-rhs", name: "BRACKETED_LIST_RHS"}
	PARSER_RULE_CONTEXT_BRACED_LIST_RHS                            = ParserRuleContext{value: "braced-list-rhs", name: "BRACED_LIST_RHS"}
	PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER                      = ParserRuleContext{value: "bracketed-list-member", name: "BRACKETED_LIST_MEMBER"}
	PARSER_RULE_CONTEXT_BRACKETED_LIST_MEMBER_END                  = ParserRuleContext{value: "bracketed-list-member-end", name: "BRACKETED_LIST_MEMBER_END"}
	PARSER_RULE_CONTEXT_LIST_BINDING_MEMBER_OR_ARRAY_LENGTH        = ParserRuleContext{value: "list-binding-member-or-array-length", name: "LIST_BINDING_MEMBER_OR_ARRAY_LENGTH"}
	PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS             = ParserRuleContext{value: "type-binding-pattern-type-rhs", name: "TYPED_BINDING_PATTERN_TYPE_RHS"}
	PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN                = ParserRuleContext{value: "union-or-intersection", name: "UNION_OR_INTERSECTION_TOKEN"}
	PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR          = ParserRuleContext{value: "mapping-bp-or-mapping-cons", name: "MAPPING_BP_OR_MAPPING_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER   = ParserRuleContext{value: "mapping-bp-or-mapping-cons-member", name: "MAPPING_BP_OR_MAPPING_CONSTRUCTOR_MEMBER"}
	PARSER_RULE_CONTEXT_LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER         = ParserRuleContext{value: "list-bp-or-list-cons-member", name: "LIST_BP_OR_LIST_CONSTRUCTOR_MEMBER"}
	PARSER_RULE_CONTEXT_VAR_REF_OR_TYPE_REF                        = ParserRuleContext{value: "var-ref", name: "VAR_REF_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC                = ParserRuleContext{value: "func-desc-type-or-anon-func", name: "FUNC_TYPE_DESC_OR_ANON_FUNC"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC_START          = ParserRuleContext{value: "func-desc-type-or-anon-func-start", name: "FUNC_TYPE_DESC_OR_ANON_FUNC_START"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY       = ParserRuleContext{value: "func-type-desc-rhs-or-anon-func-body", name: "FUNC_TYPE_DESC_RHS_OR_ANON_FUNC_BODY"}
	PARSER_RULE_CONTEXT_STMT_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS    = ParserRuleContext{value: "stmt-level-func-type-desc-rhs", name: "STMT_LEVEL_AMBIGUOUS_FUNC_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_RECORD_FIELD_NAME_OR_TYPE_NAME             = ParserRuleContext{value: "record-field-name-or-type-name", name: "RECORD_FIELD_NAME_OR_TYPE_NAME"}
	PARSER_RULE_CONTEXT_MATCH_BODY                                 = ParserRuleContext{value: "match-body", name: "MATCH_BODY"}
	PARSER_RULE_CONTEXT_MATCH_PATTERN                              = ParserRuleContext{value: "match-pattern", name: "MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_MATCH_PATTERN_START                        = ParserRuleContext{value: "match-pattern-start", name: "MATCH_PATTERN_START"}
	PARSER_RULE_CONTEXT_MATCH_PATTERN_END                          = ParserRuleContext{value: "match-pattern-end", name: "MATCH_PATTERN_END"}
	PARSER_RULE_CONTEXT_MATCH_PATTERN_RHS                          = ParserRuleContext{value: "match-pattern-rhs", name: "MATCH_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_MATCH_PATTERN_LIST_MEMBER_RHS              = ParserRuleContext{value: "match-pattern-list-memebr-rhs", name: "MATCH_PATTERN_LIST_MEMBER_RHS"}
	PARSER_RULE_CONTEXT_OPTIONAL_MATCH_GUARD                       = ParserRuleContext{value: "optional-match-guard", name: "OPTIONAL_MATCH_GUARD"}
	PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN                         = ParserRuleContext{value: "list-match-pattern", name: "LIST_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_LIST_MATCH_PATTERNS_START                  = ParserRuleContext{value: "list-match-patterns-start", name: "LIST_MATCH_PATTERNS_START"}
	PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER                  = ParserRuleContext{value: "list-match-pattern-member", name: "LIST_MATCH_PATTERN_MEMBER"}
	PARSER_RULE_CONTEXT_LIST_MATCH_PATTERN_MEMBER_RHS              = ParserRuleContext{value: "list-match-pattern-member-rhs", name: "LIST_MATCH_PATTERN_MEMBER_RHS"}
	PARSER_RULE_CONTEXT_REST_MATCH_PATTERN                         = ParserRuleContext{value: "rest-match-pattern", name: "REST_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_MAPPING_MATCH_PATTERN                      = ParserRuleContext{value: "mapping-match-pattern", name: "MAPPING_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERNS_START                 = ParserRuleContext{value: "field-match-patterns-start", name: "FIELD_MATCH_PATTERNS_START"}
	PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER_RHS             = ParserRuleContext{value: "field-match-pattern-member-rhs", name: "FIELD_MATCH_PATTERN_MEMBER_RHS"}
	PARSER_RULE_CONTEXT_FIELD_MATCH_PATTERN_MEMBER                 = ParserRuleContext{value: "field-match-pattern-member", name: "FIELD_MATCH_PATTERN_MEMBER"}
	PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN                        = ParserRuleContext{value: "error-match-pattern", name: "ERROR_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS      = ParserRuleContext{value: "error-match-pattern-error-keyword-rhs", name: "ERROR_MATCH_PATTERN_ERROR_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG     = ParserRuleContext{value: "error-arg-list-match-pattern-first-arg", name: "ERROR_ARG_LIST_MATCH_PATTERN_FIRST_ARG"}
	PARSER_RULE_CONTEXT_ERROR_ARG_LIST_MATCH_PATTERN_START         = ParserRuleContext{value: "error-arg-list-match-pattern-start", name: "ERROR_ARG_LIST_MATCH_PATTERN_START"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END            = ParserRuleContext{value: "error-message-match-pattern-end", name: "ERROR_MESSAGE_MATCH_PATTERN_END"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA      = ParserRuleContext{value: "error-message-match-pattern-end-comma", name: "ERROR_MESSAGE_MATCH_PATTERN_END_COMMA"}
	PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_RHS            = ParserRuleContext{value: "error-message-match-pattern-rhs", name: "ERROR_MESSAGE_MATCH_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_ERROR_CAUSE_MATCH_PATTERN                  = ParserRuleContext{value: "error-cause-match-pattern", name: "ERROR_CAUSE_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN                  = ParserRuleContext{value: "error-field-match-pattern", name: "ERROR_FIELD_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_ERROR_FIELD_MATCH_PATTERN_RHS              = ParserRuleContext{value: "error-field-match-pattern-rhs", name: "ERROR_FIELD_MATCH_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN_OR_CONST_PATTERN       = ParserRuleContext{value: "error-match-pattern-or-const-pattern", name: "ERROR_MATCH_PATTERN_OR_CONST_PATTERN"}
	PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN                    = ParserRuleContext{value: "named-arg-match-pattern", name: "NAMED_ARG_MATCH_PATTERN"}
	PARSER_RULE_CONTEXT_NAMED_ARG_MATCH_PATTERN_RHS                = ParserRuleContext{value: "named-arg-match-pattern-rhs", name: "NAMED_ARG_MATCH_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_ORDER_BY_CLAUSE                            = ParserRuleContext{value: "order-by-clause", name: "ORDER_BY_CLAUSE"}
	PARSER_RULE_CONTEXT_ORDER_KEY_LIST                             = ParserRuleContext{value: "order-key-list", name: "ORDER_KEY_LIST"}
	PARSER_RULE_CONTEXT_ORDER_KEY_LIST_END                         = ParserRuleContext{value: "order-key-list-end", name: "ORDER_KEY_LIST_END"}
	PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE                            = ParserRuleContext{value: "group-by-clause", name: "GROUP_BY_CLAUSE"}
	PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT                  = ParserRuleContext{value: "grouping-key-list-element", name: "GROUPING_KEY_LIST_ELEMENT"}
	PARSER_RULE_CONTEXT_GROUPING_KEY_LIST_ELEMENT_END              = ParserRuleContext{value: "grouping-key-list-element-end", name: "GROUPING_KEY_LIST_ELEMENT_END"}
	PARSER_RULE_CONTEXT_GROUP_BY_CLAUSE_END                        = ParserRuleContext{value: "group-by-clause-end", name: "GROUP_BY_CLAUSE_END"}
	PARSER_RULE_CONTEXT_ON_CONFLICT_CLAUSE                         = ParserRuleContext{value: "on-conflict-clause", name: "ON_CONFLICT_CLAUSE"}
	PARSER_RULE_CONTEXT_LIMIT_CLAUSE                               = ParserRuleContext{value: "limit-clause", name: "LIMIT_CLAUSE"}
	PARSER_RULE_CONTEXT_JOIN_CLAUSE                                = ParserRuleContext{value: "join-clause", name: "JOIN_CLAUSE"}
	PARSER_RULE_CONTEXT_JOIN_CLAUSE_START                          = ParserRuleContext{value: "join-clause-start", name: "JOIN_CLAUSE_START"}
	PARSER_RULE_CONTEXT_JOIN_CLAUSE_END                            = ParserRuleContext{value: "join-clause-end", name: "JOIN_CLAUSE_END"}
	PARSER_RULE_CONTEXT_ON_CLAUSE                                  = ParserRuleContext{value: "on-clause", name: "ON_CLAUSE"}
	PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE                        = ParserRuleContext{value: "intermediate-clause", name: "INTERMEDIATE_CLAUSE"}
	PARSER_RULE_CONTEXT_INTERMEDIATE_CLAUSE_START                  = ParserRuleContext{value: "intermediate-clause-start", name: "INTERMEDIATE_CLAUSE_START"}
	PARSER_RULE_CONTEXT_ON_FAIL_CLAUSE                             = ParserRuleContext{value: "on_fail_clause", name: "ON_FAIL_CLAUSE"}
	PARSER_RULE_CONTEXT_ON_FA                                      = ParserRuleContext{value: "on_fail_clause", name: "ON_FA"}
	PARSER_RULE_CONTEXT_OPTIONAL_TYPE_PARAMETER                    = ParserRuleContext{value: "optional-type-parameter", name: "OPTIONAL_TYPE_PARAMETER"}
	PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE                         = ParserRuleContext{value: "parameterized-type", name: "PARAMETERIZED_TYPE"}
	PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR                        = ParserRuleContext{value: "map-type-descriptor", name: "MAP_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_MODULE_VAR_DECL                            = ParserRuleContext{value: "module-var-decl", name: "MODULE_VAR_DECL"}
	PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL                      = ParserRuleContext{value: "module-var-first-qual", name: "MODULE_VAR_FIRST_QUAL"}
	PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL                     = ParserRuleContext{value: "module-var-second-qual", name: "MODULE_VAR_SECOND_QUAL"}
	PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL                      = ParserRuleContext{value: "module-var-third-qual", name: "MODULE_VAR_THIRD_QUAL"}
	PARSER_RULE_CONTEXT_MODULE_VAR_DECL_START                      = ParserRuleContext{value: "module-var-decl-start", name: "MODULE_VAR_DECL_START"}
	PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_FIRST_QUAL              = ParserRuleContext{value: "module-var-without-first-qual", name: "MODULE_VAR_WITHOUT_FIRST_QUAL"}
	PARSER_RULE_CONTEXT_MODULE_VAR_WITHOUT_SECOND_QUAL             = ParserRuleContext{value: "module-var-without-second-qual", name: "MODULE_VAR_WITHOUT_SECOND_QUAL"}
	PARSER_RULE_CONTEXT_FUNC_DEF_OR_TYPE_DESC_RHS                  = ParserRuleContext{value: "func-def-or-type-desc-rhs", name: "FUNC_DEF_OR_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_CLIENT_RESOURCE_ACCESS_ACTION              = ParserRuleContext{value: "client-resource-access-action", name: "CLIENT_RESOURCE_ACCESS_ACTION"}
	PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_PATH              = ParserRuleContext{value: "optional-resource-access-path", name: "OPTIONAL_RESOURCE_ACCESS_PATH"}
	PARSER_RULE_CONTEXT_RESOURCE_ACCESS_PATH_SEGMENT               = ParserRuleContext{value: "resource-access-path-segment", name: "RESOURCE_ACCESS_PATH_SEGMENT"}
	PARSER_RULE_CONTEXT_COMPUTED_SEGMENT_OR_REST_SEGMENT           = ParserRuleContext{value: "computed-segment-or-rest-segment", name: "COMPUTED_SEGMENT_OR_REST_SEGMENT"}
	PARSER_RULE_CONTEXT_RESOURCE_ACCESS_SEGMENT_RHS                = ParserRuleContext{value: "resource-access-segment-rhs", name: "RESOURCE_ACCESS_SEGMENT_RHS"}
	PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_METHOD            = ParserRuleContext{value: "optional-resource-access-method", name: "OPTIONAL_RESOURCE_ACCESS_METHOD"}
	PARSER_RULE_CONTEXT_OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST   = ParserRuleContext{value: "optional-resource-method-call-arg-list", name: "OPTIONAL_RESOURCE_ACCESS_ACTION_ARG_LIST"}
	PARSER_RULE_CONTEXT_ACTION_END                                 = ParserRuleContext{value: "action-end", name: "ACTION_END"}
	PARSER_RULE_CONTEXT_OPTIONAL_PARENTHESIZED_ARG_LIST            = ParserRuleContext{value: "optional-parenthesized-arg-list", name: "OPTIONAL_PARENTHESIZED_ARG_LIST"}
	PARSER_RULE_CONTEXT_NATURAL_EXPRESSION                         = ParserRuleContext{value: "natural-expression", name: "NATURAL_EXPRESSION"}
	PARSER_RULE_CONTEXT_NATURAL_EXPRESSION_START                   = ParserRuleContext{value: "natural-expression-start", name: "NATURAL_EXPRESSION_START"}

	// Statements
	PARSER_RULE_CONTEXT_STATEMENT                      = ParserRuleContext{value: "statement", name: "STATEMENT"}
	PARSER_RULE_CONTEXT_STATEMENTS                     = ParserRuleContext{value: "statements", name: "STATEMENTS"}
	PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS       = ParserRuleContext{value: "statement-without-annots", name: "STATEMENT_WITHOUT_ANNOTS"}
	PARSER_RULE_CONTEXT_ASSIGNMENT_STMT                = ParserRuleContext{value: "assignment-stmt", name: "ASSIGNMENT_STMT"}
	PARSER_RULE_CONTEXT_VAR_DECL_STMT                  = ParserRuleContext{value: "var-decl-stmt", name: "VAR_DECL_STMT"}
	PARSER_RULE_CONTEXT_VAR_DECL_STMT_RHS              = ParserRuleContext{value: "var-decl-rhs", name: "VAR_DECL_STMT_RHS"}
	PARSER_RULE_CONTEXT_CONFIG_VAR_DECL_RHS            = ParserRuleContext{value: "config-var-decl-rhs", name: "CONFIG_VAR_DECL_RHS"}
	PARSER_RULE_CONTEXT_TYPE_NAME_OR_VAR_NAME          = ParserRuleContext{value: "type-or-var-name", name: "TYPE_NAME_OR_VAR_NAME"}
	PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT    = ParserRuleContext{value: "assign-or-var-decl", name: "ASSIGNMENT_OR_VAR_DECL_STMT"}
	PARSER_RULE_CONTEXT_IF_BLOCK                       = ParserRuleContext{value: "if-block", name: "IF_BLOCK"}
	PARSER_RULE_CONTEXT_BLOCK_STMT                     = ParserRuleContext{value: "block-stmt", name: "BLOCK_STMT"}
	PARSER_RULE_CONTEXT_ELSE_BLOCK                     = ParserRuleContext{value: "else-block", name: "ELSE_BLOCK"}
	PARSER_RULE_CONTEXT_ELSE_BODY                      = ParserRuleContext{value: "else-body", name: "ELSE_BODY"}
	PARSER_RULE_CONTEXT_WHILE_BLOCK                    = ParserRuleContext{value: "while-block", name: "WHILE_BLOCK"}
	PARSER_RULE_CONTEXT_DO_BLOCK                       = ParserRuleContext{value: "do-block", name: "DO_BLOCK"}
	PARSER_RULE_CONTEXT_CALL_STMT                      = ParserRuleContext{value: "call-statement", name: "CALL_STMT"}
	PARSER_RULE_CONTEXT_CALL_STMT_START                = ParserRuleContext{value: "call-statement-start", name: "CALL_STMT_START"}
	PARSER_RULE_CONTEXT_CONTINUE_STATEMENT             = ParserRuleContext{value: "continue-statement", name: "CONTINUE_STATEMENT"}
	PARSER_RULE_CONTEXT_BREAK_STATEMENT                = ParserRuleContext{value: "break-statement", name: "BREAK_STATEMENT"}
	PARSER_RULE_CONTEXT_PANIC_STMT                     = ParserRuleContext{value: "panic-statement", name: "PANIC_STMT"}
	PARSER_RULE_CONTEXT_RETURN_STMT                    = ParserRuleContext{value: "return-stmt", name: "RETURN_STMT"}
	PARSER_RULE_CONTEXT_RETURN_STMT_RHS                = ParserRuleContext{value: "return-stmt-rhs", name: "RETURN_STMT_RHS"}
	PARSER_RULE_CONTEXT_REGULAR_COMPOUND_STMT_RHS      = ParserRuleContext{value: "regular-compound-statement-rhs", name: "REGULAR_COMPOUND_STMT_RHS"}
	PARSER_RULE_CONTEXT_LOCAL_TYPE_DEFINITION_STMT     = ParserRuleContext{value: "local-type-definition-statement", name: "LOCAL_TYPE_DEFINITION_STMT"}
	PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_EXPR_RHS    = ParserRuleContext{value: "binding-pattern-or-expr-rhs", name: "BINDING_PATTERN_OR_EXPR_RHS"}
	PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS = ParserRuleContext{value: "binding.pattern.or.var.ref.rhs", name: "BINDING_PATTERN_OR_VAR_REF_RHS"}
	PARSER_RULE_CONTEXT_TYPE_DESC_OR_EXPR_RHS          = ParserRuleContext{value: "type-desc-or-expr-rhs", name: "TYPE_DESC_OR_EXPR_RHS"}
	PARSER_RULE_CONTEXT_STMT_START_WITH_EXPR_RHS       = ParserRuleContext{value: "stmt-start-with-expr-rhs", name: "STMT_START_WITH_EXPR_RHS"}
	PARSER_RULE_CONTEXT_EXPR_STMT_RHS                  = ParserRuleContext{value: "expr-stmt-rhs", name: "EXPR_STMT_RHS"}
	PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT           = ParserRuleContext{value: "expression-statement", name: "EXPRESSION_STATEMENT"}
	PARSER_RULE_CONTEXT_EXPRESSION_STATEMENT_START     = ParserRuleContext{value: "expression-statement-start", name: "EXPRESSION_STATEMENT_START"}
	PARSER_RULE_CONTEXT_LOCK_STMT                      = ParserRuleContext{value: "lock-stmt", name: "LOCK_STMT"}
	PARSER_RULE_CONTEXT_NAMED_WORKER_DECL              = ParserRuleContext{value: "named-worker-decl", name: "NAMED_WORKER_DECL"}
	PARSER_RULE_CONTEXT_NAMED_WORKER_DECL_START        = ParserRuleContext{value: "named-worker-decl-start", name: "NAMED_WORKER_DECL_START"}
	PARSER_RULE_CONTEXT_FORK_STMT                      = ParserRuleContext{value: "fork-stmt", name: "FORK_STMT"}
	PARSER_RULE_CONTEXT_FOREACH_STMT                   = ParserRuleContext{value: "foreach-stmt", name: "FOREACH_STMT"}
	PARSER_RULE_CONTEXT_TRANSACTION_STMT               = ParserRuleContext{value: "transaction-stmt", name: "TRANSACTION_STMT"}
	PARSER_RULE_CONTEXT_RETRY_STMT                     = ParserRuleContext{value: "retry-stmt", name: "RETRY_STMT"}
	PARSER_RULE_CONTEXT_ROLLBACK_STMT                  = ParserRuleContext{value: "rollback-stmt", name: "ROLLBACK_STMT"}
	PARSER_RULE_CONTEXT_AMBIGUOUS_STMT                 = ParserRuleContext{value: "ambiguous-stmt", name: "AMBIGUOUS_STMT"}
	PARSER_RULE_CONTEXT_MATCH_STMT                     = ParserRuleContext{value: "match-stmt", name: "MATCH_STMT"}
	PARSER_RULE_CONTEXT_FAIL_STATEMENT                 = ParserRuleContext{value: "fail-stmt", name: "FAIL_STATEMENT"}

	// Keywords
	PARSER_RULE_CONTEXT_RETURNS_KEYWORD       = ParserRuleContext{value: "returns", name: "RETURNS_KEYWORD"}
	PARSER_RULE_CONTEXT_TYPE_KEYWORD          = ParserRuleContext{value: "type", name: "TYPE_KEYWORD"}
	PARSER_RULE_CONTEXT_CLASS_KEYWORD         = ParserRuleContext{value: "class", name: "CLASS_KEYWORD"}
	PARSER_RULE_CONTEXT_PUBLIC_KEYWORD        = ParserRuleContext{value: "public", name: "PUBLIC_KEYWORD"}
	PARSER_RULE_CONTEXT_PRIVATE_KEYWORD       = ParserRuleContext{value: "private", name: "PRIVATE_KEYWORD"}
	PARSER_RULE_CONTEXT_FUNCTION_KEYWORD      = ParserRuleContext{value: "function", name: "FUNCTION_KEYWORD"}
	PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD      = ParserRuleContext{value: "external", name: "EXTERNAL_KEYWORD"}
	PARSER_RULE_CONTEXT_RECORD_KEYWORD        = ParserRuleContext{value: "record", name: "RECORD_KEYWORD"}
	PARSER_RULE_CONTEXT_OBJECT_KEYWORD        = ParserRuleContext{value: "object", name: "OBJECT_KEYWORD"}
	PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD      = ParserRuleContext{value: "abstract", name: "ABSTRACT_KEYWORD"}
	PARSER_RULE_CONTEXT_CLIENT_KEYWORD        = ParserRuleContext{value: "client", name: "CLIENT_KEYWORD"}
	PARSER_RULE_CONTEXT_IF_KEYWORD            = ParserRuleContext{value: "if", name: "IF_KEYWORD"}
	PARSER_RULE_CONTEXT_ELSE_KEYWORD          = ParserRuleContext{value: "else", name: "ELSE_KEYWORD"}
	PARSER_RULE_CONTEXT_WHILE_KEYWORD         = ParserRuleContext{value: "while", name: "WHILE_KEYWORD"}
	PARSER_RULE_CONTEXT_CONTINUE_KEYWORD      = ParserRuleContext{value: "continue", name: "CONTINUE_KEYWORD"}
	PARSER_RULE_CONTEXT_BREAK_KEYWORD         = ParserRuleContext{value: "break", name: "BREAK_KEYWORD"}
	PARSER_RULE_CONTEXT_PANIC_KEYWORD         = ParserRuleContext{value: "panic", name: "PANIC_KEYWORD"}
	PARSER_RULE_CONTEXT_IMPORT_KEYWORD        = ParserRuleContext{value: "import", name: "IMPORT_KEYWORD"}
	PARSER_RULE_CONTEXT_AS_KEYWORD            = ParserRuleContext{value: "as", name: "AS_KEYWORD"}
	PARSER_RULE_CONTEXT_RETURN_KEYWORD        = ParserRuleContext{value: "return", name: "RETURN_KEYWORD"}
	PARSER_RULE_CONTEXT_SERVICE_KEYWORD       = ParserRuleContext{value: "service", name: "SERVICE_KEYWORD"}
	PARSER_RULE_CONTEXT_ON_KEYWORD            = ParserRuleContext{value: "on", name: "ON_KEYWORD"}
	PARSER_RULE_CONTEXT_FINAL_KEYWORD         = ParserRuleContext{value: "final", name: "FINAL_KEYWORD"}
	PARSER_RULE_CONTEXT_LISTENER_KEYWORD      = ParserRuleContext{value: "listener", name: "LISTENER_KEYWORD"}
	PARSER_RULE_CONTEXT_CONST_KEYWORD         = ParserRuleContext{value: "const", name: "CONST_KEYWORD"}
	PARSER_RULE_CONTEXT_TYPEOF_KEYWORD        = ParserRuleContext{value: "typeof", name: "TYPEOF_KEYWORD"}
	PARSER_RULE_CONTEXT_IS_KEYWORD            = ParserRuleContext{value: "is", name: "IS_KEYWORD"}
	PARSER_RULE_CONTEXT_MAP_KEYWORD           = ParserRuleContext{value: "map", name: "MAP_KEYWORD"}
	PARSER_RULE_CONTEXT_NULL_KEYWORD          = ParserRuleContext{value: "null", name: "NULL_KEYWORD"}
	PARSER_RULE_CONTEXT_LOCK_KEYWORD          = ParserRuleContext{value: "lock", name: "LOCK_KEYWORD"}
	PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD    = ParserRuleContext{value: "annotation", name: "ANNOTATION_KEYWORD"}
	PARSER_RULE_CONTEXT_SOURCE_KEYWORD        = ParserRuleContext{value: "source", name: "SOURCE_KEYWORD"}
	PARSER_RULE_CONTEXT_XMLNS_KEYWORD         = ParserRuleContext{value: "xmlns", name: "XMLNS_KEYWORD"}
	PARSER_RULE_CONTEXT_WORKER_KEYWORD        = ParserRuleContext{value: "worker", name: "WORKER_KEYWORD"}
	PARSER_RULE_CONTEXT_FORK_KEYWORD          = ParserRuleContext{value: "fork", name: "FORK_KEYWORD"}
	PARSER_RULE_CONTEXT_TRAP_KEYWORD          = ParserRuleContext{value: "trap", name: "TRAP_KEYWORD"}
	PARSER_RULE_CONTEXT_IN_KEYWORD            = ParserRuleContext{value: "in", name: "IN_KEYWORD"}
	PARSER_RULE_CONTEXT_FOREACH_KEYWORD       = ParserRuleContext{value: "foreach", name: "FOREACH_KEYWORD"}
	PARSER_RULE_CONTEXT_TABLE_KEYWORD         = ParserRuleContext{value: "table", name: "TABLE_KEYWORD"}
	PARSER_RULE_CONTEXT_KEY_KEYWORD           = ParserRuleContext{value: "key", name: "KEY_KEYWORD"}
	PARSER_RULE_CONTEXT_ERROR_KEYWORD         = ParserRuleContext{value: "error", name: "ERROR_KEYWORD"}
	PARSER_RULE_CONTEXT_LET_KEYWORD           = ParserRuleContext{value: "let", name: "LET_KEYWORD"}
	PARSER_RULE_CONTEXT_STREAM_KEYWORD        = ParserRuleContext{value: "stream", name: "STREAM_KEYWORD"}
	PARSER_RULE_CONTEXT_XML_KEYWORD           = ParserRuleContext{value: "xml", name: "XML_KEYWORD"}
	PARSER_RULE_CONTEXT_STRING_KEYWORD        = ParserRuleContext{value: "string", name: "STRING_KEYWORD"}
	PARSER_RULE_CONTEXT_NEW_KEYWORD           = ParserRuleContext{value: "new", name: "NEW_KEYWORD"}
	PARSER_RULE_CONTEXT_FROM_KEYWORD          = ParserRuleContext{value: "from", name: "FROM_KEYWORD"}
	PARSER_RULE_CONTEXT_WHERE_KEYWORD         = ParserRuleContext{value: "where", name: "WHERE_KEYWORD"}
	PARSER_RULE_CONTEXT_SELECT_KEYWORD        = ParserRuleContext{value: "select", name: "SELECT_KEYWORD"}
	PARSER_RULE_CONTEXT_COLLECT_KEYWORD       = ParserRuleContext{value: "collect", name: "COLLECT_KEYWORD"}
	PARSER_RULE_CONTEXT_START_KEYWORD         = ParserRuleContext{value: "start", name: "START_KEYWORD"}
	PARSER_RULE_CONTEXT_FLUSH_KEYWORD         = ParserRuleContext{value: "flush", name: "FLUSH_KEYWORD"}
	PARSER_RULE_CONTEXT_WAIT_KEYWORD          = ParserRuleContext{value: "wait", name: "WAIT_KEYWORD"}
	PARSER_RULE_CONTEXT_DO_KEYWORD            = ParserRuleContext{value: "do", name: "DO_KEYWORD"}
	PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD   = ParserRuleContext{value: "transaction", name: "TRANSACTION_KEYWORD"}
	PARSER_RULE_CONTEXT_COMMIT_KEYWORD        = ParserRuleContext{value: "commit", name: "COMMIT_KEYWORD"}
	PARSER_RULE_CONTEXT_RETRY_KEYWORD         = ParserRuleContext{value: "retry", name: "RETRY_KEYWORD"}
	PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD      = ParserRuleContext{value: "rollback", name: "ROLLBACK_KEYWORD"}
	PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD = ParserRuleContext{value: "transactional", name: "TRANSACTIONAL_KEYWORD"}
	PARSER_RULE_CONTEXT_ENUM_KEYWORD          = ParserRuleContext{value: "enum", name: "ENUM_KEYWORD"}
	PARSER_RULE_CONTEXT_BASE16_KEYWORD        = ParserRuleContext{value: "base16", name: "BASE16_KEYWORD"}
	PARSER_RULE_CONTEXT_BASE64_KEYWORD        = ParserRuleContext{value: "base64", name: "BASE64_KEYWORD"}
	PARSER_RULE_CONTEXT_READONLY_KEYWORD      = ParserRuleContext{value: "readonly", name: "READONLY_KEYWORD"}
	PARSER_RULE_CONTEXT_MATCH_KEYWORD         = ParserRuleContext{value: "match", name: "MATCH_KEYWORD"}
	PARSER_RULE_CONTEXT_DISTINCT_KEYWORD      = ParserRuleContext{value: "distinct", name: "DISTINCT_KEYWORD"}
	PARSER_RULE_CONTEXT_CONFLICT_KEYWORD      = ParserRuleContext{value: "conflict", name: "CONFLICT_KEYWORD"}
	PARSER_RULE_CONTEXT_LIMIT_KEYWORD         = ParserRuleContext{value: "limit", name: "LIMIT_KEYWORD"}
	PARSER_RULE_CONTEXT_JOIN_KEYWORD          = ParserRuleContext{value: "join", name: "JOIN_KEYWORD"}
	PARSER_RULE_CONTEXT_OUTER_KEYWORD         = ParserRuleContext{value: "outer", name: "OUTER_KEYWORD"}
	PARSER_RULE_CONTEXT_VAR_KEYWORD           = ParserRuleContext{value: "var", name: "VAR_KEYWORD"}
	PARSER_RULE_CONTEXT_FAIL_KEYWORD          = ParserRuleContext{value: "fail", name: "FAIL_KEYWORD"}
	PARSER_RULE_CONTEXT_ORDER_KEYWORD         = ParserRuleContext{value: "order", name: "ORDER_KEYWORD"}
	PARSER_RULE_CONTEXT_BY_KEYWORD            = ParserRuleContext{value: "by", name: "BY_KEYWORD"}
	PARSER_RULE_CONTEXT_EQUALS_KEYWORD        = ParserRuleContext{value: "equals", name: "EQUALS_KEYWORD"}
	PARSER_RULE_CONTEXT_NOT_IS_KEYWORD        = ParserRuleContext{value: "!is", name: "NOT_IS_KEYWORD"}
	PARSER_RULE_CONTEXT_RE_KEYWORD            = ParserRuleContext{value: "re", name: "RE_KEYWORD"}
	PARSER_RULE_CONTEXT_GROUP_KEYWORD         = ParserRuleContext{value: "group", name: "GROUP_KEYWORD"}
	PARSER_RULE_CONTEXT_NATURAL_KEYWORD       = ParserRuleContext{value: "natural", name: "NATURAL_KEYWORD"}

	// Syntax tokens
	PARSER_RULE_CONTEXT_OPEN_PARENTHESIS                      = ParserRuleContext{value: "(", name: "OPEN_PARENTHESIS"}
	PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS                     = ParserRuleContext{value: ")", name: "CLOSE_PARENTHESIS"}
	PARSER_RULE_CONTEXT_OPEN_BRACE                            = ParserRuleContext{value: "{", name: "OPEN_BRACE"}
	PARSER_RULE_CONTEXT_CLOSE_BRACE                           = ParserRuleContext{value: "}", name: "CLOSE_BRACE"}
	PARSER_RULE_CONTEXT_ASSIGN_OP                             = ParserRuleContext{value: "=", name: "ASSIGN_OP"}
	PARSER_RULE_CONTEXT_SEMICOLON                             = ParserRuleContext{value: ";", name: "SEMICOLON"}
	PARSER_RULE_CONTEXT_COLON                                 = ParserRuleContext{value: ":", name: "COLON"}
	PARSER_RULE_CONTEXT_COMMA                                 = ParserRuleContext{value: "", name: "COMMA"}
	PARSER_RULE_CONTEXT_ELLIPSIS                              = ParserRuleContext{value: "...", name: "ELLIPSIS"}
	PARSER_RULE_CONTEXT_QUESTION_MARK                         = ParserRuleContext{value: "?", name: "QUESTION_MARK"}
	PARSER_RULE_CONTEXT_ASTERISK                              = ParserRuleContext{value: "*", name: "ASTERISK"}
	PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START              = ParserRuleContext{value: "{|", name: "CLOSED_RECORD_BODY_START"}
	PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END                = ParserRuleContext{value: "|}", name: "CLOSED_RECORD_BODY_END"}
	PARSER_RULE_CONTEXT_DOT                                   = ParserRuleContext{value: ".", name: "DOT"}
	PARSER_RULE_CONTEXT_OPEN_BRACKET                          = ParserRuleContext{value: "[", name: "OPEN_BRACKET"}
	PARSER_RULE_CONTEXT_CLOSE_BRACKET                         = ParserRuleContext{value: "]", name: "CLOSE_BRACKET"}
	PARSER_RULE_CONTEXT_SLASH                                 = ParserRuleContext{value: "/", name: "SLASH"}
	PARSER_RULE_CONTEXT_AT                                    = ParserRuleContext{value: "@", name: "AT"}
	PARSER_RULE_CONTEXT_RIGHT_ARROW                           = ParserRuleContext{value: "->", name: "RIGHT_ARROW"}
	PARSER_RULE_CONTEXT_GT                                    = ParserRuleContext{value: ">", name: "GT"}
	PARSER_RULE_CONTEXT_LT                                    = ParserRuleContext{value: "<", name: "LT"}
	PARSER_RULE_CONTEXT_PIPE                                  = ParserRuleContext{value: "|", name: "PIPE"}
	PARSER_RULE_CONTEXT_TEMPLATE_START                        = ParserRuleContext{value: "`", name: "TEMPLATE_START"}
	PARSER_RULE_CONTEXT_TEMPLATE_END                          = ParserRuleContext{value: "`", name: "TEMPLATE_END"}
	PARSER_RULE_CONTEXT_LT_TOKEN                              = ParserRuleContext{value: "<", name: "LT_TOKEN"}
	PARSER_RULE_CONTEXT_GT_TOKEN                              = ParserRuleContext{value: ">", name: "GT_TOKEN"}
	PARSER_RULE_CONTEXT_ERROR_TYPE_PARAM_START                = ParserRuleContext{value: "<", name: "ERROR_TYPE_PARAM_START"}
	PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START         = ParserRuleContext{value: "(", name: "PARENTHESISED_TYPE_DESC_START"}
	PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR                  = ParserRuleContext{value: "&", name: "BITWISE_AND_OPERATOR"}
	PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START                  = ParserRuleContext{value: "=>", name: "EXPR_FUNC_BODY_START"}
	PARSER_RULE_CONTEXT_PLUS_TOKEN                            = ParserRuleContext{value: "+", name: "PLUS_TOKEN"}
	PARSER_RULE_CONTEXT_MINUS_TOKEN                           = ParserRuleContext{value: "-", name: "MINUS_TOKEN"}
	PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START                 = ParserRuleContext{value: "[", name: "TUPLE_TYPE_DESC_START"}
	PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN                       = ParserRuleContext{value: "->>", name: "SYNC_SEND_TOKEN"}
	PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN                      = ParserRuleContext{value: "<-", name: "LEFT_ARROW_TOKEN"}
	PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN                  = ParserRuleContext{value: ".@", name: "ANNOT_CHAINING_TOKEN"}
	PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN               = ParserRuleContext{value: "?.", name: "OPTIONAL_CHAINING_TOKEN"}
	PARSER_RULE_CONTEXT_DOT_LT_TOKEN                          = ParserRuleContext{value: ".<", name: "DOT_LT_TOKEN"}
	PARSER_RULE_CONTEXT_SLASH_LT_TOKEN                        = ParserRuleContext{value: "/<", name: "SLASH_LT_TOKEN"}
	PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN = ParserRuleContext{value: "/**/<", name: "DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN"}
	PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN                  = ParserRuleContext{value: "/*", name: "SLASH_ASTERISK_TOKEN"}
	PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW                    = ParserRuleContext{value: "=>", name: "RIGHT_DOUBLE_ARROW"}
	PARSER_RULE_CONTEXT_DOUBLE_LT                             = ParserRuleContext{value: "<<", name: "DOUBLE_LT"}
	PARSER_RULE_CONTEXT_DOUBLE_EQUAL                          = ParserRuleContext{value: "==", name: "DOUBLE_EQUAL"}
	PARSER_RULE_CONTEXT_BITWISE_XOR                           = ParserRuleContext{value: "^", name: "BITWISE_XOR"}
	PARSER_RULE_CONTEXT_LOGICAL_AND                           = ParserRuleContext{value: "&&", name: "LOGICAL_AND"}
	PARSER_RULE_CONTEXT_LOGICAL_OR                            = ParserRuleContext{value: "||", name: "LOGICAL_OR"}
	PARSER_RULE_CONTEXT_ELVIS                                 = ParserRuleContext{value: "?:", name: "ELVIS"}

	// Other terminals
	PARSER_RULE_CONTEXT_FUNC_NAME                        = ParserRuleContext{value: "func-name", name: "FUNC_NAME"}
	PARSER_RULE_CONTEXT_VARIABLE_NAME                    = ParserRuleContext{value: "variable-name", name: "VARIABLE_NAME"}
	PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR           = ParserRuleContext{value: "simple-type-desc", name: "SIMPLE_TYPE_DESCRIPTOR"}
	PARSER_RULE_CONTEXT_BINARY_OPERATOR                  = ParserRuleContext{value: "binary-operator", name: "BINARY_OPERATOR"}
	PARSER_RULE_CONTEXT_TYPE_NAME                        = ParserRuleContext{value: "type-name", name: "TYPE_NAME"}
	PARSER_RULE_CONTEXT_CLASS_NAME                       = ParserRuleContext{value: "class-name", name: "CLASS_NAME"}
	PARSER_RULE_CONTEXT_BOOLEAN_LITERAL                  = ParserRuleContext{value: "boolean-literal", name: "BOOLEAN_LITERAL"}
	PARSER_RULE_CONTEXT_CHECKING_KEYWORD                 = ParserRuleContext{value: "checking-keyword", name: "CHECKING_KEYWORD"}
	PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR         = ParserRuleContext{value: "compound-binary-operator", name: "COMPOUND_BINARY_OPERATOR"}
	PARSER_RULE_CONTEXT_UNARY_OPERATOR                   = ParserRuleContext{value: "unary-operator", name: "UNARY_OPERATOR"}
	PARSER_RULE_CONTEXT_FUNCTION_IDENT                   = ParserRuleContext{value: "func-ident", name: "FUNCTION_IDENT"}
	PARSER_RULE_CONTEXT_FIELD_IDENT                      = ParserRuleContext{value: "field-ident", name: "FIELD_IDENT"}
	PARSER_RULE_CONTEXT_OBJECT_IDENT                     = ParserRuleContext{value: "object-ident", name: "OBJECT_IDENT"}
	PARSER_RULE_CONTEXT_SERVICE_IDENT                    = ParserRuleContext{value: "service-ident", name: "SERVICE_IDENT"}
	PARSER_RULE_CONTEXT_SERVICE_IDENT_RHS                = ParserRuleContext{value: "service-ident-rhs", name: "SERVICE_IDENT_RHS"}
	PARSER_RULE_CONTEXT_REMOTE_IDENT                     = ParserRuleContext{value: "remote-ident", name: "REMOTE_IDENT"}
	PARSER_RULE_CONTEXT_RECORD_IDENT                     = ParserRuleContext{value: "record-ident", name: "RECORD_IDENT"}
	PARSER_RULE_CONTEXT_ANNOTATION_TAG                   = ParserRuleContext{value: "annotation-tag", name: "ANNOTATION_TAG"}
	PARSER_RULE_CONTEXT_ATTACH_POINT_END                 = ParserRuleContext{value: "attach-point-end", name: "ATTACH_POINT_END"}
	PARSER_RULE_CONTEXT_IDENTIFIER                       = ParserRuleContext{value: "identifier", name: "IDENTIFIER"}
	PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT               = ParserRuleContext{value: "path-segment-ident", name: "PATH_SEGMENT_IDENT"}
	PARSER_RULE_CONTEXT_NAMESPACE_PREFIX                 = ParserRuleContext{value: "namespace-prefix", name: "NAMESPACE_PREFIX"}
	PARSER_RULE_CONTEXT_WORKER_NAME                      = ParserRuleContext{value: "worker-name", name: "WORKER_NAME"}
	PARSER_RULE_CONTEXT_FIELD_OR_FUNC_NAME               = ParserRuleContext{value: "field-or-func-name", name: "FIELD_OR_FUNC_NAME"}
	PARSER_RULE_CONTEXT_ORDER_DIRECTION                  = ParserRuleContext{value: "order-direction", name: "ORDER_DIRECTION"}
	PARSER_RULE_CONTEXT_VAR_REF_COLON                    = ParserRuleContext{value: "var-ref-colon", name: "VAR_REF_COLON"}
	PARSER_RULE_CONTEXT_TYPE_REF_COLON                   = ParserRuleContext{value: "type-ref-colon", name: "TYPE_REF_COLON"}
	PARSER_RULE_CONTEXT_METHOD_CALL_DOT                  = ParserRuleContext{value: "method-call-dot", name: "METHOD_CALL_DOT"}
	PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN = ParserRuleContext{value: "resource-method-call-slash-token", name: "RESOURCE_METHOD_CALL_SLASH_TOKEN"}

	// Expressions
	PARSER_RULE_CONTEXT_EXPRESSION                                             = ParserRuleContext{value: "expression", name: "EXPRESSION"}
	PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION                                    = ParserRuleContext{value: "terminal-expression", name: "TERMINAL_EXPRESSION"}
	PARSER_RULE_CONTEXT_EXPRESSION_RHS                                         = ParserRuleContext{value: "expression-rhs", name: "EXPRESSION_RHS"}
	PARSER_RULE_CONTEXT_FUNC_CALL                                              = ParserRuleContext{value: "func-call", name: "FUNC_CALL"}
	PARSER_RULE_CONTEXT_BASIC_LITERAL                                          = ParserRuleContext{value: "basic-literal", name: "BASIC_LITERAL"}
	PARSER_RULE_CONTEXT_ACCESS_EXPRESSION                                      = ParserRuleContext{value: "access-expr", name: "ACCESS_EXPRESSION"} // method-call, field-access, member-access
	PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN                          = ParserRuleContext{value: "decimal-int-literal-token", name: "DECIMAL_INTEGER_LITERAL_TOKEN"}
	PARSER_RULE_CONTEXT_VARIABLE_REF                                           = ParserRuleContext{value: "var-ref", name: "VARIABLE_REF"}
	PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN                                   = ParserRuleContext{value: "string-literal-token", name: "STRING_LITERAL_TOKEN"}
	PARSER_RULE_CONTEXT_MAPPING_CONSTRUCTOR                                    = ParserRuleContext{value: "mapping-constructor", name: "MAPPING_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_MAPPING_FIELD                                          = ParserRuleContext{value: "maping-field", name: "MAPPING_FIELD"}
	PARSER_RULE_CONTEXT_FIRST_MAPPING_FIELD                                    = ParserRuleContext{value: "first-mapping-field", name: "FIRST_MAPPING_FIELD"}
	PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME                                     = ParserRuleContext{value: "maping-field-name", name: "MAPPING_FIELD_NAME"}
	PARSER_RULE_CONTEXT_SPECIFIC_FIELD_RHS                                     = ParserRuleContext{value: "specific-field-rhs", name: "SPECIFIC_FIELD_RHS"}
	PARSER_RULE_CONTEXT_SPECIFIC_FIELD                                         = ParserRuleContext{value: "specific-field", name: "SPECIFIC_FIELD"}
	PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME                                    = ParserRuleContext{value: "computed-field-name", name: "COMPUTED_FIELD_NAME"}
	PARSER_RULE_CONTEXT_MAPPING_FIELD_END                                      = ParserRuleContext{value: "mapping-field-end", name: "MAPPING_FIELD_END"}
	PARSER_RULE_CONTEXT_TYPEOF_EXPRESSION                                      = ParserRuleContext{value: "typeof-expr", name: "TYPEOF_EXPRESSION"}
	PARSER_RULE_CONTEXT_UNARY_EXPRESSION                                       = ParserRuleContext{value: "unary-expr", name: "UNARY_EXPRESSION"}
	PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN                              = ParserRuleContext{value: "hex-integer-literal-token", name: "HEX_INTEGER_LITERAL_TOKEN"}
	PARSER_RULE_CONTEXT_NIL_LITERAL                                            = ParserRuleContext{value: "nil-literal", name: "NIL_LITERAL"}
	PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION                                    = ParserRuleContext{value: "constant-expr", name: "CONSTANT_EXPRESSION"}
	PARSER_RULE_CONTEXT_CONSTANT_EXPRESSION_START                              = ParserRuleContext{value: "constant-expr-start", name: "CONSTANT_EXPRESSION_START"}
	PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN                   = ParserRuleContext{value: "decimal-floating-point-literal-token", name: "DECIMAL_FLOATING_POINT_LITERAL_TOKEN"}
	PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN                       = ParserRuleContext{value: "hex-floating-point-literal-token", name: "HEX_FLOATING_POINT_LITERAL_TOKEN"}
	PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR                                       = ParserRuleContext{value: "list-constructor", name: "LIST_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_FIRST_MEMBER                          = ParserRuleContext{value: "list-constructor-first-member", name: "LIST_CONSTRUCTOR_FIRST_MEMBER"}
	PARSER_RULE_CONTEXT_LIST_CONSTRUCTOR_MEMBER                                = ParserRuleContext{value: "list-constructor-member", name: "LIST_CONSTRUCTOR_MEMBER"}
	PARSER_RULE_CONTEXT_TYPE_CAST                                              = ParserRuleContext{value: "type-cast", name: "TYPE_CAST"}
	PARSER_RULE_CONTEXT_TYPE_CAST_PARAM                                        = ParserRuleContext{value: "type-cast-param", name: "TYPE_CAST_PARAM"}
	PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_RHS                                    = ParserRuleContext{value: "type-cast-param-rhs", name: "TYPE_CAST_PARAM_RHS"}
	PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START                                  = ParserRuleContext{value: "type-cast-param-start", name: "TYPE_CAST_PARAM_START"}
	PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR                                      = ParserRuleContext{value: "table-constructor", name: "TABLE_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_TABLE_KEYWORD_RHS                                      = ParserRuleContext{value: "table-keyword-rhs", name: "TABLE_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_ROW_LIST_RHS                                           = ParserRuleContext{value: "row-list-rhs", name: "ROW_LIST_RHS"}
	PARSER_RULE_CONTEXT_TABLE_ROW_END                                          = ParserRuleContext{value: "table-row-end", name: "TABLE_ROW_END"}
	PARSER_RULE_CONTEXT_NEW_KEYWORD_RHS                                        = ParserRuleContext{value: "new-keyword-rhs", name: "NEW_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_IMPLICIT_NEW                                           = ParserRuleContext{value: "implicit-new", name: "IMPLICIT_NEW"}
	PARSER_RULE_CONTEXT_CLASS_DESCRIPTOR_IN_NEW_EXPR                           = ParserRuleContext{value: "class-descriptor-in-new-expr", name: "CLASS_DESCRIPTOR_IN_NEW_EXPR"}
	PARSER_RULE_CONTEXT_LET_EXPRESSION                                         = ParserRuleContext{value: "let-expr", name: "LET_EXPRESSION"}
	PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION                                   = ParserRuleContext{value: "anon-func-expression", name: "ANON_FUNC_EXPRESSION"}
	PARSER_RULE_CONTEXT_ANON_FUNC_EXPRESSION_START                             = ParserRuleContext{value: "anon-func-expression-start", name: "ANON_FUNC_EXPRESSION_START"}
	PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION                  = ParserRuleContext{value: "table-constructor-or-query-expr", name: "TABLE_CONSTRUCTOR_OR_QUERY_EXPRESSION"}
	PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_START                       = ParserRuleContext{value: "table-constructor-or-query-start", name: "TABLE_CONSTRUCTOR_OR_QUERY_START"}
	PARSER_RULE_CONTEXT_TABLE_CONSTRUCTOR_OR_QUERY_RHS                         = ParserRuleContext{value: "table-constructor-or-query-rhs", name: "TABLE_CONSTRUCTOR_OR_QUERY_RHS"}
	PARSER_RULE_CONTEXT_QUERY_EXPRESSION                                       = ParserRuleContext{value: "query-expr", name: "QUERY_EXPRESSION"}
	PARSER_RULE_CONTEXT_QUERY_EXPRESSION_RHS                                   = ParserRuleContext{value: "query-expr-rhs", name: "QUERY_EXPRESSION_RHS"}
	PARSER_RULE_CONTEXT_QUERY_ACTION_RHS                                       = ParserRuleContext{value: "query-action-rhs", name: "QUERY_ACTION_RHS"}
	PARSER_RULE_CONTEXT_QUERY_EXPRESSION_END                                   = ParserRuleContext{value: "query-expr-end", name: "QUERY_EXPRESSION_END"}
	PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER                                = ParserRuleContext{value: "field-access-identifier", name: "FIELD_ACCESS_IDENTIFIER"}
	PARSER_RULE_CONTEXT_QUERY_PIPELINE_RHS                                     = ParserRuleContext{value: "query-pipeline-rhs", name: "QUERY_PIPELINE_RHS"}
	PARSER_RULE_CONTEXT_LET_CLAUSE_END                                         = ParserRuleContext{value: "let-clause-end", name: "LET_CLAUSE_END"}
	PARSER_RULE_CONTEXT_CONDITIONAL_EXPRESSION                                 = ParserRuleContext{value: "conditional-expr", name: "CONDITIONAL_EXPRESSION"}
	PARSER_RULE_CONTEXT_XML_NAVIGATE_EXPR                                      = ParserRuleContext{value: "xml-navigate-expr", name: "XML_NAVIGATE_EXPR"}
	PARSER_RULE_CONTEXT_XML_FILTER_EXPR                                        = ParserRuleContext{value: "xml-filter-expr", name: "XML_FILTER_EXPR"}
	PARSER_RULE_CONTEXT_XML_STEP_EXPR                                          = ParserRuleContext{value: "xml-step-expr", name: "XML_STEP_EXPR"}
	PARSER_RULE_CONTEXT_XML_NAME_PATTERN                                       = ParserRuleContext{value: "xml-name-pattern", name: "XML_NAME_PATTERN"}
	PARSER_RULE_CONTEXT_XML_NAME_PATTERN_RHS                                   = ParserRuleContext{value: "xml-name-pattern-rhs", name: "XML_NAME_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN                                = ParserRuleContext{value: "xml-atomic_name-pattern", name: "XML_ATOMIC_NAME_PATTERN"}
	PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_PATTERN_START                          = ParserRuleContext{value: "xml-atomic_name-pattern-start", name: "XML_ATOMIC_NAME_PATTERN_START"}
	PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER                             = ParserRuleContext{value: "xml-atomic_name-identifier", name: "XML_ATOMIC_NAME_IDENTIFIER"}
	PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER_RHS                         = ParserRuleContext{value: "xml-atomic_name-identifier-rhs", name: "XML_ATOMIC_NAME_IDENTIFIER_RHS"}
	PARSER_RULE_CONTEXT_XML_STEP_START                                         = ParserRuleContext{value: "xml-step-start", name: "XML_STEP_START"}
	PARSER_RULE_CONTEXT_VARIABLE_REF_RHS                                       = ParserRuleContext{value: "variable-ref-rhs", name: "VARIABLE_REF_RHS"}
	PARSER_RULE_CONTEXT_ORDER_CLAUSE_END                                       = ParserRuleContext{value: "order-clause-end", name: "ORDER_CLAUSE_END"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR                                     = ParserRuleContext{value: "object-constructor", name: "OBJECT_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_OBJECT_CONSTRUCTOR_TYPE_REF                            = ParserRuleContext{value: "object-constructor-type-ref", name: "OBJECT_CONSTRUCTOR_TYPE_REF"}
	PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR                                      = ParserRuleContext{value: "error-constructor", name: "ERROR_CONSTRUCTOR"}
	PARSER_RULE_CONTEXT_ERROR_CONSTRUCTOR_RHS                                  = ParserRuleContext{value: "error-constructor-rhs", name: "ERROR_CONSTRUCTOR_RHS"}
	PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT                     = ParserRuleContext{value: "inferred-typedesc-default-start-lt", name: "INFERRED_TYPEDESC_DEFAULT_START_LT"}
	PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT                       = ParserRuleContext{value: "inferred-typedesc-default-end-gt", name: "INFERRED_TYPEDESC_DEFAULT_END_GT"}
	PARSER_RULE_CONTEXT_EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START          = ParserRuleContext{value: "expr-start-or-inferred-typedesc-default-start", name: "EXPR_START_OR_INFERRED_TYPEDESC_DEFAULT_START"}
	PARSER_RULE_CONTEXT_TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END = ParserRuleContext{value: "type-cast-param-start-or-inferred-typedesc-default-end", name: "TYPE_CAST_PARAM_START_OR_INFERRED_TYPEDESC_DEFAULT_END"}
	PARSER_RULE_CONTEXT_END_OF_PARAMS_OR_NEXT_PARAM_START                      = ParserRuleContext{value: "end-of-params-or-next-param-start", name: "END_OF_PARAMS_OR_NEXT_PARAM_START"}
	PARSER_RULE_CONTEXT_BRACED_EXPRESSION                                      = ParserRuleContext{value: "braced-expression", name: "BRACED_EXPRESSION"}
	PARSER_RULE_CONTEXT_ACTION                                                 = ParserRuleContext{value: "action", name: "ACTION"}

	// Contexts that expect a type
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANNOTATION_DECL                = ParserRuleContext{value: "type-desc-annotation-descl", name: "TYPE_DESC_IN_ANNOTATION_DECL"}
	PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER                 = ParserRuleContext{value: "type-desc-before-identifier", name: "TYPE_DESC_BEFORE_IDENTIFIER"} // object/record fields, params, const, listener
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_RECORD_FIELD                   = ParserRuleContext{value: "type-desc-in-record-field", name: "TYPE_DESC_IN_RECORD_FIELD"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARAM                          = ParserRuleContext{value: "type-desc-in-param", name: "TYPE_DESC_IN_PARAM"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_BINDING_PATTERN           = ParserRuleContext{value: "type-desc-in-type-binding-pattern", name: "TYPE_DESC_IN_TYPE_BINDING_PATTERN"} // foreach, let-var-decl, var-decl
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_TYPE_DEF                       = ParserRuleContext{value: "type-def-type-desc", name: "TYPE_DESC_IN_TYPE_DEF"}                            // local/mdule type defitions
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_ANGLE_BRACKETS                 = ParserRuleContext{value: "type-desc-in-angle-bracket", name: "TYPE_DESC_IN_ANGLE_BRACKETS"}              // type-cast, parameterized-type
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_RETURN_TYPE_DESC               = ParserRuleContext{value: "type-desc-in-return-type-desc", name: "TYPE_DESC_IN_RETURN_TYPE_DESC"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_EXPRESSION                     = ParserRuleContext{value: "type-desc-in-expression", name: "TYPE_DESC_IN_EXPRESSION"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_STREAM_TYPE_DESC               = ParserRuleContext{value: "type-desc-in-stream-type-desc", name: "TYPE_DESC_IN_STREAM_TYPE_DESC"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_TUPLE                          = ParserRuleContext{value: "type-desc-in-tuple", name: "TYPE_DESC_IN_TUPLE"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_PARENTHESIS                    = ParserRuleContext{value: "type-desc-in-parenthesis", name: "TYPE_DESC_IN_PARENTHESIS"}
	PARSER_RULE_CONTEXT_VAR_DECL_STARTED_WITH_DENTIFIER             = ParserRuleContext{value: "var-decl-started-with-dentifier", name: "VAR_DECL_STARTED_WITH_DENTIFIER"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_SERVICE                        = ParserRuleContext{value: "type-desc-in-service", name: "TYPE_DESC_IN_SERVICE"}
	PARSER_RULE_CONTEXT_TYPE_DESC_IN_PATH_PARAM                     = ParserRuleContext{value: "type-desc-in-path-param", name: "TYPE_DESC_IN_PATH_PARAM"}
	PARSER_RULE_CONTEXT_TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY = ParserRuleContext{value: "type-desc-before-identifier-in-grouping-key", name: "TYPE_DESC_BEFORE_IDENTIFIER_IN_GROUPING_KEY"}

	// XML
	PARSER_RULE_CONTEXT_XML_CONTENT                = ParserRuleContext{value: "xml-content", name: "XML_CONTENT"}
	PARSER_RULE_CONTEXT_XML_TAG                    = ParserRuleContext{value: "xml-tag", name: "XML_TAG"}
	PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG     = ParserRuleContext{value: "xml-start-or-empty-tag", name: "XML_START_OR_EMPTY_TAG"}
	PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END = ParserRuleContext{value: "xml-start-or-empty-tag-end", name: "XML_START_OR_EMPTY_TAG_END"}
	PARSER_RULE_CONTEXT_XML_END_TAG                = ParserRuleContext{value: "xml-end-tag", name: "XML_END_TAG"}
	PARSER_RULE_CONTEXT_XML_NAME                   = ParserRuleContext{value: "xml-name", name: "XML_NAME"}
	PARSER_RULE_CONTEXT_XML_PI                     = ParserRuleContext{value: "xml-pi", name: "XML_PI"}
	PARSER_RULE_CONTEXT_XML_TEXT                   = ParserRuleContext{value: "xml-text", name: "XML_TEXT"}
	PARSER_RULE_CONTEXT_XML_ATTRIBUTES             = ParserRuleContext{value: "xml-attributes", name: "XML_ATTRIBUTES"}
	PARSER_RULE_CONTEXT_XML_ATTRIBUTE              = ParserRuleContext{value: "xml-attribute", name: "XML_ATTRIBUTE"}
	PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_ITEM   = ParserRuleContext{value: "xml-attribute-value-item", name: "XML_ATTRIBUTE_VALUE_ITEM"}
	PARSER_RULE_CONTEXT_XML_ATTRIBUTE_VALUE_TEXT   = ParserRuleContext{value: "xml-attribute-value-text", name: "XML_ATTRIBUTE_VALUE_TEXT"}
	PARSER_RULE_CONTEXT_XML_COMMENT_START          = ParserRuleContext{value: "<!--", name: "XML_COMMENT_START"}
	PARSER_RULE_CONTEXT_XML_COMMENT_END            = ParserRuleContext{value: "-->", name: "XML_COMMENT_END"}
	PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT        = ParserRuleContext{value: "xml-comment-content", name: "XML_COMMENT_CONTENT"}
	PARSER_RULE_CONTEXT_XML_PI_START               = ParserRuleContext{value: "<?", name: "XML_PI_START"}
	PARSER_RULE_CONTEXT_XML_PI_END                 = ParserRuleContext{value: "?>", name: "XML_PI_END"}
	PARSER_RULE_CONTEXT_XML_PI_DATA                = ParserRuleContext{value: "xml-pi-data", name: "XML_PI_DATA"}
	PARSER_RULE_CONTEXT_XML_PI_TARGET_RHS          = ParserRuleContext{value: "xml-pi-target-rhs", name: "XML_PI_TARGET_RHS"}
	PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN  = ParserRuleContext{value: "${", name: "INTERPOLATION_START_TOKEN"}
	PARSER_RULE_CONTEXT_INTERPOLATION              = ParserRuleContext{value: "interoplation", name: "INTERPOLATION"}
	PARSER_RULE_CONTEXT_TEMPLATE_BODY              = ParserRuleContext{value: "template-body", name: "TEMPLATE_BODY"}
	PARSER_RULE_CONTEXT_TEMPLATE_MEMBER            = ParserRuleContext{value: "template-member", name: "TEMPLATE_MEMBER"}
	PARSER_RULE_CONTEXT_TEMPLATE_STRING            = ParserRuleContext{value: "template-string", name: "TEMPLATE_STRING"}
	PARSER_RULE_CONTEXT_TEMPLATE_STRING_RHS        = ParserRuleContext{value: "template-string-rhs", name: "TEMPLATE_STRING_RHS"}
	PARSER_RULE_CONTEXT_XML_QUOTE_START            = ParserRuleContext{value: "xml-quote-start", name: "XML_QUOTE_START"}
	PARSER_RULE_CONTEXT_XML_QUOTE_END              = ParserRuleContext{value: "xml-quote-end", name: "XML_QUOTE_END"}
	PARSER_RULE_CONTEXT_XML_CDATA_START            = ParserRuleContext{value: "xml-cdata-start", name: "XML_CDATA_START"}
	PARSER_RULE_CONTEXT_XML_OPTIONAL_CDATA_CONTENT = ParserRuleContext{value: "xml-optional-cdata-content", name: "XML_OPTIONAL_CDATA_CONTENT"}
	PARSER_RULE_CONTEXT_XML_CDATA_CONTENT          = ParserRuleContext{value: "xml-cdata-content", name: "XML_CDATA_CONTENT"}
	PARSER_RULE_CONTEXT_XML_CDATA_END              = ParserRuleContext{value: "xml-cdata-end", name: "XML_CDATA_END"}

	// Other
	PARSER_RULE_CONTEXT_TYPE_DESC_RHS                            = ParserRuleContext{value: "type-desc-rhs", name: "TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS               = ParserRuleContext{value: "func-type-func-keyword-rhs", name: "FUNC_TYPE_FUNC_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_FUNC_TYPE_FUNC_KEYWORD_RHS_START         = ParserRuleContext{value: "func-type-func-keyword-rhs-start", name: "FUNC_TYPE_FUNC_KEYWORD_RHS_START"}
	PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN            = ParserRuleContext{value: "stream-type-param-start-token", name: "STREAM_TYPE_PARAM_START_TOKEN"}
	PARSER_RULE_CONTEXT_STREAM_TYPE_FIRST_PARAM_RHS              = ParserRuleContext{value: "stream-type-params", name: "STREAM_TYPE_FIRST_PARAM_RHS"}
	PARSER_RULE_CONTEXT_KEY_CONSTRAINTS_RHS                      = ParserRuleContext{value: "key-constraints-rhs", name: "KEY_CONSTRAINTS_RHS"}
	PARSER_RULE_CONTEXT_ROW_TYPE_PARAM                           = ParserRuleContext{value: "row-type-param", name: "ROW_TYPE_PARAM"}
	PARSER_RULE_CONTEXT_TABLE_TYPE_DESC_RHS                      = ParserRuleContext{value: "table-type-desc-rhs", name: "TABLE_TYPE_DESC_RHS"}
	PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS                  = ParserRuleContext{value: "signed-int-or-float-rhs", name: "SIGNED_INT_OR_FLOAT_RHS"}
	PARSER_RULE_CONTEXT_ENUM_MEMBER_LIST                         = ParserRuleContext{value: "enum-member-list", name: "ENUM_MEMBER_LIST"}
	PARSER_RULE_CONTEXT_ENUM_MEMBER_END                          = ParserRuleContext{value: "enum-member-rhs", name: "ENUM_MEMBER_END"}
	PARSER_RULE_CONTEXT_ENUM_MEMBER_RHS                          = ParserRuleContext{value: "enum-member-internal-rhs", name: "ENUM_MEMBER_RHS"}
	PARSER_RULE_CONTEXT_ENUM_MEMBER_START                        = ParserRuleContext{value: "enum-member-start", name: "ENUM_MEMBER_START"}
	PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER     = ParserRuleContext{value: "tuple-type-desc-or-list-cont-member", name: "TUPLE_TYPE_DESC_OR_LIST_CONST_MEMBER"}
	PARSER_RULE_CONTEXT_MAP_TYPE_OR_TYPE_REF                     = ParserRuleContext{value: "map-type-or-type-ref", name: "MAP_TYPE_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_OR_TYPE_REF                  = ParserRuleContext{value: "object-type-or-type-ref", name: "OBJECT_TYPE_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_STREAM_TYPE_OR_TYPE_REF                  = ParserRuleContext{value: "stream-type-or-type-ref", name: "STREAM_TYPE_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_TABLE_TYPE_OR_TYPE_REF                   = ParserRuleContext{value: "table-type-or-type-ref", name: "TABLE_TYPE_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE_OR_TYPE_REF           = ParserRuleContext{value: "parameterized-type-or-type-ref", name: "PARAMETERIZED_TYPE_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_TYPE_REF                = ParserRuleContext{value: "type-desc-rhs-or-type-ref", name: "TYPE_DESC_RHS_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_OBJECT_TYPE_OBJECT_KEYWORD_RHS           = ParserRuleContext{value: "object-type-object-keyword-rhs", name: "OBJECT_TYPE_OBJECT_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF      = ParserRuleContext{value: "table-cons-or-query-expr-or-var-ref", name: "TABLE_CONS_OR_QUERY_EXPR_OR_VAR_REF"}
	PARSER_RULE_CONTEXT_EXPRESSION_START_TABLE_KEYWORD_RHS       = ParserRuleContext{value: "expression-start-table-keyword-rhs", name: "EXPRESSION_START_TABLE_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_QUERY_EXPR_OR_VAR_REF                    = ParserRuleContext{value: "query-expr-or-var-ref", name: "QUERY_EXPR_OR_VAR_REF"}
	PARSER_RULE_CONTEXT_QUERY_CONSTRUCT_TYPE_RHS                 = ParserRuleContext{value: "query-construct-type-rhs", name: "QUERY_CONSTRUCT_TYPE_RHS"}
	PARSER_RULE_CONTEXT_ERROR_CONS_EXPR_OR_VAR_REF               = ParserRuleContext{value: "error-cons-expr-or-var-ref", name: "ERROR_CONS_EXPR_OR_VAR_REF"}
	PARSER_RULE_CONTEXT_ERROR_CONS_ERROR_KEYWORD_RHS             = ParserRuleContext{value: "error-cons-error-keyword-rhs", name: "ERROR_CONS_ERROR_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS = ParserRuleContext{value: "transaction-stmt-transaction-keyword-rhs", name: "TRANSACTION_STMT_TRANSACTION_KEYWORD_RHS"}
	PARSER_RULE_CONTEXT_TRANSACTION_STMT_RHS_OR_TYPE_REF         = ParserRuleContext{value: "transaction-stmt-rhs-or-type-ref", name: "TRANSACTION_STMT_RHS_OR_TYPE_REF"}
	PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER    = ParserRuleContext{value: "qualified-identifier-start-identifier", name: "QUALIFIED_IDENTIFIER_START_IDENTIFIER"}
	PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX  = ParserRuleContext{value: "qualified-identifier-predeclared-prefix", name: "QUALIFIED_IDENTIFIER_PREDECLARED_PREFIX"}
	PARSER_RULE_CONTEXT_TYPE_DESC_RHS_OR_BP_RHS                  = ParserRuleContext{value: "type-desc-rhs-or-binding-pattern-rhs", name: "TYPE_DESC_RHS_OR_BP_RHS"}
	PARSER_RULE_CONTEXT_LIST_BINDING_PATTERN_RHS                 = ParserRuleContext{value: "list-binding-pattern-rhs", name: "LIST_BINDING_PATTERN_RHS"}
	PARSER_RULE_CONTEXT_TYPE_DESC_RHS_IN_TYPED_BP                = ParserRuleContext{value: "type-desc-rhs-in-typed-binding-pattern", name: "TYPE_DESC_RHS_IN_TYPED_BP"}
	PARSER_RULE_CONTEXT_ASSIGNMENT_STMT_RHS                      = ParserRuleContext{value: "assignment-stmt-rhs", name: "ASSIGNMENT_STMT_RHS"}
	PARSER_RULE_CONTEXT_ANNOTATION_DECL_START                    = ParserRuleContext{value: "annotation-declaration-start", name: "ANNOTATION_DECL_START"}
	PARSER_RULE_CONTEXT_OPTIONAL_TOP_LEVEL_SEMICOLON             = ParserRuleContext{value: "optional-top-level-semicolon", name: "OPTIONAL_TOP_LEVEL_SEMICOLON"}
	PARSER_RULE_CONTEXT_TUPLE_MEMBERS                            = ParserRuleContext{value: "tuple-members", name: "TUPLE_MEMBERS"}
	PARSER_RULE_CONTEXT_TUPLE_MEMBER                             = ParserRuleContext{value: "tuple-member", name: "TUPLE_MEMBER"}
	PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER               = ParserRuleContext{value: "single-or-alternate-worker", name: "SINGLE_OR_ALTERNATE_WORKER"}
	PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_SEPARATOR     = ParserRuleContext{value: "single-or-alternate-worker-separator", name: "SINGLE_OR_ALTERNATE_WORKER_SEPARATOR"}
	PARSER_RULE_CONTEXT_SINGLE_OR_ALTERNATE_WORKER_END           = ParserRuleContext{value: "single-or-alternate-worker-end", name: "SINGLE_OR_ALTERNATE_WORKER_END"}
	PARSER_RULE_CONTEXT_XML_STEP_EXTENDS                         = ParserRuleContext{value: "xml-step-extends", name: "XML_STEP_EXTENDS"}
	PARSER_RULE_CONTEXT_XML_STEP_EXTEND                          = ParserRuleContext{value: "xml-step-extend", name: "XML_STEP_EXTEND"}
	PARSER_RULE_CONTEXT_XML_STEP_EXTEND_END                      = ParserRuleContext{value: "xml-step-extend-end", name: "XML_STEP_EXTEND_END"}
	PARSER_RULE_CONTEXT_XML_STEP_START_END                       = ParserRuleContext{value: "xml-step-start-end", name: "XML_STEP_START_END"}
)

func (p ParserRuleContext) GetErrorCode() diagnostics.DiagnosticCode {
	switch p {
	case PARSER_RULE_CONTEXT_EXTERNAL_FUNC_BODY:
		return &ERROR_MISSING_EQUAL_TOKEN
	case PARSER_RULE_CONTEXT_FUNC_BODY_BLOCK:
		return &ERROR_MISSING_OPEN_BRACE_TOKEN
	case PARSER_RULE_CONTEXT_FUNC_DEF,
		PARSER_RULE_CONTEXT_FUNC_DEF_OR_FUNC_TYPE,
		PARSER_RULE_CONTEXT_FUNC_TYPE_DESC,
		PARSER_RULE_CONTEXT_FUNC_TYPE_DESC_OR_ANON_FUNC,
		PARSER_RULE_CONTEXT_IDENT_AFTER_OBJECT_IDENT,
		PARSER_RULE_CONTEXT_FUNC_DEF_FIRST_QUALIFIER,
		PARSER_RULE_CONTEXT_FUNC_DEF_SECOND_QUALIFIER,
		PARSER_RULE_CONTEXT_FUNC_TYPE_FIRST_QUALIFIER,
		PARSER_RULE_CONTEXT_FUNC_TYPE_SECOND_QUALIFIER,
		PARSER_RULE_CONTEXT_OBJECT_METHOD_FIRST_QUALIFIER,
		PARSER_RULE_CONTEXT_OBJECT_METHOD_SECOND_QUALIFIER,
		PARSER_RULE_CONTEXT_OBJECT_METHOD_THIRD_QUALIFIER,
		PARSER_RULE_CONTEXT_OBJECT_METHOD_FOURTH_QUALIFIER:
		return &ERROR_MISSING_FUNCTION_KEYWORD
	case PARSER_RULE_CONTEXT_SINGLE_KEYWORD_ATTACH_POINT_IDENT:
		return &ERROR_MISSING_ATTACH_POINT_NAME
	case PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESCRIPTOR:
		return &ERROR_MISSING_BUILTIN_TYPE
	case PARSER_RULE_CONTEXT_REQUIRED_PARAM,
		PARSER_RULE_CONTEXT_VAR_DECL_STMT,
		PARSER_RULE_CONTEXT_ASSIGNMENT_OR_VAR_DECL_STMT,
		PARSER_RULE_CONTEXT_DEFAULTABLE_PARAM,
		PARSER_RULE_CONTEXT_REST_PARAM,
		PARSER_RULE_CONTEXT_TYPE_DESCRIPTOR,
		PARSER_RULE_CONTEXT_OPTIONAL_TYPE_DESCRIPTOR,
		PARSER_RULE_CONTEXT_ARRAY_TYPE_DESCRIPTOR,
		PARSER_RULE_CONTEXT_SIMPLE_TYPE_DESC_IDENTIFIER:
		return &ERROR_MISSING_TYPE_DESC
	case PARSER_RULE_CONTEXT_TYPE_REFERENCE:
		return &ERROR_MISSING_TYPE_REFERENCE
	case PARSER_RULE_CONTEXT_TYPE_NAME,
		PARSER_RULE_CONTEXT_TYPE_REFERENCE_IN_TYPE_INCLUSION,
		PARSER_RULE_CONTEXT_FIELD_ACCESS_IDENTIFIER,
		PARSER_RULE_CONTEXT_CLASS_NAME,
		PARSER_RULE_CONTEXT_FUNC_NAME,
		PARSER_RULE_CONTEXT_VARIABLE_NAME,
		PARSER_RULE_CONTEXT_IMPORT_MODULE_NAME,
		PARSER_RULE_CONTEXT_IMPORT_ORG_OR_MODULE_NAME,
		PARSER_RULE_CONTEXT_IMPORT_PREFIX,
		PARSER_RULE_CONTEXT_VARIABLE_REF,
		PARSER_RULE_CONTEXT_BASIC_LITERAL,
		PARSER_RULE_CONTEXT_IDENTIFIER,
		PARSER_RULE_CONTEXT_QUALIFIED_IDENTIFIER_START_IDENTIFIER,
		PARSER_RULE_CONTEXT_NAMESPACE_PREFIX,
		PARSER_RULE_CONTEXT_IMPLICIT_ANON_FUNC_PARAM,
		PARSER_RULE_CONTEXT_METHOD_NAME,
		PARSER_RULE_CONTEXT_PEER_WORKER_NAME,
		PARSER_RULE_CONTEXT_RECEIVE_FIELD_NAME,
		PARSER_RULE_CONTEXT_WAIT_FIELD_NAME,
		PARSER_RULE_CONTEXT_FIELD_BINDING_PATTERN_NAME,
		PARSER_RULE_CONTEXT_XML_ATOMIC_NAME_IDENTIFIER,
		PARSER_RULE_CONTEXT_MAPPING_FIELD_NAME,
		PARSER_RULE_CONTEXT_WORKER_NAME,
		PARSER_RULE_CONTEXT_NAMED_WORKERS,
		PARSER_RULE_CONTEXT_ANNOTATION_TAG,
		PARSER_RULE_CONTEXT_AFTER_PARAMETER_TYPE,
		PARSER_RULE_CONTEXT_MODULE_ENUM_NAME,
		PARSER_RULE_CONTEXT_ENUM_MEMBER_NAME,
		PARSER_RULE_CONTEXT_TYPED_BINDING_PATTERN_TYPE_RHS,
		PARSER_RULE_CONTEXT_ASSIGNMENT_STMT,
		PARSER_RULE_CONTEXT_XML_NAME,
		PARSER_RULE_CONTEXT_ACCESS_EXPRESSION,
		PARSER_RULE_CONTEXT_BINDING_PATTERN_STARTING_IDENTIFIER,
		PARSER_RULE_CONTEXT_COMPUTED_FIELD_NAME,
		PARSER_RULE_CONTEXT_SIMPLE_BINDING_PATTERN,
		PARSER_RULE_CONTEXT_ERROR_FIELD_BINDING_PATTERN,
		PARSER_RULE_CONTEXT_ERROR_CAUSE_SIMPLE_BINDING_PATTERN,
		PARSER_RULE_CONTEXT_PATH_SEGMENT_IDENT,
		PARSER_RULE_CONTEXT_NAMED_ARG_BINDING_PATTERN,
		PARSER_RULE_CONTEXT_MODULE_VAR_FIRST_QUAL,
		PARSER_RULE_CONTEXT_MODULE_VAR_SECOND_QUAL,
		PARSER_RULE_CONTEXT_MODULE_VAR_THIRD_QUAL,
		PARSER_RULE_CONTEXT_OBJECT_MEMBER_VISIBILITY_QUAL:
		return &ERROR_MISSING_IDENTIFIER
	case PARSER_RULE_CONTEXT_EXPRESSION,
		PARSER_RULE_CONTEXT_TERMINAL_EXPRESSION:
		return &ERROR_MISSING_EXPRESSION
	case PARSER_RULE_CONTEXT_STRING_LITERAL_TOKEN:
		return &ERROR_MISSING_STRING_LITERAL
	case PARSER_RULE_CONTEXT_DECIMAL_INTEGER_LITERAL_TOKEN,
		PARSER_RULE_CONTEXT_SIGNED_INT_OR_FLOAT_RHS:
		return &ERROR_MISSING_DECIMAL_INTEGER_LITERAL
	case PARSER_RULE_CONTEXT_HEX_INTEGER_LITERAL_TOKEN:
		return &ERROR_MISSING_HEX_INTEGER_LITERAL
	case PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS,
		PARSER_RULE_CONTEXT_BINDING_PATTERN_OR_VAR_REF_RHS:
		return &ERROR_MISSING_SEMICOLON_TOKEN
	case PARSER_RULE_CONTEXT_NIL_LITERAL,
		PARSER_RULE_CONTEXT_ERROR_MATCH_PATTERN:
		return &ERROR_MISSING_ERROR_KEYWORD
	case PARSER_RULE_CONTEXT_DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
		return &ERROR_MISSING_DECIMAL_FLOATING_POINT_LITERAL
	case PARSER_RULE_CONTEXT_HEX_FLOATING_POINT_LITERAL_TOKEN:
		return &ERROR_MISSING_HEX_FLOATING_POINT_LITERAL
	case PARSER_RULE_CONTEXT_STATEMENT,
		PARSER_RULE_CONTEXT_STATEMENT_WITHOUT_ANNOTS:
		return &ERROR_MISSING_CLOSE_BRACE_TOKEN
	case PARSER_RULE_CONTEXT_XML_COMMENT_CONTENT,
		PARSER_RULE_CONTEXT_XML_PI_DATA:
		return &ERROR_MISSING_XML_TEXT_CONTENT
	default:
		return p.getSeperatorTokenErrorCode()
	}
}

func (p ParserRuleContext) getSeperatorTokenErrorCode() diagnostics.DiagnosticCode {
	switch p {
	case PARSER_RULE_CONTEXT_BITWISE_AND_OPERATOR:
		return &ERROR_MISSING_BITWISE_AND_TOKEN
	case PARSER_RULE_CONTEXT_EQUAL_OR_RIGHT_ARROW,
		PARSER_RULE_CONTEXT_ASSIGN_OP:
		return &ERROR_MISSING_EQUAL_TOKEN
	case PARSER_RULE_CONTEXT_BINARY_OPERATOR,
		PARSER_RULE_CONTEXT_UNARY_OPERATOR,
		PARSER_RULE_CONTEXT_COMPOUND_BINARY_OPERATOR,
		PARSER_RULE_CONTEXT_UNARY_EXPRESSION,
		PARSER_RULE_CONTEXT_EXPRESSION_RHS,
		PARSER_RULE_CONTEXT_PLUS_TOKEN:
		return &ERROR_MISSING_BINARY_OPERATOR
	case PARSER_RULE_CONTEXT_CLOSE_BRACE:
		return &ERROR_MISSING_CLOSE_BRACE_TOKEN
	case PARSER_RULE_CONTEXT_CLOSE_PARENTHESIS,
		PARSER_RULE_CONTEXT_ARG_LIST_CLOSE_PAREN:
		return &ERROR_MISSING_CLOSE_PAREN_TOKEN
	case PARSER_RULE_CONTEXT_COMMA,
		PARSER_RULE_CONTEXT_ERROR_MESSAGE_BINDING_PATTERN_END_COMMA,
		PARSER_RULE_CONTEXT_ERROR_MESSAGE_MATCH_PATTERN_END_COMMA:
		return &ERROR_MISSING_COMMA_TOKEN
	case PARSER_RULE_CONTEXT_OPEN_BRACE:
		return &ERROR_MISSING_OPEN_BRACE_TOKEN
	case PARSER_RULE_CONTEXT_OPEN_PARENTHESIS,
		PARSER_RULE_CONTEXT_ARG_LIST_OPEN_PAREN,
		PARSER_RULE_CONTEXT_PARENTHESISED_TYPE_DESC_START:
		return &ERROR_MISSING_OPEN_PAREN_TOKEN
	case PARSER_RULE_CONTEXT_SEMICOLON,
		PARSER_RULE_CONTEXT_OBJECT_FIELD_RHS:
		return &ERROR_MISSING_SEMICOLON_TOKEN
	case PARSER_RULE_CONTEXT_ASTERISK:
		return &ERROR_MISSING_ASTERISK_TOKEN
	case PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_END:
		return &ERROR_MISSING_CLOSE_BRACE_PIPE_TOKEN
	case PARSER_RULE_CONTEXT_CLOSED_RECORD_BODY_START:
		return &ERROR_MISSING_OPEN_BRACE_PIPE_TOKEN
	case PARSER_RULE_CONTEXT_ELLIPSIS:
		return &ERROR_MISSING_ELLIPSIS_TOKEN
	case PARSER_RULE_CONTEXT_QUESTION_MARK:
		return &ERROR_MISSING_QUESTION_MARK_TOKEN
	case PARSER_RULE_CONTEXT_CLOSE_BRACKET:
		return &ERROR_MISSING_CLOSE_BRACKET_TOKEN
	case PARSER_RULE_CONTEXT_DOT,
		PARSER_RULE_CONTEXT_METHOD_CALL_DOT:
		return &ERROR_MISSING_DOT_TOKEN
	case PARSER_RULE_CONTEXT_OPEN_BRACKET,
		PARSER_RULE_CONTEXT_TUPLE_TYPE_DESC_START:
		return &ERROR_MISSING_OPEN_BRACKET_TOKEN
	case PARSER_RULE_CONTEXT_SLASH,
		PARSER_RULE_CONTEXT_ABSOLUTE_PATH_SINGLE_SLASH,
		PARSER_RULE_CONTEXT_RESOURCE_METHOD_CALL_SLASH_TOKEN:
		return &ERROR_MISSING_SLASH_TOKEN
	case PARSER_RULE_CONTEXT_COLON,
		PARSER_RULE_CONTEXT_VAR_REF_COLON,
		PARSER_RULE_CONTEXT_TYPE_REF_COLON:
		return &ERROR_MISSING_COLON_TOKEN
	case PARSER_RULE_CONTEXT_AT:
		return &ERROR_MISSING_AT_TOKEN
	case PARSER_RULE_CONTEXT_RIGHT_ARROW:
		return &ERROR_MISSING_RIGHT_ARROW_TOKEN
	case PARSER_RULE_CONTEXT_GT,
		PARSER_RULE_CONTEXT_GT_TOKEN,
		PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG_END,
		PARSER_RULE_CONTEXT_XML_ATTRIBUTES,
		PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_END_GT:
		return &ERROR_MISSING_GT_TOKEN
	case PARSER_RULE_CONTEXT_LT,
		PARSER_RULE_CONTEXT_LT_TOKEN,
		PARSER_RULE_CONTEXT_XML_START_OR_EMPTY_TAG,
		PARSER_RULE_CONTEXT_XML_END_TAG,
		PARSER_RULE_CONTEXT_INFERRED_TYPEDESC_DEFAULT_START_LT,
		PARSER_RULE_CONTEXT_STREAM_TYPE_PARAM_START_TOKEN:
		return &ERROR_MISSING_LT_TOKEN
	case PARSER_RULE_CONTEXT_SYNC_SEND_TOKEN:
		return &ERROR_MISSING_SYNC_SEND_TOKEN
	case PARSER_RULE_CONTEXT_ANNOT_CHAINING_TOKEN:
		return &ERROR_MISSING_ANNOT_CHAINING_TOKEN
	case PARSER_RULE_CONTEXT_OPTIONAL_CHAINING_TOKEN:
		return &ERROR_MISSING_OPTIONAL_CHAINING_TOKEN
	case PARSER_RULE_CONTEXT_DOT_LT_TOKEN:
		return &ERROR_MISSING_DOT_LT_TOKEN
	case PARSER_RULE_CONTEXT_SLASH_LT_TOKEN:
		return &ERROR_MISSING_SLASH_LT_TOKEN
	case PARSER_RULE_CONTEXT_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN:
		return &ERROR_MISSING_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN
	case PARSER_RULE_CONTEXT_SLASH_ASTERISK_TOKEN:
		return &ERROR_MISSING_SLASH_ASTERISK_TOKEN
	case PARSER_RULE_CONTEXT_MINUS_TOKEN:
		return &ERROR_MISSING_MINUS_TOKEN
	case PARSER_RULE_CONTEXT_LEFT_ARROW_TOKEN:
		return &ERROR_MISSING_LEFT_ARROW_TOKEN
	case PARSER_RULE_CONTEXT_TEMPLATE_END,
		PARSER_RULE_CONTEXT_TEMPLATE_START,
		PARSER_RULE_CONTEXT_XML_CONTENT,
		PARSER_RULE_CONTEXT_XML_TEXT:
		return &ERROR_MISSING_BACKTICK_TOKEN
	case PARSER_RULE_CONTEXT_XML_COMMENT_START:
		return &ERROR_MISSING_XML_COMMENT_START_TOKEN
	case PARSER_RULE_CONTEXT_XML_COMMENT_END:
		return &ERROR_MISSING_XML_COMMENT_END_TOKEN
	case PARSER_RULE_CONTEXT_XML_PI,
		PARSER_RULE_CONTEXT_XML_PI_START:
		return &ERROR_MISSING_XML_PI_START_TOKEN
	case PARSER_RULE_CONTEXT_XML_PI_END:
		return &ERROR_MISSING_XML_PI_END_TOKEN
	case PARSER_RULE_CONTEXT_XML_QUOTE_END,
		PARSER_RULE_CONTEXT_XML_QUOTE_START:
		return &ERROR_MISSING_DOUBLE_QUOTE_TOKEN
	case PARSER_RULE_CONTEXT_INTERPOLATION_START_TOKEN:
		return &ERROR_MISSING_INTERPOLATION_START_TOKEN
	case PARSER_RULE_CONTEXT_EXPR_FUNC_BODY_START,
		PARSER_RULE_CONTEXT_RIGHT_DOUBLE_ARROW:
		return &ERROR_MISSING_RIGHT_DOUBLE_ARROW_TOKEN
	case PARSER_RULE_CONTEXT_XML_CDATA_END:
		return &ERROR_MISSING_XML_CDATA_END_TOKEN
	default:
		return p.getKeywordErrorCode()
	}
}

func (p ParserRuleContext) getKeywordErrorCode() diagnostics.DiagnosticCode {
	switch p {
	case PARSER_RULE_CONTEXT_PUBLIC_KEYWORD:
		return &ERROR_MISSING_PUBLIC_KEYWORD
	case PARSER_RULE_CONTEXT_PRIVATE_KEYWORD:
		return &ERROR_MISSING_PRIVATE_KEYWORD
	case PARSER_RULE_CONTEXT_ABSTRACT_KEYWORD:
		return &ERROR_MISSING_ABSTRACT_KEYWORD
	case PARSER_RULE_CONTEXT_CLIENT_KEYWORD:
		return &ERROR_MISSING_CLIENT_KEYWORD
	case PARSER_RULE_CONTEXT_IMPORT_KEYWORD:
		return &ERROR_MISSING_IMPORT_KEYWORD
	case PARSER_RULE_CONTEXT_FUNCTION_KEYWORD,
		PARSER_RULE_CONTEXT_FUNCTION_IDENT,
		PARSER_RULE_CONTEXT_OPTIONAL_PEER_WORKER,
		PARSER_RULE_CONTEXT_DEFAULT_WORKER_NAME_IN_ASYNC_SEND:
		return &ERROR_MISSING_FUNCTION_KEYWORD
	case PARSER_RULE_CONTEXT_CONST_KEYWORD:
		return &ERROR_MISSING_CONST_KEYWORD
	case PARSER_RULE_CONTEXT_LISTENER_KEYWORD:
		return &ERROR_MISSING_LISTENER_KEYWORD
	case PARSER_RULE_CONTEXT_SERVICE_KEYWORD,
		PARSER_RULE_CONTEXT_SERVICE_IDENT,
		PARSER_RULE_CONTEXT_SERVICE_DECL_QUALIFIER:
		return &ERROR_MISSING_SERVICE_KEYWORD
	case PARSER_RULE_CONTEXT_XMLNS_KEYWORD,
		PARSER_RULE_CONTEXT_XML_NAMESPACE_DECLARATION:
		return &ERROR_MISSING_XMLNS_KEYWORD
	case PARSER_RULE_CONTEXT_ANNOTATION_KEYWORD:
		return &ERROR_MISSING_ANNOTATION_KEYWORD
	case PARSER_RULE_CONTEXT_TYPE_KEYWORD:
		return &ERROR_MISSING_TYPE_KEYWORD
	case PARSER_RULE_CONTEXT_RECORD_KEYWORD,
		PARSER_RULE_CONTEXT_RECORD_FIELD,
		PARSER_RULE_CONTEXT_RECORD_IDENT:
		return &ERROR_MISSING_RECORD_KEYWORD
	case PARSER_RULE_CONTEXT_OBJECT_KEYWORD,
		PARSER_RULE_CONTEXT_OBJECT_IDENT,
		PARSER_RULE_CONTEXT_OBJECT_TYPE_DESCRIPTOR,
		PARSER_RULE_CONTEXT_FIRST_OBJECT_CONS_QUALIFIER,
		PARSER_RULE_CONTEXT_SECOND_OBJECT_CONS_QUALIFIER,
		PARSER_RULE_CONTEXT_FIRST_OBJECT_TYPE_QUALIFIER,
		PARSER_RULE_CONTEXT_SECOND_OBJECT_TYPE_QUALIFIER:
		return &ERROR_MISSING_OBJECT_KEYWORD
	case PARSER_RULE_CONTEXT_AS_KEYWORD:
		return &ERROR_MISSING_AS_KEYWORD
	case PARSER_RULE_CONTEXT_ON_KEYWORD:
		return &ERROR_MISSING_ON_KEYWORD
	case PARSER_RULE_CONTEXT_FINAL_KEYWORD:
		return &ERROR_MISSING_FINAL_KEYWORD
	case PARSER_RULE_CONTEXT_SOURCE_KEYWORD:
		return &ERROR_MISSING_SOURCE_KEYWORD
	case PARSER_RULE_CONTEXT_WORKER_KEYWORD:
		return &ERROR_MISSING_WORKER_KEYWORD
	case PARSER_RULE_CONTEXT_FIELD_IDENT:
		return &ERROR_MISSING_FIELD_KEYWORD
	case PARSER_RULE_CONTEXT_RETURNS_KEYWORD:
		return &ERROR_MISSING_RETURNS_KEYWORD
	case PARSER_RULE_CONTEXT_RETURN_KEYWORD:
		return &ERROR_MISSING_RETURN_KEYWORD
	case PARSER_RULE_CONTEXT_EXTERNAL_KEYWORD:
		return &ERROR_MISSING_EXTERNAL_KEYWORD
	case PARSER_RULE_CONTEXT_BOOLEAN_LITERAL:
		return &ERROR_MISSING_TRUE_KEYWORD
	case PARSER_RULE_CONTEXT_IF_KEYWORD:
		return &ERROR_MISSING_IF_KEYWORD
	case PARSER_RULE_CONTEXT_ELSE_KEYWORD:
		return &ERROR_MISSING_ELSE_KEYWORD
	case PARSER_RULE_CONTEXT_WHILE_KEYWORD:
		return &ERROR_MISSING_WHILE_KEYWORD
	case PARSER_RULE_CONTEXT_CHECKING_KEYWORD:
		return &ERROR_MISSING_CHECK_KEYWORD
	case PARSER_RULE_CONTEXT_PANIC_KEYWORD:
		return &ERROR_MISSING_PANIC_KEYWORD
	case PARSER_RULE_CONTEXT_CONTINUE_KEYWORD:
		return &ERROR_MISSING_CONTINUE_KEYWORD
	case PARSER_RULE_CONTEXT_BREAK_KEYWORD:
		return &ERROR_MISSING_BREAK_KEYWORD
	case PARSER_RULE_CONTEXT_TYPEOF_KEYWORD:
		return &ERROR_MISSING_TYPEOF_KEYWORD
	case PARSER_RULE_CONTEXT_IS_KEYWORD:
		return &ERROR_MISSING_IS_KEYWORD
	case PARSER_RULE_CONTEXT_NULL_KEYWORD:
		return &ERROR_MISSING_NULL_KEYWORD
	case PARSER_RULE_CONTEXT_LOCK_KEYWORD:
		return &ERROR_MISSING_LOCK_KEYWORD
	case PARSER_RULE_CONTEXT_FORK_KEYWORD:
		return &ERROR_MISSING_FORK_KEYWORD
	case PARSER_RULE_CONTEXT_TRAP_KEYWORD:
		return &ERROR_MISSING_TRAP_KEYWORD
	case PARSER_RULE_CONTEXT_IN_KEYWORD:
		return &ERROR_MISSING_IN_KEYWORD
	case PARSER_RULE_CONTEXT_FOREACH_KEYWORD:
		return &ERROR_MISSING_FOREACH_KEYWORD
	case PARSER_RULE_CONTEXT_TABLE_KEYWORD:
		return &ERROR_MISSING_TABLE_KEYWORD
	case PARSER_RULE_CONTEXT_KEY_KEYWORD:
		return &ERROR_MISSING_KEY_KEYWORD
	case PARSER_RULE_CONTEXT_LET_KEYWORD:
		return &ERROR_MISSING_LET_KEYWORD
	case PARSER_RULE_CONTEXT_NEW_KEYWORD:
		return &ERROR_MISSING_NEW_KEYWORD
	case PARSER_RULE_CONTEXT_FROM_KEYWORD:
		return &ERROR_MISSING_FROM_KEYWORD
	case PARSER_RULE_CONTEXT_WHERE_KEYWORD:
		return &ERROR_MISSING_WHERE_KEYWORD
	case PARSER_RULE_CONTEXT_SELECT_KEYWORD:
		return &ERROR_MISSING_SELECT_KEYWORD
	case PARSER_RULE_CONTEXT_START_KEYWORD:
		return &ERROR_MISSING_START_KEYWORD
	case PARSER_RULE_CONTEXT_FLUSH_KEYWORD:
		return &ERROR_MISSING_FLUSH_KEYWORD
	case PARSER_RULE_CONTEXT_WAIT_KEYWORD:
		return &ERROR_MISSING_WAIT_KEYWORD
	case PARSER_RULE_CONTEXT_DO_KEYWORD:
		return &ERROR_MISSING_DO_KEYWORD
	case PARSER_RULE_CONTEXT_TRANSACTION_KEYWORD:
		return &ERROR_MISSING_TRANSACTION_KEYWORD
	case PARSER_RULE_CONTEXT_TRANSACTIONAL_KEYWORD:
		return &ERROR_MISSING_TRANSACTIONAL_KEYWORD
	case PARSER_RULE_CONTEXT_COMMIT_KEYWORD:
		return &ERROR_MISSING_COMMIT_KEYWORD
	case PARSER_RULE_CONTEXT_ROLLBACK_KEYWORD:
		return &ERROR_MISSING_ROLLBACK_KEYWORD
	case PARSER_RULE_CONTEXT_RETRY_KEYWORD:
		return &ERROR_MISSING_RETRY_KEYWORD
	case PARSER_RULE_CONTEXT_ENUM_KEYWORD:
		return &ERROR_MISSING_ENUM_KEYWORD
	case PARSER_RULE_CONTEXT_BASE16_KEYWORD:
		return &ERROR_MISSING_BASE16_KEYWORD
	case PARSER_RULE_CONTEXT_BASE64_KEYWORD:
		return &ERROR_MISSING_BASE64_KEYWORD
	case PARSER_RULE_CONTEXT_MATCH_KEYWORD:
		return &ERROR_MISSING_MATCH_KEYWORD
	case PARSER_RULE_CONTEXT_CONFLICT_KEYWORD:
		return &ERROR_MISSING_CONFLICT_KEYWORD
	case PARSER_RULE_CONTEXT_LIMIT_KEYWORD:
		return &ERROR_MISSING_LIMIT_KEYWORD
	case PARSER_RULE_CONTEXT_ORDER_KEYWORD:
		return &ERROR_MISSING_ORDER_KEYWORD
	case PARSER_RULE_CONTEXT_BY_KEYWORD:
		return &ERROR_MISSING_BY_KEYWORD
	case PARSER_RULE_CONTEXT_GROUP_KEYWORD:
		return &ERROR_MISSING_GROUP_KEYWORD
	case PARSER_RULE_CONTEXT_ORDER_DIRECTION:
		return &ERROR_MISSING_ASCENDING_KEYWORD
	case PARSER_RULE_CONTEXT_JOIN_KEYWORD:
		return &ERROR_MISSING_JOIN_KEYWORD
	case PARSER_RULE_CONTEXT_OUTER_KEYWORD:
		return &ERROR_MISSING_OUTER_KEYWORD
	case PARSER_RULE_CONTEXT_FAIL_KEYWORD:
		return &ERROR_MISSING_FAIL_KEYWORD
	case PARSER_RULE_CONTEXT_PIPE,
		PARSER_RULE_CONTEXT_UNION_OR_INTERSECTION_TOKEN:
		return &ERROR_MISSING_PIPE_TOKEN
	case PARSER_RULE_CONTEXT_EQUALS_KEYWORD:
		return &ERROR_MISSING_EQUALS_KEYWORD
	case PARSER_RULE_CONTEXT_REMOTE_IDENT:
		return &ERROR_MISSING_REMOTE_KEYWORD

	// Type keywords
	case PARSER_RULE_CONTEXT_STRING_KEYWORD:
		return &ERROR_MISSING_STRING_KEYWORD
	case PARSER_RULE_CONTEXT_XML_KEYWORD:
		return &ERROR_MISSING_XML_KEYWORD
	case PARSER_RULE_CONTEXT_RE_KEYWORD:
		return &ERROR_MISSING_RE_KEYWORD
	case PARSER_RULE_CONTEXT_VAR_KEYWORD:
		return &ERROR_MISSING_VAR_KEYWORD
	case PARSER_RULE_CONTEXT_MAP_KEYWORD,
		PARSER_RULE_CONTEXT_NAMED_WORKER_DECL,
		PARSER_RULE_CONTEXT_MAP_TYPE_DESCRIPTOR:
		return &ERROR_MISSING_MAP_KEYWORD
	case PARSER_RULE_CONTEXT_ERROR_KEYWORD,
		PARSER_RULE_CONTEXT_ERROR_BINDING_PATTERN,
		PARSER_RULE_CONTEXT_PARAMETERIZED_TYPE:
		return &ERROR_MISSING_ERROR_KEYWORD
	case PARSER_RULE_CONTEXT_STREAM_KEYWORD:
		return &ERROR_MISSING_STREAM_KEYWORD
	case PARSER_RULE_CONTEXT_READONLY_KEYWORD:
		return &ERROR_MISSING_READONLY_KEYWORD
	case PARSER_RULE_CONTEXT_DISTINCT_KEYWORD:
		return &ERROR_MISSING_DISTINCT_KEYWORD
	case PARSER_RULE_CONTEXT_CLASS_KEYWORD,
		PARSER_RULE_CONTEXT_FIRST_CLASS_TYPE_QUALIFIER,
		PARSER_RULE_CONTEXT_SECOND_CLASS_TYPE_QUALIFIER,
		PARSER_RULE_CONTEXT_THIRD_CLASS_TYPE_QUALIFIER,
		PARSER_RULE_CONTEXT_FOURTH_CLASS_TYPE_QUALIFIER:
		return &ERROR_MISSING_CLASS_KEYWORD
	case PARSER_RULE_CONTEXT_COLLECT_KEYWORD:
		return &ERROR_MISSING_COLLECT_KEYWORD
	case PARSER_RULE_CONTEXT_NATURAL_KEYWORD:
		return &ERROR_MISSING_NATURAL_KEYWORD
	default:
		return &ERROR_SYNTAX_ERROR
	}
}
