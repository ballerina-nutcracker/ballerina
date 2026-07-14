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

type NN record {|
    int x;
    int y;
|};

type SS record {|
    string x;
    string y;
|};

type NS record {|
    int x;
    string y;
|};

type SN record {|
    string x;
    int y;
|};

type UU record {|
    int|string x;
    int|string y;
|};

type U NN|SS|NS|SN;

public function main() {
    SN ns = {x: "str", y: 3};
    U u = ns;

    if u is NN {
        io:println("NN");
    } else if u is SS {
        io:println("SS");
    } else if u is NS {
        io:println("NS");
    } else {
        SN _ = u;
        io:println("SN"); //  @output SN
    }
}
