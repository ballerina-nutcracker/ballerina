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

public function main() {
    foo({a: 2, b: 3}); // @output any
    foo({a: 2, "b": 3}); // @output any
    foo1({a: 2, "b": 3}); // @output { any a; }
    foo2({a: 2, b: 3}); // @output {| any a; any b; |}
    foo2({a: 2, "b": 3}); // @output {| any a; any b; |}
    foo3({a: 2, b: 3}); // @output {| int a; int b; |}
    foo3({a: 2, "b": 3}); // @output {| int a; int b; |}
}

// function call arguments get their expected type from the function type
function foo(any a) {
    if a is record {|int a; int b;|} {
        io:println("unexpected");
    }
    io:println("any");
}

function foo1(record {any a;} c) {
    if c is record {|int a; int b;|} {
        io:println("unexpected");
    }
    if c is record {|2 a; 3 b;|} {
        io:println("unexpected");
    }
    io:println("{ any a; }");
}

function foo2(record {|any a; any b;|} c) {
    if c is record {|int a; int b;|} {
        io:println("unexpected");
    }
    if c is record {|2 a; 3 b;|} {
        io:println("unexpected");
    }
    io:println("{| any a; any b; |}");
}

function foo3(record {|int a; int b;|} c) {
    if c is record {|2 a; 3 b;|} {
        io:println("unexpected");
    }
    else {
        io:println("{| int a; int b; |}");
    }
}
