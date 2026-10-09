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

type F future<stream<F>>;

function convert(stream<int, error> e) {
    stream<int> s = e; // @error completion type of stream<int> does not allow error
    _ = s;
}

function narrowUnion(stream<int>|stream<string, error> u) {
    stream<boolean> s = u; // @error union of streams
    _ = s;
}

function negated(stream s) {
    if s is stream<int> {
        return;
    } else {
        int i = s; // @error stream that is not stream<int>
        _ = i;
    }
}

function recursive(F f) {
    int i = f; // @error recursive stream nested in a future
    _ = i;
}

public function main() {
}
