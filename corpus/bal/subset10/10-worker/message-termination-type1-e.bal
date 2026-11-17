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

// A returned value whose type is only partly an error can terminate the worker
// with success before its message actions.
function intOrError(int|error v, boolean flag) returns int|error {
    worker a returns boolean {
        int|error r = <- function;
        return r is error;
    }
    if flag {
        return v; // @error v may be an int
    } else {
    }
    1 -> a;
    return 0;
}
