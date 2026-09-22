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

enum Suit {
    HEARTS,
    SPADES,
    CLUBS
}

enum Narrow {
    HEARTS_ONLY = "HEARTS"
}

const string SUIT_SCHEMA = string
    `{"type": "enum", "name": "Suit", "namespace": "cards", "symbols": ["HEARTS", "SPADES", "CLUBS"]}`;

public function main() returns error? {
    avro:Schema suitSchema = check new (SUIT_SCHEMA);

    // A symbol is one byte: the zigzag-encoded index into the symbol list.
    byte[] first = check suitSchema.toAvro(HEARTS);
    io:println(first.length()); // @output 1
    Suit hearts = check suitSchema.fromAvro(first);
    io:println(hearts); // @output HEARTS

    Suit last = check suitSchema.fromAvro(check suitSchema.toAvro(CLUBS));
    io:println(last); // @output CLUBS
    io:println(last == CLUBS); // @output true

    // A plain string target keeps the symbol as text.
    string asText = check suitSchema.fromAvro(check suitSchema.toAvro(SPADES));
    io:println(asText); // @output SPADES

    // A string outside the schema's symbol list cannot be serialized.
    byte[]|avro:Error unknown = suitSchema.toAvro("DIAMONDS");
    io:println(unknown is avro:Error); // @output true

    // A symbol the target enum does not admit fails on the way back.
    Narrow|avro:Error tooWide = suitSchema.fromAvro(check suitSchema.toAvro(SPADES));
    io:println(tooWide is avro:Error); // @output true
    Narrow admitted = check suitSchema.fromAvro(check suitSchema.toAvro(HEARTS));
    io:println(admitted); // @output HEARTS

    // Enums nest inside records like any other type.
    avro:Schema handSchema = check new (string `{
        "type": "record", "name": "Hand", "namespace": "cards",
        "fields": [{"name": "suit", "type": ${SUIT_SCHEMA}}]}`);
    map<string> hand = check handSchema.fromAvro(check handSchema.toAvro({suit: "CLUBS"}));
    io:println(hand["suit"]); // @output CLUBS
}
