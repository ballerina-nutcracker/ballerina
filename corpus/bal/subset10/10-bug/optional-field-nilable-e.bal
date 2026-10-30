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

type R record {|
    int a;
    int? c?;
    int d?;
    int? e;
|};

type R1 record {
    int? foo?;
    int bar;
};

type R2 record {
    int? foo;
    string baz;
};

public function main() {
    R r = {a: 1, e: ()};
    int? x = r.c; // @error
    int? y = r?.c;
    int? z = r.d;
    int? w = r.e;
    R? nr = r;
    int? v = nr.c; // @error
    R1|R2 u = {foo: 10, bar: 10};
    int? t = u.foo; // @error
    io:println(x, y, z, w, v, t);
}
