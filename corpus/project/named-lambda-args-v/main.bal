// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import ballerina/io;
import testorg/named_lambda_args.util;

type Pair record {|
    int left;
    int right;
|};

public function main() {
    function(int a = 5) returns int base = function(int x = 10) returns int {
        return x;
    };
    io:println(base()); // @output 5
    io:println(util:callWithDefault(base)); // @output 30

    function(*Pair p) returns int recordFn = function(Pair q) returns int {
        return q.left + q.right;
    };
    io:println(recordFn(left = 4, right = 6)); // @output 10
}
