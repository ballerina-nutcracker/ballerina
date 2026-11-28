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

// The function's workers and the fork's workers form separate peer groups that
// exchange messages independently in the same function.
public function main() {
    worker named returns int {
        int v = <- function;
        return v;
    }
    fork {
        worker a {
            "x" -> b;
        }
        worker b returns string {
            string s = <- a;
            return s;
        }
    }
    7 -> named;
    record {int named; string b;} r = wait {named, b};
    io:println(r); // @output {"named":7,"b":"x"}
}
