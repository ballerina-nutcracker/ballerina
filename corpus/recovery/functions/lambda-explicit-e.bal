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

function explicitBodies() {
    int|string captured = 1;
    if captured is int {
        var broken = function(int x) returns int => captured + x + missing; // @error
        consume(broken(2));
        consume(captured); // @error capture effect survives body failure
        var block = function(int x) returns int {
            consume(missingBlock); // @error
            return x;
        };
        consume(block(3));
        var complete = function(int x) returns int => x + 1;
        consume(complete(4));
    }
}

