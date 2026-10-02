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

function consume(int value) returns int => value;
function consumeFunction(function() returns int callable) returns int => callable();

function nestedBodies() {
    int|string before = 1;
    int|string after = 2;
    if before is int {
        if after is int {
            var enclosing = function() returns int {
                consumeFunction(function() returns int => before + missingNested); // @error
                consume(after); // @error capture after nested failure in enclosing context
                return 1;
            };
            consume(enclosing());
            consume(before); // @error nested capture propagated
            consume(after); // @error enclosing capture context restored
        }
    }
}

