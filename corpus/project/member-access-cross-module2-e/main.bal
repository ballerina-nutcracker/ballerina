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

import testorg/memberaccesscrossmodule2.foo;

function viaObjectType(foo:T t) returns int {
    return t.modOnly; // @error module-level member of an object type is not visible outside its module
}

function privateField(foo:C c) returns int {
    return c.secret; // @error private field is not visible outside its class
}

function privateMethod(foo:C c) returns int {
    return c.hidden(); // @error private method is not visible outside its class
}

function assignModuleField(foo:C c) {
    c.mod = 1; // @error module-level field is not assignable outside its module
}

function inLambda(foo:C c) returns int {
    var g = function() returns int {
        return c.mod; // @error module-level field is not visible in a lambda outside its module
    };
    return g();
}

public function main() {
}
