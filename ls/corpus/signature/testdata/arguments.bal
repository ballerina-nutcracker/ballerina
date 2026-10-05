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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

type Count int;
function choose(Count count, string title = "default", string... extra) returns int {
    _ = title; _ = extra; return count;
}
function pair(int first, int second) returns int { return first + second; }
function arrays(int[] values, int|string value) { _ = values; _ = value; }
function zero() {}
public function main() {
    _ = choose(1, "a,b", "x", "y");
    _ = choose(title = "second", count = 1);
    _ = pair(pair(1, 2), 3);
    _ = pair(1, 2, 3);
    arrays([1,2], "value");
    zero();
}
