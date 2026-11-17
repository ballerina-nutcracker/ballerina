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


function failWith(string message) returns error? {
    return error(message);
}

function boom() {
    panic error("c panicked"); // @panic
}

// b fails and c panics before receiving; like flush, the delivery at
// termination prefers the panic.
public function main() {
    worker b returns error? {
        check failWith("b failed");
        int _ = check <- function;
    }
    worker c {
        boom();
        int _ = <- function;
    }
    1 -> b;
    2 -> c;
    error? e = flush b;
    boolean _ = e is error;
}
