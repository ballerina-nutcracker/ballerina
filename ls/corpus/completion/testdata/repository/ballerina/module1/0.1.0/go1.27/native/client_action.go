// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package native

import (
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

func get(_ string, _ *values.TypeDesc) any {
	return &struct{}{}
}

func forward(_ string, _ *values.Object, _ *values.TypeDesc) any {
	return &struct{}{}
}

func delete(_ *values.TypeDesc) any {
	return &struct{}{}
}

func responses(_ *values.TypeDesc) *values.Stream {
	return values.NewStream(semtypes.Stream, nil, nil)
}

func postResource(_ *values.List, _, _, _ any, _ *values.TypeDesc) any {
	return &struct{}{}
}
