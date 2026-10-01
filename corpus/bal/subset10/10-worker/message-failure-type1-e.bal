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

function singleReceive() {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    int _ = <- a; // @error the receive's type includes a's failure type
}

function multipleReceive() {
    worker a returns error? {
        check failWith("a failed");
        1 -> function;
    }
    record {int a;} _ = <- {a}; // @error the receive's type includes a's failure type
}

function syncSend() returns error? {
    worker a returns error? {
        check failWith("a failed");
        int _ = check <- function;
    }
    () _ = 1 ->> a; // @error the sync send's type includes a's failure type
    check wait a;
}
