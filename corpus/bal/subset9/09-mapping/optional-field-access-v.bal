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

type RequiredX record {|
    int x;
|};

type OptionalX record {|
    int x?;
|};

type RequiredY record {|
    int y;
|};

type OptionalX2 record {|
    int x?;
    int y;
|};

type RequiredX2 record {|
    int x;
    int y;
|};

type RequiredXAlso record {|
    int x;
|};

type ReadonlyRequiredX readonly & record {|
    int x;
|};

type IntersectX RequiredX & RequiredXAlso;

type Inner record {|
    int x;
|};

type Outer record {|
    Inner inner;
|};

type ErrorField record {|
    error e?;
|};

public function main() {
    RequiredX r = {x: 1};
    int rX = r?.x;
    io:println(rX); // @output 1

    int bracedRX = (r)?.x;
    io:println(bracedRX); // @output 1

    OptionalX o = {x: 11};
    int? oXWithValue = o?.x;
    io:println(oXWithValue); // @output 11
    o.x = ();
    int? oX = o?.x;
    if oX is () {
        io:println("single optional nil"); // @output single optional nil
    }

    RequiredX|RequiredY oneRequired = {y: 2};
    int? oneRequiredX = oneRequired?.x;
    if oneRequiredX is () {
        io:println("union one required nil"); // @output union one required nil
    }

    OptionalX|RequiredY oneOptional = {x: 3};
    int? oneOptionalX = oneOptional?.x;
    io:println(oneOptionalX); // @output 3

    RequiredX2 allRequiredValue = {x: 4, y: 5};
    RequiredX|RequiredX2 allRequired = allRequiredValue;
    int allRequiredX = allRequired?.x;
    io:println(allRequiredX); // @output 4

    RequiredX|OptionalX allOptional = {};
    int? allOptionalX = allOptional?.x;
    if allOptionalX is () {
        io:println("union required optional nil"); // @output union required optional nil
    }

    OptionalX2 allOptionalValue = {y: 10};
    OptionalX|OptionalX2 allOptionalFields = allOptionalValue;
    int? allOptionalFieldsX = allOptionalFields?.x;
    if allOptionalFieldsX is () {
        io:println("union all optional nil"); // @output union all optional nil
    }

    ReadonlyRequiredX ro = {x: 6};
    int roX = ro?.x;
    io:println(roX); // @output 6

    IntersectX ix = {x: 7};
    int ixX = ix?.x;
    io:println(ixX); // @output 7

    Outer nested = {inner: {x: 9}};
    int nestedX = nested.inner?.x;
    io:println(nestedX); // @output 9

    int nestedOptionalX = nested?.inner?.x;
    io:println(nestedOptionalX); // @output 9

    Outer? maybeNested = nested;
    int? maybeNestedX = maybeNested?.inner?.x;
    io:println(maybeNestedX); // @output 9

    ErrorField errorField = {e: error("boom")};
    error? maybeError = errorField?.e;
    if maybeError is error {
        io:println(maybeError.message()); // @output boom
    }
}
