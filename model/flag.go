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

package model

type Flag uint64

const (
	FlagPublic                       Flag = 1 << 0
	FlagRemote                       Flag = 1 << 2
	FlagTransactional                Flag = 1 << 3
	FlagNative                       Flag = 1 << 4
	FlagFinal                        Flag = 1 << 5
	FlagAttached                     Flag = 1 << 6
	FlagLambda                       Flag = 1 << 7
	FlagWorker                       Flag = 1 << 8
	FlagParallel                     Flag = 1 << 9
	FlagListener                     Flag = 1 << 10
	FlagReadonly                     Flag = 1 << 11
	FlagFunctionFinal                Flag = 1 << 12
	FlagInterface                    Flag = 1 << 13
	FlagRequired                     Flag = 1 << 14
	FlagRecord                       Flag = 1 << 15
	FlagAnonymous                    Flag = 1 << 16
	FlagOptional                     Flag = 1 << 17
	FlagTestable                     Flag = 1 << 18
	FlagClient                       Flag = 1 << 19
	FlagResource                     Flag = 1 << 20
	FlagIsolated                     Flag = 1 << 21
	FlagService                      Flag = 1 << 22
	FlagConstant                     Flag = 1 << 23
	FlagTypeParam                    Flag = 1 << 24
	FlagLangLib                      Flag = 1 << 25
	FlagForked                       Flag = 1 << 26
	FlagDistinct                     Flag = 1 << 27
	FlagClass                        Flag = 1 << 28
	FlagConfigurable                 Flag = 1 << 29
	FlagObjectCtor                   Flag = 1 << 30
	FlagEnum                         Flag = 1 << 31
	FlagIncluded                     Flag = 1 << 32
	FlagRequiredParam                Flag = 1 << 33
	FlagDefaultableParam             Flag = 1 << 34
	FlagRestParam                    Flag = 1 << 35
	FlagField                        Flag = 1 << 36
	FlagAnyFunction                  Flag = 1 << 37
	FlagNeverAllowed                 Flag = 1 << 38
	FlagEnumMember                   Flag = 1 << 39
	FlagQueryLambda                  Flag = 1 << 40
	FlagDeprecated                   Flag = 1 << 41
	FlagParameterized                Flag = 1 << 42
	FlagIsolatedParam                Flag = 1 << 43
	FlagInfer                        Flag = 1 << 44
	FlagEffectiveTypeDef             Flag = 1 << 45
	FlagSourceAnnotation             Flag = 1 << 46
	FlagExplicitReturnTypeDescriptor Flag = 1 << 47
)

func (f Flag) Has(flag Flag) bool {
	return f&flag == flag
}
