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

type Item record {|
    int id;
|};

type A record {|
    Item[] v;
    int m;
|};

type B record {|
    string[] v;
    int m;
|};

type InnerA record {|
    int[] xs;
|};

type InnerB record {|
    string[] xs;
|};

type InnerC record {|
    boolean[] xs;
|};

type OuterA record {|
    InnerA|InnerB inner;
    int m;
|};

type OuterB record {|
    InnerC|InnerB inner;
    int m;
|};

public function main() {
    int[] arr = [1, 2];
    A|B r = {
        v: from int x in arr
            select {id: x * 3},
        m: 1
    };
    io:println(r); // @output {"v":[{"id":3},{"id":6}],"m":1}
    io:println(r is A); // @output true
    OuterA|OuterB o = {
        inner: {xs: from int x in arr select x + 1},
        m: 1
    };
    io:println(o); // @output {"inner":{"xs":[2,3]},"m":1}
    io:println(o is OuterA); // @output true
    io:println(o.inner is InnerA); // @output true
}
