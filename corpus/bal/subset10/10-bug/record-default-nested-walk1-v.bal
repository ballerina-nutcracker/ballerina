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

isolated function mk() returns any {
    return function () returns record {| int b = 2; |} { return {}; };
}

// A record field default whose expression contains another record type
// descriptor with its own field default: the outer default function body is
// desugared once when the record type is desugared and again when the
// generated default function is desugared as a package function.
type Outer record {|
    boolean isFn = mk() is function () returns record {| int b = 2; |};
    function () returns int nested = function () returns int {
        record {| int b = 2; |} r = {};
        return r.b;
    };
|};

function localOuter() returns boolean {
    record {|
        boolean isFn = mk() is function () returns record {| int b = 2; |};
    |} o = {};
    return o.isFn;
}

function paramOuter(record {| boolean isFn = mk() is function () returns record {| int b = 2; |}; |} p = {}) returns boolean {
    return p.isFn;
}

public function main() {
    Outer o = {};
    io:println(o.isFn); // @output true
    function () returns int nested = o.nested;
    io:println(nested()); // @output 2
    io:println(localOuter()); // @output true
    io:println(paramOuter()); // @output true
}
