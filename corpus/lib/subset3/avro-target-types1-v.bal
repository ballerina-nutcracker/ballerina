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

type Pair record {|
    string k;
    int v;
|};

const string PAIR_SCHEMA = string `{
    "type": "record", "name": "Pair", "namespace": "demo", "fields": [
        {"name": "k", "type": "string"},
        {"name": "v", "type": "long"}]}`;

public function main() returns error? {
    avro:Schema pairSchema = check new (PAIR_SCHEMA);
    byte[] encoded = check pairSchema.toAvro({k: "one", v: 1});

    // The target type is inferred from the contextually expected type.
    Pair asRecord = check pairSchema.fromAvro(encoded);
    io:println(asRecord.k); // @output one

    // ... or supplied explicitly by name.
    Pair named = check pairSchema.fromAvro(encoded, targetType = Pair);
    io:println(named.v); // @output 1

    // Wider targets keep the decoded shape without narrowing it.
    json asJson = check pairSchema.fromAvro(encoded);
    io:println(asJson); // @output {"k":"one","v":1}

    anydata asAnydata = check pairSchema.fromAvro(encoded);
    io:println(asAnydata is map<anydata>); // @output true

    map<json> asJsonMap = check pairSchema.fromAvro(encoded);
    io:println(asJsonMap["k"]); // @output one

    map<anydata> asAnydataMap = check pairSchema.fromAvro(encoded);
    io:println(asAnydataMap.length()); // @output 2

    // A nilable target accepts the decoded value unchanged.
    Pair? nilable = check pairSchema.fromAvro(encoded);
    io:println(nilable is Pair); // @output true

    // Scalars narrow to singleton and enum-shaped targets.
    avro:Schema intSchema = check new (string `{"type": "int"}`);
    5 five = check intSchema.fromAvro(check intSchema.toAvro(5));
    io:println(five); // @output 5

    avro:Schema strSchema = check new (string `{"type": "string"}`);
    "yes"|"no" answer = check strSchema.fromAvro(check strSchema.toAvro("yes"));
    io:println(answer); // @output yes

    // An int payload widens into a float target.
    float widened = check intSchema.fromAvro(check intSchema.toAvro(3));
    io:println(widened); // @output 3.0

    // ... and into a decimal target.
    decimal asDecimal = check intSchema.fromAvro(check intSchema.toAvro(3));
    io:println(asDecimal); // @output 3

    // Arrays bind to tuples as well as to arrays.
    avro:Schema pairArray = check new (string `{"type": "array", "items": "int"}`);
    byte[] two = check pairArray.toAvro([1, 2]);
    [int, int] tuple = check pairArray.fromAvro(two);
    io:println(tuple[1]); // @output 2
    int[] list = check pairArray.fromAvro(two);
    io:println(list.length()); // @output 2

    // A readonly & intersection target binds and freezes the decoded value,
    // for a record as well as an array.
    readonly & Pair frozenRecord = check pairSchema.fromAvro(encoded);
    io:println(frozenRecord.k); // @output one
    io:println(frozenRecord is readonly); // @output true

    readonly & int[] frozenArray = check pairArray.fromAvro(two);
    io:println(frozenArray.length()); // @output 2
    io:println(frozenArray is readonly); // @output true
}
