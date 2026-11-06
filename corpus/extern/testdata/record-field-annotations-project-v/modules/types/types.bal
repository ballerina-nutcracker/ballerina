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

public type Meta record {|
    string name;
|};

public annotation Meta fieldMeta on record field;

public type Person record {|
    @fieldMeta {name: "imported-name"}
    string name;
    int age;
|};

public type Audited record {|
    @fieldMeta {name: "imported-createdAt"}
    string createdAt;
|};

// An alias of a record is a plain type symbol, not a record symbol, so its
// field annotations travel through the symbol pool on a different path.
public type PersonAlias Person;

public annotation Meta typeMeta on type;

@typeMeta {name: "imported-age"}
public type Age int;

public type Inner record {|
    @fieldMeta {name: "imported-x"}
    int x;
|};

public type Node record {|
    @fieldMeta {name: "imported-value"}
    int value;
    Node? next;
|};

public type Outer record {|
    Age age;
    Inner[] list;
    Node node;
|};

// Returns Outer's typedesc from this module, so the typedesc and its nested field types are
// written into this module's BIR.
public function outerType() returns typedesc<anydata> {
    return Outer;
}
