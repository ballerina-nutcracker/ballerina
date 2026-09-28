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

function returnInsideWhere() returns int {
    from var x in [1] where x > 5 do { return x; }; // @error missing return statement
}

function noReturnAfterBody() returns int {
    from var x in [1] do { _ = x; }; // @error missing return statement
}

function noPanicAfterBody() returns never {
    from var x in [1] do { _ = x; panic error("boom"); }; // @error expected panic
}

function noReturnAfterCall() returns int {
    noop(); // @error missing return statement
}

function noop() {
}

public function main() {
    _ = returnInsideWhere();
    _ = noReturnAfterBody();
    _ = noReturnAfterCall();
    noPanicAfterBody();
}
