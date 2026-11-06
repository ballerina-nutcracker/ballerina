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

function failWith(string message) returns error? {
    return error(message);
}

// A multiple receive that can fail may be returned, or be the operand of check
// or checkpanic (also in parentheses): each ends the worker on error.
function returned() returns record {| int a; int b; |}|error {
    worker a returns error? {
        check failWith("a failed");
        check (1 ->> function);
    }
    worker b returns error? {
        check (2 ->> function);
    }
    return <- {a, b};
}

function checked() returns int|error {
    worker a returns error? {
        check (1 ->> function);
    }
    worker b returns error? {
        check failWith("b failed");
        check (2 ->> function);
    }
    record {| int a; int b; |} r = check (<- {a, b});
    return r.a + r.b;
}

function checkPanicked() returns int {
    worker a {
        1 -> function;
    }
    worker b returns error? {
        2 -> function;
    }
    record {| int a; int b; |} r = checkpanic <- {a, b};
    return r.a + r.b;
}

public function main() {
    io:println(returned()); // @output error("a failed")
    io:println(checked()); // @output error("b failed")
    io:println(checkPanicked()); // @output 3
}
