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

type OpenRecord record {|
    int fixed;
    json...;
|};

type Person record {|
    json name;
    json...;
|};

type NoX record {|
    never x?;
    json...;
|};

function assignmentBreaksLaxChain(map<json> m) {
    json|error a = m.a;
    _ = a.b; // @error expression-local laxness does not survive assignment
}

function laxLvalue(map<json> m) {
    m.a = {}; // @error lax access cannot be an lvalue
}

function strictOpenRecordAccess(OpenRecord rec) {
    _ = rec.missing; // @error undeclared open-record field remains invalid
    _ = rec?.missing; // @error undeclared optional open-record field remains invalid
}

function strictJSONRecordAccess() {
    Person p = {name: "Ada", "extra": 1};
    _ = p.age; // @error record with named json field is not lax
}

function emptyLaxFieldMember(NoX noX) {
    _ = noX.x; // @error optional never field does not prevent laxness
}
