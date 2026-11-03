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

class DecList {
    decimal[] values;

    function init(decimal[] values, int marker) {
        self.values = values;
        var _ = marker;
    }
}

class FloatList {
    float[] values;

    function init(float[] values, string marker) {
        self.values = values;
        var _ = marker;
    }
}

class Pair {
    [decimal, string] pair;

    function init([decimal, string] pair, int marker) {
        self.pair = pair;
        var _ = marker;
    }
}

class Strings {
    string[] values;

    function init(string[] values, string marker) {
        self.values = values;
        var _ = marker;
    }
}

public function main() {
    DecList|FloatList l = new ([1.50], 1);
    if l is DecList {
        io:println(l.values[0]); // @output 1.50
    }
    Pair|Strings p = new ([2.50, "two"], 0);
    io:println(p is Pair); // @output true
    decimal[] rest = [3.5, 4.5];
    DecList|FloatList s = new ([1.50, ...rest], 2);
    if s is DecList {
        io:println(s.values.length()); // @output 3
    }
}
