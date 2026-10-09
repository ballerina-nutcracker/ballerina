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
    int value;

    function init(int value) {
        self.value = value;
    }
}

class StringBox {
    string value;

    function init(string value) {
        self.value = value;
    }
}

class IntOuter {
    IntBox|StringBox inner;

    function init(IntBox|StringBox inner, int marker) {
        self.inner = inner;
        var _ = marker;
    }
}

class StringOuter {
    IntBox|StringBox inner;

    function init(IntBox|StringBox inner, string marker) {
        self.inner = inner;
        var _ = marker;
    }
}

public function main() {
    IntOuter|StringOuter o = new (new (5), 1);
    if o is IntOuter {
        io:println(o.inner is IntBox); // @output true
    }
    IntOuter|StringOuter s = new (new ("five"), "x");
    if s is StringOuter {
        io:println(s.inner is StringBox); // @output true
    }
}
