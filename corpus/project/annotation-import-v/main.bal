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

import testorg/annotation_import_v.meta;

type LocalInfo record {|
    string name;
|};

annotation LocalInfo info on type;

@info {name: "local"}
@meta:info {name: "imported", code: 11, values: [11, 12]}
@meta:marker
@meta:sourceInfo {name: "local-source", code: 12}
type Person record {|
    string name;
|};

@meta:info {name: "const-ref", code: meta:DEFAULT_CODE, values: [meta:DEFAULT_CODE, 100]}
type ConstTagged record {|
    string value;
|};

@meta:info {name: "const-list-ref", code: meta:DEFAULT_CODE, values: meta:DEFAULT_VALUES}
type ConstListTagged record {|
    string value;
|};

public function main() {
    LocalInfo? localInfo = Person.@info;
    if localInfo is LocalInfo {
        io:println(localInfo.name); // @output local
    }

    meta:Info? importedInfo = Person.@meta:info;
    if importedInfo is meta:Info {
        io:println(importedInfo.name); // @output imported
        io:println(importedInfo.code); // @output 11
        io:println(importedInfo.values[1]); // @output 12
    }

    boolean? importedMarker = Person.@meta:marker;
    if importedMarker is boolean {
        io:println(importedMarker); // @output true
    }

    meta:SourceInfo? localSourceInfo = Person.@meta:sourceInfo;
    io:println(localSourceInfo is ()); // @output true

    meta:Info? dependencyInfo = meta:Tagged.@meta:info;
    if dependencyInfo is meta:Info {
        io:println(dependencyInfo.name); // @output dependency
        io:println(dependencyInfo.code); // @output 17
        io:println(dependencyInfo.values[1]); // @output 18
    }

    boolean? dependencyMarker = meta:Tagged.@meta:marker;
    if dependencyMarker is boolean {
        io:println(dependencyMarker); // @output true
    }

    meta:SourceInfo? dependencySourceInfo = meta:Tagged.@meta:sourceInfo;
    io:println(dependencySourceInfo is ()); // @output true

    meta:Info? runtimeDependencyInfo = meta:RuntimeTagged.@meta:info;
    if runtimeDependencyInfo is meta:Info {
        io:println(runtimeDependencyInfo.name); // @output runtime-dependency
        io:println(runtimeDependencyInfo.code); // @output 23
        io:println(runtimeDependencyInfo.values[1]); // @output 24
    }

    meta:NumericInfo? numericInfo = meta:NumericTagged.@meta:numericInfo;
    if numericInfo is meta:NumericInfo {
        io:println(numericInfo.ratio); // @output 3.14
        io:println(numericInfo.amount); // @output 2.5
        io:println(numericInfo.optional); // @output
    }

    meta:Info? constInfo = ConstTagged.@meta:info;
    if constInfo is meta:Info {
        io:println(constInfo.name); // @output const-ref
        io:println(constInfo.code); // @output 99
        io:println(constInfo.values[0]); // @output 99
        io:println(constInfo.values[1]); // @output 100
    }

    meta:Info? constListInfo = ConstListTagged.@meta:info;
    if constListInfo is meta:Info {
        io:println(constListInfo.name); // @output const-list-ref
        io:println(constListInfo.values[0]); // @output 31
        io:println(constListInfo.values[1]); // @output 32
    }
}
