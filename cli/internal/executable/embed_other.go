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

//go:build !darwin && !linux && !windows

package executable

import (
	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// tryLoadFrom always reports "not packed" here (e.g. js/wasm) — bal
// build's --target-os is limited to darwin/linux/windows.
func tryLoadFrom(string) ([]*bir.BIRPackage, semtypes.Env, error) {
	return nil, nil, nil
}
