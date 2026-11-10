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

// The flush returns once the receiver took the message sent before it.
public function main() {
    worker sender returns error? {
        1 -> receiver;
        error? e = flush receiver;
        return e;
    }
    worker receiver returns int|error {
        return check <- sender;
    }
    int|error r = wait receiver;
    io:println(r); // @output 1
    error? s = wait sender;
    io:println(s is error ? "failed" : "flushed"); // @output flushed
}
