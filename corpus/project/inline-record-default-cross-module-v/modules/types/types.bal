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

import testorg/inline_record_default_cross_module.defaults;

public type NestedAlias record {|
    record {| int nestedValue = defaults:value(2); |} nested;
|};

public type AliasList record {| int listValue = defaults:value(3); |}[];

public type Factory function () returns record {| int returnedValue = defaults:value(5); |};

public function parameterDefault(record {| int parameterValue = defaults:value(1); |} value) returns int {
    return value.parameterValue;
}

public function restDefaults(record {| int restValue = defaults:value(4); |}... values) returns int {
    return values[0].restValue + values[1].restValue;
}

public function invokeFactory(Factory factory) returns int {
    return factory().returnedValue;
}
