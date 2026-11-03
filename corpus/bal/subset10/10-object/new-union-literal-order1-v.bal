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

class Decimal {
    decimal value;

    function init(decimal value) {
        self.value = value;
    }
}

class Float {
    float value;

    function init(float value, int marker) {
        self.value = value;
        var _ = marker;
    }
}

class HexFloat {
    float value;

    function init(float value, string marker) {
        self.value = value;
        var _ = marker;
    }
}

class HexDecimal {
    decimal value;

    function init(decimal value, int marker) {
        self.value = value;
        var _ = marker;
    }
}

public function main() {
    Decimal|Float d = new (1.50);
    Float|Decimal e = new (1.50);
    if d is Decimal && e is Decimal {
        io:println(d.value, " ", e.value); // @output 1.50 1.50
    }
    Decimal|Float f = new (1.50, 1);
    Float|Decimal g = new (1.50, 1);
    if f is Float && g is Float {
        io:println(f.value, " ", g.value); // @output 1.5 1.5
    }
    HexFloat|HexDecimal h = new (0x5, "x");
    HexDecimal|HexFloat i = new (0x5, 1);
    if h is HexFloat && i is HexDecimal {
        io:println(h.value, " ", i.value); // @output 5.0 5
    }
}
