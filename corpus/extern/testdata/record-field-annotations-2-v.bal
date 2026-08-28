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
annotation Meta[] repeatable on record field;

type Base record {|
    @fieldMeta {name: "base-id"}
    string id;
|};

// A field pulled in through a type inclusion keeps the annotations declared on
// it. jBallerina drops them at the inclusion boundary; we inherit them.
type Derived record {|
    *Base;
    @fieldMeta {name: "derived-extra"}
    int extra;
|};

// Root <- Middle <- Leaf: annotations travel the whole chain.
type Root record {|
    @fieldMeta {name: "root-tag"}
    string tag;
|};

type Middle record {|
    *Root;
    @fieldMeta {name: "middle-mid"}
    int mid;
|};

type Leaf record {|
    *Middle;
    @fieldMeta {name: "leaf-own"}
    boolean own;
|};

// Both branches of the diamond reach Root. A field reachable through more than
// one inclusion has to be overridden, so the local declaration decides which
// annotation applies.
type SideA record {|
    *Root;
|};

type SideB record {|
    *Root;
|};

type Diamond record {|
    *SideA;
    *SideB;
    @fieldMeta {name: "diamond-tag"}
    string tag;
|};

// An annotation declared on the including record wins over the inherited one.
type Overriding record {|
    *Root;
    @fieldMeta {name: "overridden"}
    string tag;
|};

// A record alias carries the aliased record's field annotations, through a
// chain of aliases as well.
type Alias Base;

type AliasOfAlias Alias;

// A field the including record declares itself is a new declaration: it carries
// exactly the annotations written on it, and none of the inherited ones.
type Shadowing record {|
    *Base;
    string id;
|};

type Repeated record {|
    @repeatable {name: "one"}
    @repeatable {name: "two"}
    string tags;
|};

type WithRest record {|
    @fieldMeta {name: "known"}
    string known;
    json...;
|};

public function main() {
    io:println(annotatedFields(Base)); //@output id
    io:println(annotatedFields(Derived)); //@output extra,id
    io:println(fieldMetaName(Derived, "id")); //@output base-id
    io:println(fieldMetaName(Derived, "extra")); //@output derived-extra
    io:println(annotatedFields(Leaf)); //@output mid,own,tag
    io:println(fieldMetaName(Leaf, "tag")); //@output root-tag
    io:println(fieldMetaName(Leaf, "mid")); //@output middle-mid
    io:println(annotatedFields(Diamond)); //@output tag
    io:println(fieldMetaName(Diamond, "tag")); //@output diamond-tag
    io:println(fieldMetaName(Overriding, "tag")); //@output overridden
    io:println(fieldMetaName(Alias, "id")); //@output base-id
    io:println(fieldMetaName(AliasOfAlias, "id")); //@output base-id
    io:println(annotatedFields(Shadowing)); //@output
    io:println(fieldMetaName(Shadowing, "id")); //@output <absent>
    io:println(repeatableNames(Repeated, "tags")); //@output one|two
    io:println(annotatedFields(WithRest)); //@output known
}

function annotatedFields(typedesc<anydata> td) returns string = external;

function fieldMetaName(typedesc<anydata> td, string fieldName) returns string = external;

function repeatableNames(typedesc<anydata> td, string fieldName) returns string = external;
