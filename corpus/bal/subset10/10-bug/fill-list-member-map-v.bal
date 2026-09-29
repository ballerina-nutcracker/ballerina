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

type Holder record {|
    map<int> m?;
|};

type S record {|
    S s?;
    map<int> m?;
|};

public function main() {
    Holder[] grown = [];
    grown[1]["m"]["k"] = 1;
    io:println(grown); // @output [{},{"m":{"k":1}}]

    Holder[2] fixed = [];
    fixed[0]["m"]["k"] = 2;
    io:println(fixed); // @output [{"m":{"k":2}},{}]

    S[] recursive = [];
    recursive[0]["s"]["s"]["m"]["k"] = 3;
    io:println(recursive); // @output [{"s":{"s":{"m":{"k":3}}}}]

    map<int>[][] lists = [];
    lists[0][1]["k"] = 4;
    io:println(lists); // @output [[{},{"k":4}]]
}
