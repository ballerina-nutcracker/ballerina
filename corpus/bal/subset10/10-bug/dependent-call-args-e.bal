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

function depend(byte[] b, typedesc<anydata> t = <>) returns t|error = external;

function inferredMismatch() returns json|error {
    return depend("text"); // @error
}

function explicitMismatch() returns json|error {
    return depend("text", json); // @error
}

function namedMismatch() returns json|error {
    return depend(t = json, b = "text"); // @error
}

function nonTypedesc() returns any|error {
    return depend([], "text"); // @error
}

type Failure error;

function outsideConstraint() returns any|error {
    return depend([], Failure); // @error
}

public function main() {
}
