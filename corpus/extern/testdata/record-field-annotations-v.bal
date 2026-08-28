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

type Meta record {|
    string name;
|};

annotation Meta fieldMeta on record field;
annotation Meta extraMeta on record field;
annotation marker on record field;
// A source-only annotation is compile-time metadata: it must not be visible at
// runtime even though it is attached to a record field.
const annotation Meta sourceOnlyMeta on source record field;

function runtimeName() returns string {
    return "computed";
}

type Person record {|
    @fieldMeta {name: "person-name"}
    string name;
    @fieldMeta {name: "person-age"}
    @extraMeta {name: "second"}
    int age;
    @marker
    boolean active?;
    @fieldMeta {name: runtimeName()}
    string dynamic;
    @sourceOnlyMeta {name: "compile-time-only"}
    string sourceAnnotated;
    string plain;
|};

type Plain record {|
    int x;
|};

public function main() {
    io:println(annotatedFields(Person)); //@output active,age,dynamic,name
    io:println(annotatedFields(Plain)); //@output
    io:println(annotatedFields(int)); //@output
    io:println(fieldMetaName(Person, "name")); //@output person-name
    io:println(fieldMetaName(Person, "age")); //@output person-age
    io:println(fieldMetaName(Person, "dynamic")); //@output computed
    io:println(fieldMetaName(Person, "plain")); //@output <absent>
    io:println(fieldMetaName(Person, "missing")); //@output <absent>
    io:println(extraMetaName(Person, "age")); //@output second
    io:println(hasMarker(Person, "active")); //@output true
    io:println(hasMarker(Person, "name")); //@output false
    io:println(fieldMetaName(Person, "sourceAnnotated")); //@output <absent>
}

function annotatedFields(typedesc<anydata> td) returns string = external;

function fieldMetaName(typedesc<anydata> td, string fieldName) returns string = external;

function extraMetaName(typedesc<anydata> td, string fieldName) returns string = external;

function hasMarker(typedesc<anydata> td, string fieldName) returns boolean = external;
