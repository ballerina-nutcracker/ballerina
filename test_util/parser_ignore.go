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

package test_util

// RegexParserIgnoreList lists parser testdata files (relative to parser/testdata/bal) that hang the parser
// because the regex parser is not yet implemented.
var RegexParserIgnoreList = []string{
	"bala/test_bala/types/regexp_type_test.bal",
	"bala/test_projects/test_project_regexp/regexpTypes.bal",
	"jvm/largeMethods/modules/functions/large-functions.bal",
	"query/query-action.bal",
	"query/query-expr-with-query-construct-type.bal",
	"query/query_action_or_expr.bal",
	"query/simple-query-with-defined-type.bal",
	"types/readonly/test_inherently_immutable_type.bal",
	"types/readonly/test_selectively_immutable_type.bal",
	"types/regexp/regexp_type_test.bal",
	"types/regexp/regexp_value_negative_test.bal",
	"types/regexp/regexp_value_test.bal",
	"types/table/table_key_field_value_test.bal",
}
