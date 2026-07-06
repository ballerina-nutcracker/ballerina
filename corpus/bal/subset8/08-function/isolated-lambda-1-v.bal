
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

function capturedParam(int n) returns int {
    isolated function () returns int f = isolated function () returns int {
        return n;
    };
    return f();
}

function nestedLayeredCapture(int n) returns int {
    function () returns int f1 = function () returns int {
        isolated function () returns int f2 = isolated function () returns int {
            isolated function () returns int f3 = isolated function () returns int {
                return n;
            };
            return f3();
        };
        return f2();
    };
    return f1();
}

public function main() {
    isolated function () returns int noCapture = isolated function () returns int {
        return 10;
    };

    final int x = 20;
    isolated function () returns int capturedFinal = isolated function () returns int {
        return x;
    };

    io:println(noCapture()); // @output 10
    io:println(capturedFinal()); // @output 20
    io:println(capturedParam(30)); // @output 30
    io:println(nestedLayeredCapture(40)); // @output 40
}
