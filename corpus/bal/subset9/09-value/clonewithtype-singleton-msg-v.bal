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

type F 2.0;
type D 2.5d;

public function main() {
    anydata a = 3.0;
    F|error f = a.cloneWithType(F);
    io:println(f is error ? f.message() : "ok"); // @output '3.0' value cannot be converted to '2.0'
    anydata b = 3.5d;
    D|error d = b.cloneWithType(D);
    io:println(d is error ? d.message() : "ok"); // @output '3.5d' value cannot be converted to '2.5d'
}
