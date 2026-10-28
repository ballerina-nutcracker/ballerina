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

type Pair record {|
    int first;
    string second;
|};

public function main() {
    worker nilWorker {
        io:println("nil"); // @output nil
    }
    worker scalarWorker returns int {
        return 42;
    }
    worker structuredWorker returns Pair {
        return {first: 1, second: "two"};
    }
    worker errorWorker returns error {
        return error("normal error value");
    }

    () nilResult = wait nilWorker;
    io:println(nilResult); // @output
    int scalar = wait scalarWorker;
    io:println(scalar); // @output 42
    Pair pair = wait structuredWorker;
    io:println(pair.first); // @output 1
    io:println(pair.second); // @output two
    error e = wait errorWorker;
    io:println(e.message()); // @output normal error value
}
