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
import testorg/optional_field_cross_module.types;

public function main() {
    types:R r = {foo: 10, bar: 10};
    io:println(r); // @output {"foo":10,"bar":10}
    r.foo = ();
    r.bar = ();
    io:println(r); // @output {"bar":null}

    types:R r2 = {foo: 7, bar: 1};
    int? before = r2.foo;
    io:println(before); // @output 7
    r2.foo = ();
    int? after = r2.foo;
    if after is () {
        io:println("after missing"); // @output after missing
    }

    types:R r3 = {foo: 10, bar: 10};
    r3["foo"] = ();
    io:println(r3); // @output {"bar":10}
    string k = "bar";
    r3[k] = ();
    io:println(r3); // @output {"bar":null}

    types:Derived d = {foo: 10, bar: 20};
    io:println(d); // @output {"foo":10,"bar":20}
    d.foo = ();
    io:println(d); // @output {"bar":20}

    types:U1|types:U2 u = {foo: 10, bar: 10};
    u.foo = ();
    io:println(u); // @output {"bar":10}
    int? uf = u.foo;
    io:println(uf); // @output
}
