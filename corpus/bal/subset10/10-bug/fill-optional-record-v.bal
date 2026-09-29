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

type Inner record {|
    int y?;
|};

type OpenInner record {
    int y?;
};

type Holder record {|
    Inner i?;
|};

type R record {|
    R r?;
    int x?;
|};

public function main() {
    map<Inner> mi = {};
    mi["k"]["y"] = 1;
    io:println(mi); // @output {"k":{"y":1}}

    map<OpenInner> mo = {};
    mo["k"]["y"] = 2;
    io:println(mo); // @output {"k":{"y":2}}

    Holder h = {};
    h.i.y = 3;
    io:println(h); // @output {"i":{"y":3}}

    Inner[] arr = [];
    arr[1].y = 4;
    io:println(arr); // @output [{},{"y":4}]

    Inner[2] fixed = [];
    io:println(fixed); // @output [{},{}]
    fixed[0].y = 5;
    io:println(fixed); // @output [{"y":5},{}]

    map<Inner> perKey = {};
    perKey["a"]["y"] = 1;
    perKey["b"]["y"] = 2;
    io:println(perKey); // @output {"a":{"y":1},"b":{"y":2}}

    map<map<Inner>> nested = {};
    nested["a"]["b"]["y"] = 6;
    io:println(nested); // @output {"a":{"b":{"y":6}}}

    map<R> rm = {};
    rm["a"]["r"]["x"] = 1;
    io:println(rm); // @output {"a":{"r":{"x":1}}}
    rm["a"]["r"]["r"]["r"]["x"] = 2;
    io:println(rm); // @output {"a":{"r":{"x":1,"r":{"r":{"x":2}}}}}

    R[2] rArr = [];
    rArr[0]["r"]["x"] = 1;
    io:println(rArr); // @output [{"r":{"x":1}},{}]
}
