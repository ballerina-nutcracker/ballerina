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

type A record {|
    int a;
|};

type Ints record {|
    int[] xs?;
|};

type Grid record {|
    int[][] rows?;
|};

type Open record {
    @jsondata:Name {value: "t"}
    string title;
};

type Node record {|
    @jsondata:Name {value: "v"}
    int val;
    Node? next;
|};

class TruncatedBlocks {
    int count = 0;

    public isolated function next() returns record {|byte[] value;|}|error? {
        self.count += 1;
        if self.count == 1 {
            return {value: "12".toBytes()};
        }
        return error("network reset");
    }
}

function show(anydata|error v) {
    if v is error {
        io:println("error: ", v.message());
    } else {
        io:println(v);
    }
}

public function main() {
    Ints|error n1 = jsondata:parseString("{\"xs\":[1,null,2]}", {allowDataProjection: {nilAsOptionalField: true}});
    if n1 is Ints {
        int sum = 0;
        foreach int x in n1.xs ?: [] {
            sum += x;
        }
        io:println(n1, " ", sum);
    }
    // @output {"xs":[1,0,2]} 3
    jsondata:Options nilAsOptional = {allowDataProjection: {nilAsOptionalField: true}};
    Ints|error n2 = jsondata:parseString("{\"xs\":[1,null]}", nilAsOptional);
    show(n2);
    // @output {"xs":[1]}
    Ints|error n3 = jsondata:parseString("{\"xs\":[1,null,null,4]}", nilAsOptional);
    show(n3);
    // @output {"xs":[1,0,0,4]}
    Grid|error n4 = jsondata:parseString("{\"rows\":[[1,null],null]}", nilAsOptional, Grid);
    show(n4);
    // @output {"rows":[[1]]}
    Grid|error n5 = jsondata:parseString("{\"rows\":[[1,null],null,[3]]}", nilAsOptional, Grid);
    show(n5);
    // @output {"rows":[[1],[],[3]]}

    (int[]|string)[]|error u1 = jsondata:parseString("[[1],[2],[3]]");
    show(u1);
    // @output [[1],[2],[3]]
    (string[]|int)[]|error u2 = jsondata:parseString("[[\"a\"],[\"b\"],5]");
    show(u2);
    // @output [["a"],["b"],5]
    record {|(int[]|string)[] a; int b;|}|error u3 = jsondata:parseString("{\"a\":[[1],[2]],\"b\":5}");
    show(u3);
    // @output {"a":[[1],[2]],"b":5}
    (A|int)[2]|error u4 = jsondata:parseString("[{\"a\":1},2,{\"x\":1}]");
    show(u4);
    // @output [{"a":1},2]
    [A|int, string]|error u5 = jsondata:parseString("[{\"a\":1},\"s\"]");
    show(u5);
    // @output [{"a":1},"s"]
    (int[][]|string)[]|error u6 = jsondata:parseString("[[[1],[2]],\"x\",[[3]]]");
    show(u6);
    // @output [[[1],[2]],"x",[[3]]]

    Open|error r1 = jsondata:parseString("{\"t\":\"x\",\"title\":\"y\"}", {}, Open);
    show(r1);
    // @output error: duplicate field 'title'
    Open|error r2 = jsondata:parseString("{\"t\":\"x\",\"title\":5}", {}, Open);
    show(r2);
    // @output error: duplicate field 'title'
    Open|error r3 = jsondata:parseAsType({"t": "x", "title": 5}, {}, Open);
    show(r3);
    // @output error: duplicate field 'title'
    Open|error r4 = jsondata:parseString("{\"t\":\"x\",\"t\":\"z\",\"other\":1}", {}, Open);
    show(r4);
    // @output {"title":"z","other":1}

    Node|error b1 = jsondata:parseString("{\"v\":1,\"next\":{\"v\":2,\"next\":null}}", {}, Node);
    show(b1);
    // @output {"val":1,"next":{"val":2,"next":null}}
    Node|error b2 = jsondata:parseAsType({"v": 1, "next": {"v": 2, "next": ()}}, {}, Node);
    show(b2);
    // @output {"val":1,"next":{"val":2,"next":null}}

    stream<byte[], error?> s1 = new (new TruncatedBlocks());
    int|error s = jsondata:parseStream(s1, {}, int);
    show(s);
    // @output error: network reset
    io:println(s is jsondata:Error);
    // @output true
}
