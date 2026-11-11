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

isolated function next() returns int {
    return 42;
}

type Runtime record {|
    int x;
    int y = next();
|};

type MaybeY record {|
    int x;
    int y?;
|};

const map<int> X = {x: 1};

annotation Runtime info on type;

// The spread's type may supply y, but X does not, so the value needs y's
// nonconstant default: a non-const annotation evaluates it at runtime.
@info {...<MaybeY>X}
type T record {||};

public function main() {
    T t = {};
    io:println(t); // @output {}
}
