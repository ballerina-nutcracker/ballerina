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

// The declared default only makes `startIndex` omittable. Every other argument
// rule is unchanged: `val` is still required, an unknown argument name is still
// an error, a parameter cannot be given twice, and an explicitly supplied
// argument still has to have a compatible type.
public function main() {
    int[] arr = [1, 2, 3];
    _ = arr.indexOf(); // @error val is required
    _ = arr.indexOf(2, begin = 2); // @error no parameter named begin
    _ = arr.indexOf(2, val = 2); // @error val given twice
    _ = arr.indexOf(2, 2, startIndex = 2); // @error startIndex given twice
    _ = arr.indexOf(2, "x"); // @error startIndex is an int
    _ = arr.indexOf(2, startIndex = "x"); // @error startIndex is an int
}
