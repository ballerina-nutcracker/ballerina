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

class Named {
    function init(string name, int count = 1) {
        var _ = name;
        var _ = count;
    }
}

class ByID {
    function init(int id, string label = "id") {
        var _ = id;
        var _ = label;
    }
}

class Required {
    function init(int value) { var _ = value; }
}

class Defaulted {
    function init(string value = "default") { var _ = value; }
}

type Config record {|
    string path;
    boolean recursive = false;
|};

class Configured {
    function init(*Config config) { var _ = config; }
}

class Numeric {
    function init(int id) { var _ = id; }
}

class DecimalFirst {
    function init(decimal value, int marker = 0) {
        var _ = value;
        var _ = marker;
    }
}

class FloatSecond {
    function init(float value, string marker) {
        var _ = value;
        var _ = marker;
    }
}

class InnerA {}
class InnerB {}

class OuterA {
    function init(InnerA value, int marker) {
        var _ = value;
        var _ = marker;
    }
}

class OuterB {
    function init(InnerB value, string marker) {
        var _ = value;
        var _ = marker;
    }
}

public function main() {
    Named|ByID named = new (name = "value");
    io:println(named is Named); // @output true

    Required|Defaulted defaulted = new ();
    io:println(defaulted is Defaulted); // @output true

    Configured|Numeric configured = new (path = "/tmp");
    io:println(configured is Configured); // @output true

    DecimalFirst|FloatSecond literal = new (1, "float");
    io:println(literal is FloatSecond); // @output true

    OuterA|OuterB nested = new (new (), "outer");
    io:println(nested is OuterB); // @output true

    Named|ByID explicit = new Named("name");
    io:println(explicit is Named); // @output true

    Named|ByID positional = new (1);
    io:println(positional is ByID); // @output true
}
