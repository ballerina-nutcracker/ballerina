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
import recordfieldannotations.types;

// A record that includes a type from another module inherits that type's field
// annotations, which reach us through the symbol pool rather than the AST.
type LocalAudited record {|
    *types:Audited;
    @localMeta {name: "local-note"}
    string note;
|};

type Meta record {|
    string name;
|};

annotation Meta localMeta on record field;

public function main() {
    io:println(annotatedFields(types:Person)); //@output name
    io:println(fieldMetaName(types:Person, "name")); //@output imported-name
    io:println(fieldMetaName(types:Person, "age")); //@output <absent>
    io:println(annotatedFields(LocalAudited)); //@output createdAt,note
    io:println(fieldMetaName(LocalAudited, "createdAt")); //@output imported-createdAt
}

function annotatedFields(typedesc<anydata> td) returns string = external;

function fieldMetaName(typedesc<anydata> td, string fieldName) returns string = external;
