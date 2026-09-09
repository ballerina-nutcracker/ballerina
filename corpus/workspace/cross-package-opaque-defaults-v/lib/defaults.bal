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

// The record type and the defaulted functions are defined in this package, so
// their `$default$N` providers are compiled here, serialized into this
// package's BIR and read back by the depending package. The `array:indexOf`
// provider they call belongs to `lang.array`, a third package.
isolated function letters() returns int[] => [10, 20, 30, 20];

public type Positions record {|
    int? first = array:indexOf(letters(), 20);
    int? fromThird = array:indexOf(letters(), 20, 2);
    int? named = array:indexOf(letters(), val = 30);
|};

public function firstOf(int[] arr, int val, int? pos = arr.indexOf(val)) returns int? => pos;

public function firstNamed(int[] arr, int? pos = arr.indexOf(val = 20)) returns int? => pos;

public function firstQualified(int[] arr, int? pos = array:indexOf(arr, 30)) returns int? => pos;
