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

type Ann record {|
    string name;
|};

annotation Ann Tag on type;

@Tag {name: "tagged"}
type Tagged record {|
    int id;
|};

class IntArgs {
    record {|int value;|}? next;
    json lax;
    Ann? ann;
    xml filtered;

    function init(record {|int value;|}? next, json lax, Ann? ann, xml filtered, int marker) {
        self.next = next;
        self.lax = lax;
        self.ann = ann;
        self.filtered = filtered;
        var _ = marker;
    }
}

class StringArgs {
    function init(record {|int value;|}? next, json lax, Ann? ann, xml filtered, string marker) {
        var _ = next;
        var _ = lax;
        var _ = ann;
        var _ = filtered;
        var _ = marker;
    }
}

public function main() returns error? {
    stream<int> s = [1, 2].toStream();
    json j = {a: {b: 7}};
    typedesc<Tagged> td = Tagged;
    xml x = xml `<a>1</a><b>2</b>`;
    IntArgs|StringArgs a = new (check s.next(), check j.a.b, td.@Tag, x.<a>, 1);
    if a is IntArgs {
        io:println(a.next, " ", a.lax, " ", a.ann, " ", a.filtered); // @output {"value":1} 7 {"name":"tagged"} <a>1</a>
    }
}
