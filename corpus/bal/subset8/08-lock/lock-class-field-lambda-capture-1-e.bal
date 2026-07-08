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

// `self.count` inside a lambda body is rejected even when the
// surrounding method has no `lock`: the lambda is a fresh closure
// that may run later, so no enclosing lock can guard the access.
isolated class Counter {
    private int count = 0;

    isolated function get() returns (function () returns int) {
        return isolated function() returns int {
            return self.count; // @error
        };
    }
}

public function main() {
    Counter _ = new;
}
