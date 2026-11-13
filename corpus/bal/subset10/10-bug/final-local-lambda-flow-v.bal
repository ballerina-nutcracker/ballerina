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

function (boolean) returns int choose = function (boolean flag) returns int {
    final int x;
    if flag {
        x = 10;
    } else {
        x = 20;
    }
    return x;
};

public function main() {
    io:println(choose(true)); // @output 10
    io:println(choose(false)); // @output 20
    var enclosing = function () returns int {
        final int x;
        x = 3;
        var inner = function () returns int {
            final int y;
            y = 4;
            return y;
        };
        return x + inner();
    };
    io:println(enclosing()); // @output 7
    var loop = function () returns int {
        int total = 0;
        foreach int i in 1 ... 3 {
            final int x;
            x = i;
            total += x;
        }
        return total;
    };
    io:println(loop()); // @output 6
}
