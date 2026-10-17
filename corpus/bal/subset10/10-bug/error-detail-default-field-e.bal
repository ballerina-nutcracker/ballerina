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
    int code;
    string reason = "unknown";
|};

type DetailError error<Detail>;

type CodeError error<record {| int code = 1; |}>;

type ReasonError error<record {| string reason = "r"; |}>;

type WideDetail record {| int x = 300; |};

type ByteDetail record {| byte x; |};

type NarrowedError error<WideDetail & ByteDetail>;

type FirstDefault record { int x = 1; };

type SecondDefault record { int x = 2; };

type ConflictingError error<FirstDefault & SecondDefault>;

public function main() {
    DetailError e = error("missing"); // @error code has no default
    io:println(e);
}

function union() {
    CodeError|ReasonError e = error("ambiguous"); // @error union of error types has no single detail type
    io:println(e);
}

function narrowed() {
    NarrowedError e = error("narrowed"); // @error default of x does not fit byte
    io:println(e);
}

function conflicting() {
    ConflictingError e = error("conflicting"); // @error both intersected records declare defaults
    io:println(e);
}
