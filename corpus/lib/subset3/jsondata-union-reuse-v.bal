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

import ballerina/data.jsondata;
import ballerina/io;

type R record {|
    U? a;
|};

type U R|map<U>|U[];

type A record {|
    A|B? a;
    string b?;
|};

type B record {|
    A|B? a;
    int b;
|};

type T1 record {|
    int v;
|};

type T2 record {|
    string v;
|};

type Q1 record {|
    T1|T2? t;
    string u?;
|};

type Q2 record {|
    T1|T2? t;
    int u;
|};

type P1 record {|
    Q1|Q2 q;
    string s?;
|};

type P2 record {|
    Q1|Q2 q;
    int s;
|};

type W record {|
    P1|P2 p;
    int k;
|};

type AB A|B;

type Two record {|
    AB l;
    AB r;
|};

type TwoOrAB Two|AB;

function rep(string x, int n) returns string {
    if n == 0 {
        return "";
    }
    string h = rep(x, n / 2);
    return n % 2 == 0 ? h + h : h + h + x;
}

public function main() {
    // R names the field it converts and map<U> does not, so each node is reached with every name count.
    string invalid = rep("{\"a\":", 2000) + "{\"z\":1}" + rep("}", 2000);
    U|error u1 = jsondata:parseString(invalid);
    io:println(u1 is jsondata:Error);
    // @output true
    json|error invalidJson = jsondata:parseString(invalid);
    if invalidJson is json {
        U|error u2 = jsondata:parseAsType(invalidJson);
        io:println(u2 is jsondata:Error);
    }
    // @output true

    // A fails on the "b" after its nested A|B field succeeded, so every level converts the subtree to B again.
    string valid = rep("{\"a\":", 4000) + "null" + rep(",\"b\":1}", 4000);
    AB|error v1 = jsondata:parseString(valid);
    io:println(v1 is B);
    // @output true
    json|error validJson = jsondata:parseString(valid);
    if validJson is json {
        AB|error v2 = jsondata:parseAsType(validJson);
        io:println(v2 is B);
    }
    // @output true

    // P2 reuses Q1's failure and Q2's success, and the names they left are in the path of the next error.
    W|error w = jsondata:parseAsType({p: {q: {t: {v: 1}, u: 1}, s: 1}, k: "bad"}, {}, W);
    io:println(w is error ? w.message() : w);
    // @output incompatible value 'bad' for type 'int' in field 't.u.t.k'

    // A container the input shares is converted into a separate value at each place.
    json inner = {a: null, b: 1};
    json mid = {a: inner, b: 2};
    AB[]|error d1 = jsondata:parseAsType([mid, mid, {a: mid, b: 3}]);
    if d1 is AB[] {
        io:println(d1, " ", d1[0] === d1[1], " ", d1[0].a === d1[1].a, " ", d1[2].a === d1[0]);
    }
    // @output [{"a":{"a":null,"b":1},"b":2},{"a":{"a":null,"b":1},"b":2},{"a":{"a":{"a":null,"b":1},"b":2},"b":3}] false false false
    TwoOrAB|error d2 = jsondata:parseAsType({l: mid, r: mid}, {}, TwoOrAB);
    if d2 is Two {
        io:println(d2, " ", d2.l === d2.r, " ", d2.l.a === d2.r.a);
    }
    // @output {"l":{"a":{"a":null,"b":1},"b":2},"r":{"a":{"a":null,"b":1},"b":2}} false false
}
