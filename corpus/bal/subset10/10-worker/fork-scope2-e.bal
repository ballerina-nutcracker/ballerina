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

function duplicateSibling() {
    fork {
        worker a {
        }
        worker a { // @error Variable already defined: a
        }
    }
}

function localDeclaredAfterFork() {
    fork {
        worker w {
            io:println(later); // @error later is declared after the fork
        }
    }
    int later = 1;
    io:println(later);
}

function nestedForkWorkerFromOuterSibling() {
    fork {
        worker host returns int {
            fork {
                worker inner returns int {
                    return 1;
                }
            }
            return wait inner;
        }
        worker sib returns int {
            return wait inner; // @error inner is only in scope in host after the nested fork
        }
    }
}
