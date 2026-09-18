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

# Configuration of a test function.
#
# + enable - Whether the test is executed
# + before - Function executed before the test body
# + after - Function executed after the test body
public type TestConfig record {|
    boolean enable = true;
    (function () returns (any|error)) before?;
    (function () returns (any|error)) after?;
|};

# Marks a function as a test.
#
# `TestConfig` is a valid annotation type even though it is `on function` (and
# therefore runtime-visible): a `function` value is inherently immutable, hence
# `Cloneable`, so this closed record is a subtype of `map<Cloneable>`.
public annotation TestConfig Config on function;

# Marks the currently running test as failed and aborts it.
#
# + msg - The failure message
# + return - Never returns; always panics
public function assertFail(string msg = "Test Failed!") returns never {
    panic externFail(msg);
}

function externFail(string msg) returns error = external;
