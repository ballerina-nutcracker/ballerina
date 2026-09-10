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

type OpenTarget record {|
    int x;
    int...;
|};

type Defaulted record {
    int x = 5;
    string y;
};

type Excluded record {|
    never y?;
    int x;
|};

public function main() {
    // An extra named source field is accepted by the target rest.
    record {| int x; int z; |} extra = {x: 1, z: 2};
    OpenTarget open = {...extra};
    io:println(open); // @output {"x":1,"z":2}

    // A default satisfies a required field without the constructor supplying it.
    Defaulted d = {y: "hi"};
    io:println(d); // @output {"y":"hi","x":5}

    // An exclusion keeps the source rest from being checked against the excluded target name.
    Excluded src = {x: 1};
    record {| int x; string y?; |} target = {...src};
    io:println(target); // @output {"x":1}
}
