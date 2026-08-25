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

import ballerina/avro;
import ballerina/io;

type Item record {|
    string sku;
    int qty;
|};

public function main() returns error? {
    avro:Schema intMap = check new (string `{"type": "map", "values": "int"}`);
    map<int> counts = check intMap.fromAvro(check intMap.toAvro({"a": 1, "b": 2}));
    io:println(counts.length()); // @output 2
    io:println(counts["a"]); // @output 1
    io:println(counts["b"]); // @output 2

    // An empty map is a single zero-count block.
    byte[] emptyBytes = check intMap.toAvro({});
    io:println(emptyBytes.length()); // @output 1
    map<int> empty = check intMap.fromAvro(emptyBytes);
    io:println(empty.length()); // @output 0

    avro:Schema strMap = check new (string `{"type": "map", "values": "string"}`);
    map<string> labels = check strMap.fromAvro(check strMap.toAvro({"k": "v"}));
    io:println(labels["k"]); // @output v

    // Maps of records.
    avro:Schema itemMap = check new (string `{
        "type": "map",
        "values": {"type": "record", "name": "Item",
                   "fields": [{"name": "sku", "type": "string"},
                              {"name": "qty", "type": "int"}]}}`);
    map<Item> stock = {"x": {sku: "a", qty: 7}};
    map<Item> backStock = check itemMap.fromAvro(check itemMap.toAvro(stock));
    Item? stocked = backStock["x"];
    if stocked is Item {
        io:println(stocked.qty); // @output 7
        io:println(stocked.sku); // @output a
    }

    // Nested maps.
    avro:Schema nestedMap = check new (string
        `{"type": "map", "values": {"type": "map", "values": "long"}}`);
    map<map<int>> nested = {"outerKey": {"innerKey": 5}};
    map<map<int>> backNested = check nestedMap.fromAvro(check nestedMap.toAvro(nested));
    map<int>? inner = backNested["outerKey"];
    if inner is map<int> {
        io:println(inner["innerKey"]); // @output 5
    }

    // Maps of arrays.
    avro:Schema arrayMap = check new (string
        `{"type": "map", "values": {"type": "array", "items": "string"}}`);
    map<string[]> groups = check arrayMap.fromAvro(check arrayMap.toAvro({"g": ["one", "two"]}));
    string[]? members = groups["g"];
    if members is string[] {
        io:println(members.length()); // @output 2
        io:println(members[0]); // @output one
    }

    // A non-mapping value cannot fill a map schema.
    byte[]|avro:Error notAMap = intMap.toAvro([1, 2]);
    io:println(notAMap is avro:Error); // @output true
}
