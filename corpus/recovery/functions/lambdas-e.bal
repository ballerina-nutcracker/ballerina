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

function lambdas() {
    var a = function(int _) returns int => 1; // @error
    var b = function(int... _) returns int => 1; // @error
    var c = function(int x = ) returns int => 1; // @error
    var d = function() returns mod:_ => 1; // @error
    var e = (_)=>1; // @error
    var f = isolated isolated function() returns int => 1; // @error
    var g = function(@mod:_ int x) returns int => x; // @error
    var h = function() returns @mod:_ int => 1; // @error
    var validParameterSibling = function(int x) returns int => x;
    var validReturnSibling = function() returns int => 1;
    int later = validParameterSibling(validReturnSibling()); // @error unused surviving sibling
}
function valid() {}
