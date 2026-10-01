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

function call(function () returns int f) returns int => f();

function str(function () returns string f) returns string => f();

public function main() {
    xml a = xml `<a>${call(function() returns int { string s = 1; return s.length(); })}</a>`; // @error incompatible type
    xml b = xml `<a/>${call(function() returns int { string s = 2; return s.length(); })}<b/>`; // @error incompatible type
    xml c = xml `<a k="${str(function() returns string { int s = "a"; return s.toString(); })}"/>`; // @error incompatible type
    xml d = xml `<a><b>${call(function() returns int { string s = 3; return s.length(); })}</b></a>`; // @error incompatible type
    _ = a + b + c + d;
}
