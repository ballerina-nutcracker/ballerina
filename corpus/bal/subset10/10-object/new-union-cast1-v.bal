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

class Fixed {
    int[2] values;
    int length;

    function init(int[2] values, int length, () pushed) {
        self.values = values;
        self.length = length;
        var _ = pushed;
    }
}

class Text {
    string values;

    function init(string values, int length, () pushed) {
        self.values = values;
        var _ = length;
        var _ = pushed;
    }
}

public function main() {
    int[2] fixed = [1, 2];
    anydata x = fixed;
    string s = "abc";
    int[] arr = [];
    Fixed|Text f = new (<int[2]>x, s.length(), arr.push(3));
    if f is Fixed {
        io:println(f.values, " ", f.length, " ", arr); // @output [1,2] 3 [3]
    }
}
