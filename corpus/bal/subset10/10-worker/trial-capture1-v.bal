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

class IntBox {
    int result;

    function init(int value) {
        self.result = value;
    }
}

class StringBox {
    string result;

    function init(string value) {
        self.result = value;
    }
}

// Worker a tries each class of `new` against arguments that read the captured
// v while worker b and the trailing statements read it too.
function run(int n) returns int {
    int v = n;
    worker a returns int {
        IntBox|StringBox first = new (v);
        IntBox|StringBox second = new (v + 1);
        return first is IntBox && second is IntBox ? first.result + second.result : 0;
    }
    worker b returns int {
        int sum = 0;
        foreach int i in 0 ..< 3 {
            sum += v + i;
        }
        return sum;
    }
    int x = v + v;
    int ra = wait a;
    int rb = wait b;
    return ra + rb + x;
}

public function main() {
    io:println(run(1)); // @output 11
}
