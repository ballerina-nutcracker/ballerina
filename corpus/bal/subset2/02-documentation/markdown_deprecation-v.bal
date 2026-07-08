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

# Documentation for deprecated function with parameters
# This function adds two numbers
# + x - first number
# + y - second number
# # Deprecated
# This function is deprecated. Use `addNumbers` instead.
function add(int x, int y) returns int {
    return x + y;
}

# Documentation for function with deprecated parameters
# This function processes data
# + x - first parameter
# + y - second parameter
# # Deprecated
# + x - This parameter is deprecated
function process(int x, int y) {
    int _ = x;
    int _ = y;
    // implementation
}

public function main() {
}
