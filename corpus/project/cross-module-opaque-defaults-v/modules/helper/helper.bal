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

import ballerina/lang.array;

// Public functions whose default parameters omit `array:indexOf`'s own
// `startIndex`. The provider for `pos` is compiled in this module and invoked
// from the caller's module, so the omitted-default lowering has to survive the
// cross-module call.
public function firstOf(int[] arr, int val, int? pos = arr.indexOf(val)) returns int? => pos;

public function firstNamed(int[] arr, int? pos = arr.indexOf(val = 20)) returns int? => pos;

public function firstQualified(int[] arr, int? pos = array:indexOf(arr, 30)) returns int? => pos;
