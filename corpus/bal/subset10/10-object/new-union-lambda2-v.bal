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

class DecFn {
    decimal value;

    function init(function () returns decimal callback, int marker) {
        self.value = callback() + <decimal>marker;
    }
}

class FloatFn {
    float value;

    function init(function () returns float callback, string marker) {
        self.value = callback();
        var _ = marker;
    }
}

class IntOnly {
    int value;

    function init(function () returns int callback, int arg) {
        self.value = callback() + arg;
    }
}

class IntOrString {
    int|string value;

    function init(function () returns int callback, int|string arg) {
        self.value = arg;
        var _ = callback;
    }
}

public function main() {
    DecFn|FloatFn d = new (function () returns decimal {
        decimal base = 1.00;
        function () returns decimal f = () => 1.50d;
        return base + f();
    }, 1);
    if d is DecFn {
        io:println(d.value); // @output 3.50
    }

    int|string x = 5;
    if x is int {
        // The lambda captures the narrowed x, so x is int|string again for the next argument.
        IntOnly|IntOrString c = new (function () returns int {
            return x is int ? 1 : 0;
        }, x);
        io:println(c is IntOrString); // @output true
    }

    int|string y = 5;
    if y is int {
        // A nested parameter default captures y too, so y is int|string again for the next argument.
        IntOnly|IntOrString n = new (function () returns int {
            var g = function (int|string a = y) returns int {
                return a is int ? a : 0;
            };
            return g();
        }, y);
        io:println(n is IntOrString); // @output true
    }
}
