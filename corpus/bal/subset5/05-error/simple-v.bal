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

type Detail record {|
    string message;
    int id;
|};

type ErrorWithDetail error<Detail>;

function foo() returns int|error {
    return error("foo", ff = "foo");
}

function bar() returns int|error {
    return error ErrorWithDetail("bar", message = "bar message", id = 1);
}

function baz() returns int|error {
    error e = error ErrorWithDetail("bar", message = "bar message", id = 2);
    return error ErrorWithDetail("baz", e, message = "baz message", id = 3);
}

public function main() {
    int|error fooResult = foo();
    if (fooResult is error) {
        io:println(fooResult); // @output error("foo",ff="foo")
    }

    int|error barResult = bar();
    if (barResult is error) {
        io:println(barResult); // @output error("bar",message="bar message",id=1)
    }

    int|error bazResult = baz();
    if (bazResult is error) {
        io:println(bazResult); // @output error("baz",error("bar",message="bar message",id=2),message="baz message",id=3)
    }
}
