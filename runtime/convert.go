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

package runtime

import (
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
	"ballerina-lang-go/values/convert"
)

// FromJsonWithType converts a json value to the given target type.
// It matches lang.value:fromJsonWithType.
func FromJsonWithType(tc semtypes.Context, value values.BalValue, targetType semtypes.SemType) (values.BalValue, *values.Error) {
	return convert.Convert(tc, value, targetType)
}
