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
type R record {| R? next = {next: ()}; |};
type A record {| int x; |};
type B record {| string x; |};
function choose(A|B value) returns A|B { return value; }
final var result = choose({x: // @error failed candidates discard their inline default batches
    (from int n in [1]
     let record {| R r = {}; int x = 1; |} row = {}
     select row.x + n - 1)[0]});
public function main() { _ = result; }
