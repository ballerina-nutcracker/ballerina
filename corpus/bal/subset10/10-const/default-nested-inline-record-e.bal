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
import ballerina/io;

// Outer and its inline field record are scheduled together. Resolving the inline
// record's default defines a record type that contains another record while
// Outer is still waiting, and Outer's own default is still checked.
type Outer record {|
    record {|
        boolean b = seed is record {| record {| int c = 1; |} r; |};
    |} inner = {};
    int x = "a"; // @error incompatible type in a default of an unused record
|};

final readonly & anydata seed = 1;

public function main() {
    io:println("done");
}
