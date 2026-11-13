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

type OpenDetail record {
    int count?;
};

type OpenErr error<OpenDetail>;

type MapErr error<map<int>>;

type RestOnlyDetail record {|
    string...;
|};

type QuotedDetail record {
    int 'type?;
};

public function main() {
    error<OpenDetail> e1 = error("a", count = 1);
    io:println(e1); // @output error("a",count=1)
    OpenErr e2 = error OpenErr("b");
    io:println(e2); // @output error("b")
    MapErr e3 = error MapErr("c", x = 1, y = 2);
    io:println(e3); // @output error("c",x=1,y=2)
    error e4 = error("d", anything = true);
    io:println(e4); // @output error("d",anything=true)
    error<RestOnlyDetail> e5 = error("e", p = "x", q = "y");
    io:println(e5); // @output error("e",p="x",q="y")
    error<QuotedDetail> e6 = error("f", 'type = 1);
    io:println(e6); // @output error("f",type=1)
}
