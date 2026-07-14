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

client class foo {
    resource function get test/[string name](int count) {
        int _ = count;
    }

    resource function post items(string body, int retries) {
        string _ = body;
        int _ = retries;
    }
}

public function main() {
    foo f = new ();
    f->/test/["a"]("two"); // @error arg must be int, got string
    f->/items.post(42, 3); // @error first arg must be string, got int
    f->/items.post("body", "three"); // @error second arg must be int, got string
}
