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

import ballerina/data.xmldata;
import ballerina/io;

type Open record {
    string id;
};

type StringRest record {|
    string id;
    string...;
|};

type IntRest record {|
    string id;
    int...;
|};

type RecordRest record {|
    string id;
    record {| string v; |}...;
|};

type ArrayRest record {|
    string id;
    string[]...;
|};

type OnlyRest record {|
    string...;
|};

type JsonRest record {|
    string id;
    json...;
|};

type Maps record {|
    map<string> m;
    map<int> n;
|};

type MapArray record {|
    map<string>[] m;
|};

type Anydata record {|
    anydata data;
|};

type JsonArray record {|
    json[] items;
|};

type Mixed record {|
    string b;
|};

// Messages that name a record type are cut before the type: type names are printed differently.
function show(anydata|error v) {
    if v is error {
        string message = v.message();
        int? typeStart = message.indexOf(" in record '");
        int? nsStart = message.indexOf(" for the type: '");
        int? intoStart = message.indexOf(" into the type '");
        int? cut = typeStart ?: nsStart ?: intoStart;
        io:println("E: ", cut is int ? message.substring(0, cut) : message);
    } else {
        io:println(v);
    }
}

function both(string s, typedesc<record {}> td, xmldata:SourceOptions o = {}) {
    record {}|error parsed = xmldata:parseString(s, o, td);
    show(parsed);
    xml|error x = xml:fromString(s);
    record {}|error traversed = x is xml ? xmldata:parseAsType(x, o, td) : x;
    show(traversed);
}

public function main() {
    both("<r><id>1</id><x>t</x><y><z>2</z><w k=\"v\">3</w></y><y><z>4</z></y><q a=\"1\">txt</q></r>", Open);
    // @output {"id":"1","x":"t","y":[{"z":2,"w":{"k":"v","#content":3}},{"z":4}],"q":{"a":1,"#content":"txt"}}
    // @output {"id":"1","x":"t","y":[{"z":2,"w":{"k":"v","#content":3}},{"z":4}],"q":{"a":1,"#content":"txt"}}
    both("<r><id>1</id><x>t</x><x>u</x><y a=\"1\"/><y a=\"2\">v</y></r>", Open);
    // @output {"id":"1","x":["t","u"],"y":[{"a":1},"v"]}
    // @output {"id":"1","x":["t","u"],"y":[{"a":1},{"a":2,"#content":"v"}]}
    both("<r><!-- c --><?pi x?><id>1</id><n>1.5</n><b>true</b></r>", Open);
    // @output {"id":"1","n":1.5,"b":true}
    // @output {"id":"1","n":1.5,"b":true}
    both("<r xmlns=\"http://d\" xmlns:p=\"http://p\"><id>1</id><p:x>2</p:x></r>", Open);
    // @output {"id":"1","x":2}
    // @output {"id":"1","x":2}
    both("<r><id>1</id><x>2</x><y>3</y></r>", StringRest);
    // @output {"id":"1","x":"2","y":"3"}
    // @output {"id":"1","x":"2","y":"3"}
    both("<r><id>1</id><x>2</x><x>3</x></r>", StringRest);
    // @output E: expected 'string' value for the field 'x' found 'array' value
    // @output E: expected 'string' value for the field 'x' found 'array' value
    both("<r><id>1</id><x>2</x><y>z</y></r>", IntRest);
    // @output E: 'string' value 'z' cannot be converted to 'int'
    // @output E: 'string' value 'z' cannot be converted to 'int'
    both("<r><id>1</id><x><v>p</v></x><y><v>q</v></y></r>", RecordRest);
    // @output {"id":"1","x":{"v":"p"},"y":{"v":"q"}}
    // @output {"id":"1","x":{"v":"p"},"y":{"v":"q"}}
    both("<r><id>1</id><x>p</x><x>q</x><y>r</y></r>", ArrayRest);
    // @output {"id":"1","x":["p","q"],"y":["r"]}
    // @output {"id":"1","x":["p","q"],"y":["r"]}
    both("<r><id>1</id><x><p>1</p><p>2</p></x><s>9</s></r>", JsonRest);
    // @output {"id":"1","x":{"p":[1,2]},"s":9}
    // @output {"id":"1","x":{"p":[1,2]},"s":9}
    both("<r><m><p>1</p><q>2</q></m><n><r>3</r></n></r>", Maps);
    // @output {"m":{"p":"1","q":"2"},"n":{"r":3}}
    // @output {"m":{"p":"1","q":"2"},"n":{"r":3}}
    both("<r><m><p>x</p></m><n><r>y</r></n></r>", Maps);
    // @output E: 'string' value 'y' cannot be converted to 'int'
    // @output E: 'string' value 'y' cannot be converted to 'int'
    both("<r><m><a>1</a></m><m><b>2</b></m></r>", MapArray);
    // @output {"m":[{"a":"1"},{"b":"2"}]}
    // @output {"m":[{"a":"1"},{"b":"2"}]}
    both("<r><data><a>1</a><b>t</b><b>u</b><c x=\"1\">z</c></data></r>", Anydata);
    // @output {"data":{"a":1,"b":["t","u"],"c":{"x":1,"#content":"z"}}}
    // @output {"data":{"a":1,"b":["t","u"],"c":{"x":1,"#content":"z"}}}
    both("<r><data>plain</data></r>", Anydata);
    // @output {"data":"plain"}
    // @output {"data":"plain"}
    both("<r><data k=\"v\">plain</data></r>", Anydata);
    // @output {"data":{"k":"v","#content":"plain"}}
    // @output {"data":{"k":"v","#content":"plain"}}
    both("<r><items><a>1</a></items><items>2</items></r>", JsonArray);
    // @output {"items":[{"a":1},2]}
    // @output {"items":[{"a":1},2]}
    both("<r><b>x</b><c>y</c></r>", Mixed, {allowDataProjection: false});
    // @output E: undefined field 'c'
    // @output E: undefined field 'c'
    both("<r><a>x</a><b>y</b></r>", OnlyRest);
    // @output {"a":"x","b":"y"}
    // @output {"a":"x","b":"y"}
}
