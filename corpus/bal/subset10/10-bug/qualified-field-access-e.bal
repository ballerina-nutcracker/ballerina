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

import ballerina/io as mod;

type Outer record {
    record {int c;} b;
};

function optionalAccessOnNilableRecord() {
    record {int x;}? value = ();
    _ = value?.mod:foo; // @error
    _ = value?.'_:foo; // @error
}

function requiredAccessOnRecord(record {int foo;} value) {
    _ = value.mod:foo; // @error
}

function accessOnLaxType(json value) {
    _ = value.mod:foo; // @error
    _ = value?.mod:foo; // @error
}

function accessOnNilableXml(xml? value) {
    _ = value?.mod:foo; // @error
    _ = value.mod:foo; // @error
}

function accessOnXmlUnion(xml|int value) {
    _ = value.mod:foo; // @error
}

function chainedAccess(Outer a) {
    _ = a.mod:b.c; // @error
}

function accessInLambda(record {int foo;} value) {
    var f = function () returns int? {
        return value?.mod:foo; // @error
    };
    _ = f;
}

function accessInQuery(record {int foo;}[] values) {
    _ = from var v in values select v.mod:foo; // @error
}

function accessAsLvalue(record {int foo;} value, xml x) {
    value.mod:foo = 2; // @error
    value.mod:foo += 2; // @error
    x.mod:foo = "a"; // @error
    x.mod:foo += "a"; // @error
}

class Holder {
    int foo = 1;

    function get() returns int {
        return self.mod:foo; // @error
    }

    function missing() returns int {
        return self.mod:bar; // @error
    }
}

public function main() {
    mod:println("done");
}
