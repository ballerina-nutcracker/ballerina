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

// The record type and its field defaults live in this module, so the
// `$default$N` providers for the fields are compiled here while the
// `array:indexOf` provider they call belongs to `lang.array`. The importing
// module only ever sees the resulting record type.
isolated function letters() returns int[] => [10, 20, 30, 20];

public type Positions record {|
    int? first = array:indexOf(letters(), 20);
    int? fromThird = array:indexOf(letters(), 20, 2);
    int? named = array:indexOf(letters(), val = 30);
|};
