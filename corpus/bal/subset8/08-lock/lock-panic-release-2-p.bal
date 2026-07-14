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

// A *dynamic* panic (divide-by-zero) inside a lock body propagates
// out of `main`. There is no static `LockEnd` on this path; the
// top-level `Interpret` recover calls `Context.ReleaseAllHeldLocks`
// to drain the held-lock stack.
isolated int counter = 0;

isolated function bump() {
    int x = 0;
    int _ = 1 / x; // @panic divide by zero
}

public function main() {
    lock {
        counter = counter + 1;
        bump();
    }
}
