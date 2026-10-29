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

type Size "small"|"medium"|"large"|"xlarge";

function size(string s) returns Size|error {
    if s == "xl" || s == "xlarge" {
        return "xlarge";
    }
    if s == "small" || s == "medium" || s == "large" {
        return s;
    }
    return error("unknown");
}

function sizeNotEqual(string s) returns Size|error {
    if s != "xl" && s != "xlarge" {
        if s != "small" && s != "medium" && s != "large" {
            return error("unknown");
        }
        return s;
    }
    return "xlarge";
}

public function main() {
    io:println(size("medium"), " ", size("xl")); // @output medium xlarge
    io:println(sizeNotEqual("large"), " ", sizeNotEqual("xlarge")); // @output large xlarge
}
