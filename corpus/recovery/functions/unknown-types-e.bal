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

function unknownOrdinary(Missing value) returns int => 1; // @error
function unknownRest(MissingRest... values) returns int => 2; // @error
function unknownReturn(int value) returns MissingReturn => value; // @error
function independent(int value, string... labels) returns int => value + labels.length();
class Signatures {
    function unknown(MissingMethod value) returns int => 1; // @error
    function surviving(int value) returns int => value;
    function anotherUnknown(MissingOther value) returns int => 2; // @error
}
function bodyOnly(int value) returns int => value + ; // @error
function callBodyOnly(int value) returns int => bodyOnly(value);
