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

function observe(future<int> f) returns int {
    return f is future<int> ? 1 : 2;
}

// The collection of a query's first from clause is evaluated once, so a single
// reference to a worker there is not a repeated reference.
public function main() {
    worker w returns int {
        return 1;
    }
    int[] r = from int i in [observe(w), 5]
        select i;
    io:println(r); // @output [1,5]
}
