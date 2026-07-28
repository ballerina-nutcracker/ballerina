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

public type Info record {|
    string name;
    int code;
    int[] values;
|};

public type SourceInfo readonly & record {|
    string name;
    int code;
|};

public type NumericInfo record {|
    float ratio;
    decimal amount;
    int? optional;
|};

public const int DEFAULT_CODE = 99;
public const DEFAULT_VALUES = [31, 32];

public annotation Info info on type;
public annotation marker on type;
public const annotation SourceInfo sourceInfo on source type;
public annotation NumericInfo numericInfo on type;

@info {name: "dependency", code: 17, values: [17, 18]}
@marker
@sourceInfo {name: "dependency-source", code: 18}
public type Tagged record {|
    string value;
|};

function runtimeCode() returns int {
    return 23;
}

@info {name: "runtime-dependency", code: runtimeCode(), values: [23, 24]}
public type RuntimeTagged record {|
    string value;
|};

@numericInfo {ratio: 3.14, amount: 2.5d, optional: ()}
public type NumericTagged record {|
    string value;
|};
