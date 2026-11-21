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
    int[1] arr = [1];
    _ = arr.pop(); // @error cannot pop a fixed-length list
    _ = arr.shift(); // @error cannot shift a fixed-length list
    arr.unshift(5); // @error cannot unshift a fixed-length list

    [int, int] pair = [1, 2];
    _ = pair.pop(); // @error cannot pop a fixed-length list
    _ = pair.shift(); // @error cannot shift a fixed-length list
    pair.unshift(); // @error cannot unshift a fixed-length list

    int[1] one = [1];
    int[1]|int[2] either = one;
    _ = either.pop(); // @error cannot pop a fixed-length list

    [int] single = [1];
    [int]|[int, int] tuples = single;
    tuples.unshift(); // @error cannot unshift a fixed-length list

    int[0] empty = [];
    _ = empty.shift(); // @error cannot shift a fixed-length list

    never[] nothing = [];
    nothing.unshift(); // @error cannot unshift a fixed-length list
}
