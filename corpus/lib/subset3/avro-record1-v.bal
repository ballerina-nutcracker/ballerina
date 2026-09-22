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

type Inner record {|
    string label;
|};

type Outer record {|
    int id;
    Inner inner;
|};

type Open record {
    int id;
    string note?;
};

type Node record {|
    int value;
    Node? next;
|};

const string NESTED_SCHEMA = string `{
    "type": "record", "name": "Outer", "namespace": "demo",
    "fields": [
        {"name": "id", "type": "int"},
        {"name": "inner", "type": {"type": "record", "name": "Inner",
            "fields": [{"name": "label", "type": "string"}]}}
    ]}`;

public function main() returns error? {
    avro:Schema nested = check new (NESTED_SCHEMA);
    Outer nestedValue = {id: 7, inner: {label: "deep"}};
    Outer back = check nested.fromAvro(check nested.toAvro(nestedValue));
    io:println(back.id); // @output 7
    io:println(back.inner.label); // @output deep

    // A schema with fewer fields than the target record still binds, because
    // the target's remaining fields are optional or defaulted.
    avro:Schema idOnly = check new (string `{
        "type": "record", "name": "IdOnly", "fields": [{"name": "id", "type": "int"}]}`);
    Open open = check idOnly.fromAvro(check idOnly.toAvro({id: 3}));
    io:println(open.id); // @output 3
    io:println(open.note is ()); // @output true

    // Fields the schema does not declare are ignored on the way out.
    byte[] ignoring = check idOnly.toAvro({id: 4, extra: "dropped"});
    Open trimmed = check idOnly.fromAvro(ignoring);
    io:println(trimmed.id); // @output 4
    io:println(trimmed["extra"] is ()); // @output true

    // A recursive schema keeps a name reference on the wire.
    avro:Schema nodeSchema = check new (string `{
        "type": "record", "name": "Node", "namespace": "demo",
        "fields": [
            {"name": "value", "type": "int"},
            {"name": "next", "type": ["null", "Node"]}
        ]}`);
    Node chain = {value: 1, next: {value: 2, next: ()}};
    Node backChain = check nodeSchema.fromAvro(check nodeSchema.toAvro(chain));
    io:println(backChain.value); // @output 1
    Node? tail = backChain.next;
    if tail is Node {
        io:println(tail.value); // @output 2
        io:println(tail.next is ()); // @output true
    }

    // Field order follows the schema, not the Ballerina value.
    avro:Schema ordered = check new (string `{
        "type": "record", "name": "Ordered", "fields": [
            {"name": "b", "type": "string"},
            {"name": "a", "type": "string"}]}`);
    map<string> pair = check ordered.fromAvro(check ordered.toAvro({a: "first", b: "second"}));
    io:println(pair["b"]); // @output second
    io:println(pair["a"]); // @output first

    // A record field the value is missing has nothing to encode.
    avro:Schema needsBoth = check new (string `{
        "type": "record", "name": "Both", "fields": [
            {"name": "a", "type": "string"},
            {"name": "b", "type": "int"}]}`);
    byte[]|avro:Error incomplete = needsBoth.toAvro({a: "only"});
    io:println(incomplete is avro:Error); // @output true

    // A dotted name carries its own namespace, with no separate "namespace" key.
    avro:Schema dotted = check new (string `{"type": "fixed", "name": "demo.dotted.Hash", "size": 2}`);
    byte[] dottedSource = [9, 8];
    byte[] dottedBytes = check dotted.toAvro(dottedSource);
    io:println(dottedBytes.length()); // @output 2

    // A recursive schema with no namespace at all resolves the self-reference
    // as a bare name, not through the namespace-abbreviation fallback.
    avro:Schema plainNode = check new (string `{
        "type": "record", "name": "PlainNode",
        "fields": [
            {"name": "value", "type": "int"},
            {"name": "next", "type": ["null", "PlainNode"]}]}`);
    map<anydata> plain = check plainNode.fromAvro(check plainNode.toAvro({value: 1, next: ()}));
    io:println(plain["value"]); // @output 1
}
