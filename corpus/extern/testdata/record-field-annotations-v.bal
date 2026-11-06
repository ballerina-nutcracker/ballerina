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

// Same shape as Person but without its annotations.
type PersonTwin record {|
    string name;
    int age;
    boolean active?;
    string dynamic;
    string sourceAnnotated;
    string plain;
|};

Person|error moduleInferred = annotatedFieldsInferred();

Person|error moduleAssigned = error("unset");

class Holder {
    Person|error person = annotatedFieldsInferred();

    function describe(Person|error person) returns string {
        return fieldsOf(person);
    }
}

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
    Person|error inferred = annotatedFieldsInferred();
    io:println(fieldsOf(inferred)); //@output active,age,dynamic,name
    io:println(fieldsOf(moduleInferred)); //@output active,age,dynamic,name
    error? checked = inferredThroughCheck();
    io:println(checked is error ? checked.message() : ""); //@output active,age,dynamic,name
    Person|PersonTwin|error ambiguous = annotatedFieldsInferred();
    io:println(ambiguousFieldsOf(ambiguous)); //@output
    Person|error assigned;
    assigned = annotatedFieldsInferred();
    io:println(fieldsOf(assigned)); //@output active,age,dynamic,name
    moduleAssigned = annotatedFieldsInferred();
    io:println(fieldsOf(moduleAssigned)); //@output active,age,dynamic,name
    io:println(fieldsOf(annotatedFieldsInferred())); //@output active,age,dynamic,name
    io:println(fieldsOf(value = annotatedFieldsInferred())); //@output active,age,dynamic,name
    io:println(ambiguousFieldsOf(annotatedFieldsInferred())); //@output
    io:println(fieldsOf(returnedInferred())); //@output active,age,dynamic,name
    io:println(fieldsOf(exprBodyInferred())); //@output active,age,dynamic,name
    function () returns Person|error lambda = function() returns Person|error {
        return annotatedFieldsInferred();
    };
    io:println(fieldsOf(lambda())); //@output active,age,dynamic,name
    function (Person|error) returns string fnValue = fieldsOf;
    io:println(fnValue(annotatedFieldsInferred())); //@output active,age,dynamic,name
    Holder holder = new;
    io:println(fieldsOf(holder.person)); //@output active,age,dynamic,name
    io:println(holder.describe(annotatedFieldsInferred())); //@output active,age,dynamic,name
    boolean flag = true;
    Person|error chosen = flag ? annotatedFieldsInferred() : error("unused");
    io:println(fieldsOf(chosen)); //@output active,age,dynamic,name
    Person|error grouped = (annotatedFieldsInferred());
    io:println(fieldsOf(grouped)); //@output active,age,dynamic,name
}

function inferredThroughCheck() returns error? {
    Person person = check annotatedFieldsInferred();
    _ = person;
}

function fieldsOf(Person|error value) returns string {
    return value is error ? value.message() : "";
}

function ambiguousFieldsOf(Person|PersonTwin|error value) returns string {
    return value is error ? value.message() : "";
}

function returnedInferred() returns Person|error {
    return annotatedFieldsInferred();
}

function exprBodyInferred() returns Person|error => annotatedFieldsInferred();

function annotatedFields(typedesc<anydata> td) returns string = external;

function annotatedFieldsInferred(typedesc<anydata> td = <>) returns td|error = external;

function fieldMetaName(typedesc<anydata> td, string fieldName) returns string = external;

function extraMetaName(typedesc<anydata> td, string fieldName) returns string = external;

function hasMarker(typedesc<anydata> td, string fieldName) returns boolean = external;
