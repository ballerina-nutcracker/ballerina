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

import crossmoduledependentfn.http;

import ballerina/io;

public function main() {
    http:Client c = new ("http://foo");
    string res1 = checkpanic c->get("bar");
    io:println(res1); // @output string response
    int res2 = checkpanic c->get("bar");
    io:println(res2); // @output 2
    int res3 = checkpanic c->get("bar");
    io:println(res3); // @output 2
}
