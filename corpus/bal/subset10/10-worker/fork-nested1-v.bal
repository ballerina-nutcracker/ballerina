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

function innerWaitsOnOuterSibling() {
    fork {
        worker host returns int {
            fork {
                worker inner returns int {
                    int s = wait sib;
                    return s + 1;
                }
            }
            return wait inner;
        }
        worker sib returns int {
            return 10;
        }
    }
    int r = wait host;
    io:println(r); // @output 11
}

function messagesAtBothLevels() {
    fork {
        worker host returns int {
            int base = <- sib;
            fork {
                worker x {
                    base -> y;
                }
                worker y returns int {
                    int v = <- x;
                    return v * 2;
                }
            }
            return wait y;
        }
        worker sib {
            21 -> host;
        }
    }
    int r = wait host;
    io:println(r); // @output 42
}

public function main() {
    innerWaitsOnOuterSibling();
    messagesAtBothLevels();
}
