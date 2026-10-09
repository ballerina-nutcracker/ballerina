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

class IntArgs {
    int value;
    boolean flag;

    function init(int value, boolean flag) {
        self.value = value;
        self.flag = flag;
    }
}

class StringArgs {
    string value;
    boolean flag;

    function init(string value, boolean flag) {
        self.value = value;
        self.flag = flag;
    }
}

public function main() {
    int|string x = 1;
    boolean y = true;
    boolean b = false;
    boolean c = true;
    IntArgs|StringArgs a = new (x is int ? x : 0, x == 1 && y);
    if a is IntArgs {
        io:println(a.value, " ", a.flag); // @output 1 true
    }
    IntArgs|StringArgs o = new (x is string ? x : "none", b || c);
    if o is StringArgs {
        io:println(o.value, " ", o.flag); // @output none true
    }
    final int|string fx = 1;
    if fx is int {
        function () returns boolean f = function () returns boolean {
            IntArgs|StringArgs inner = new (fx is int ? fx : 0, true);
            return inner is IntArgs;
        };
        io:println(f()); // @output true
    }
    int|string z = "q";
    IntArgs|StringArgs t = new (z is int ? "a" : z, z is string && z.length() > 0);
    if t is StringArgs {
        io:println(t.value, " ", t.flag); // @output q true
    }
    error|string u = error("failed");
    IntArgs|StringArgs m = new (u is error ? u.message() : u, true);
    if m is StringArgs {
        io:println(m.value); // @output failed
    }
    int i = 0;
    while i < 1 {
        IntArgs|StringArgs w = new (i, i == 0);
        io:println(w is IntArgs); // @output true
        i += 1;
    }
}
