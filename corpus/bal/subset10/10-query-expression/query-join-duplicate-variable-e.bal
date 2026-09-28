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

public function main() {
    int[] xs = [1, 2];
    int[] ys = [2, 3];
    int y = 5;
    _ = y;
    from var x in xs join var y in ys on x equals y do { _ = x + y; }; // @error y is already defined outside the query
    from var x in xs join var x in ys on x equals x do { _ = x; }; // @error x is already defined by the from clause
    int[] _ = from var x in xs join var y in ys on x equals y select x + y; // @error y is already defined outside the query
    int[] _ = from var x in xs join var x in ys on x equals x select x; // @error x is already defined by the from clause
}
