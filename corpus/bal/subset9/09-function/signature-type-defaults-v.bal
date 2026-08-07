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

function regular(function(int value = 11) returns int... callbacks) returns function(int value = 12) returns int {
    var _ = callbacks;
    return function(int value) returns int {
        return value;
    };
}

client class Client {
    resource function get value(function(int value = 13) returns int... callbacks) returns record {| int value = 14; |} {
        var _ = callbacks;
        return {value: 14};
    }
}

public function main() {
    io:println("ok"); // @output ok
}
