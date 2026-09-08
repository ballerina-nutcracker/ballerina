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

type S record {|
    future<int> fu;
|};

function inc() returns int => 1;

function die() returns never {
    panic error("unreachable");
}

public function main() {
    future<int> f = start inc();
    S x = {readonly fu: f}; // @error future is never readonly
    io:println(x.fu is future<int>);
    S y = {readonly fu: die()}; // @error never value does not make the field readonly
    io:println(y.fu is future<int>);
}
