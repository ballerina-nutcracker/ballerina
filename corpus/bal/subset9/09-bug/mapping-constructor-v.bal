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

type ZoneOffset readonly & record {
    int hours?;
    int minutes?;
};

type RequiredX record {
    int x;
};

type OptionalXRequiredY record {
    int x?;
    int y;
};

type RequiredXOptionalY record {
    int x;
    int y?;
};

public function main() {
    ZoneOffset z = {hours: 0};
    io:println(z); // @output {"hours":0}

    RequiredX|OptionalXRequiredY u1 = {y: 1};
    io:println(u1); // @output {"y":1}

    OptionalXRequiredY|RequiredXOptionalY u2 = {x: 1};
    io:println(u2); // @output {"x":1}
}
