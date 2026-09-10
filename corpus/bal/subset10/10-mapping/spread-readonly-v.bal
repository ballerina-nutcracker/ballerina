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

type Point record {|
    int x;
    int y;
|};

type ReadonlyPoint readonly & Point;

type Inner record {|
    int v;
|};

type Wrapper record {|
    Inner inner;
|};

public function main() {
    Point mutable = {x: 1, y: 2};
    ReadonlyPoint frozen = {...mutable};
    io:println(frozen); // @output {"x":1,"y":2}
    io:println(frozen is readonly); // @output true

    io:println(mutable is readonly); // @output false
    mutable.x = 5;
    io:println(mutable.x); // @output 5
    io:println(frozen.x); // @output 1

    ReadonlyPoint src = {x: 3, y: 4};
    var copied = {...src};
    io:println(copied is readonly); // @output false
    io:println(copied); // @output {"x":3,"y":4}

    readonly & Inner roInner = {v: 1};
    readonly & Wrapper roWrapper = {inner: roInner};
    readonly & Wrapper rebuilt = {...roWrapper};
    io:println(rebuilt); // @output {"inner":{"v":1}}
    io:println(rebuilt is readonly); // @output true
}
