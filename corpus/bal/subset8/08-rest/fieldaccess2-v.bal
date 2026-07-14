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

public type R1 record {|
    int x;
    int...;
|};

public type R2 record {|
    int y;
    int...;
|};

public function foo(R1|R2 r) returns int? {
    // According to spec https://ballerina.io/spec/lang/master/#section_6.10
    // T is union where atleast one (`R1`) has x an an individual field descriptor
    // M (`R1|R2["x"]`) is not nil (it's `int`)
    return r.x;
}

public function main() {
    R1 r1 = {x: 5};
    io:println(foo(r1)); // @output 5
    R2 r2 = {y: 10, "x": 8};
    io:println(foo(r2)); // @output 8
    R2 r22 = {y: 10};
    io:println(foo(r22)); // @output 
}
