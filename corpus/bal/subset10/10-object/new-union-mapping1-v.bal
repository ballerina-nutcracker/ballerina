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

class DecMap {
    map<decimal> values;

    function init(map<decimal> values, int marker) {
        self.values = values;
        var _ = marker;
    }
}

class FloatMap {
    map<float> values;

    function init(map<float> values, string marker) {
        self.values = values;
        var _ = marker;
    }
}

type Point record {|
    decimal x;
    decimal y = 2.5;
|};

class PointHolder {
    Point point;

    function init(Point point, int marker) {
        self.point = point;
        var _ = marker;
    }
}

class StringHolder {
    map<string> point;

    function init(map<string> point, string marker) {
        self.point = point;
        var _ = marker;
    }
}

public function main() {
    DecMap|FloatMap m = new ({a: 1.50}, 1);
    if m is DecMap {
        io:println(m.values["a"]); // @output 1.50
    }
    decimal x = 1.25;
    PointHolder|StringHolder p = new ({x}, 0);
    if p is PointHolder {
        io:println(p.point.x, " ", p.point.y); // @output 1.25 2.5
    }
}
