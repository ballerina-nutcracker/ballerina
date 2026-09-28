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

// Paths that leave before reaching a later assignment do not make the variable
// possibly assigned there.
public function main() {
    final int a;
    while true {
        a = 1;
        break;
    }
    io:println(a); // @output 1

    io:println(assignOrReturn(false)); // @output 3

    int n = 2;
    final string b;
    match n {
        1 => {
            b = "one";
        }
        _ => {
            b = "other";
        }
    }
    io:println(b); // @output other

    int i = 0;
    while i < 3 {
        i += 1;
        final int c;
        if i == 2 {
            continue;
        }
        c = i;
        io:println(c); // @output 1
                       // @output 3
    }
}

function assignOrReturn(boolean early) returns int {
    final int x;
    if early {
        x = 2;
        return x;
    }
    x = 3;
    return x;
}
