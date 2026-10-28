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

// Workers of different forks of one function may share a name: their
// generated functions are named after their fork.
function twoBlocks(boolean flag) returns string {
    xml x = xml `<a><b>one</b></a>`;
    if flag {
        fork {
            worker w returns string {
                (x/<b>).toString() -> v;
                return "first";
            }
            worker v returns string {
                return <- w;
            }
        }
        string a = wait w;
        string b = wait v;
        return a + ":" + b;
    }
    fork {
        worker w returns string {
            (x/<a>).toString() -> v;
            return "second";
        }
        worker v returns string {
            return <- w;
        }
    }
    string a = wait w;
    string b = wait v;
    return a + ":" + b;
}

function lambdaAndEnclosing() returns int {
    var g = function() returns int {
        fork {
            worker w returns int {
                return 1;
            }
        }
        return wait w;
    };
    fork {
        worker w returns int {
            return 2;
        }
    }
    int a = wait w;
    return a + g();
}

public function main() {
    io:println(twoBlocks(true)); // @output first:<b>one</b>
    io:println(twoBlocks(false)); // @output second:
    io:println(lambdaAndEnclosing()); // @output 3
}
