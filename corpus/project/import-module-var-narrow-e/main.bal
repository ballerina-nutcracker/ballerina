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

import ballerina/io;
import testorg/import_module_var_narrow_e.data;

// A mutable module binding of an imported module is no more stable than one of
// this module, but a final one stays eligible.
public function main() {
    if data:shared is int {
        data:mutate();
        int y = data:shared; // @error an imported mutable module variable cannot be narrowed
        io:println(y);
    }
    if data:pinned is int {
        io:println(data:pinned + 1);
    }
}
