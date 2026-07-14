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

// @productions bitwise-and-expr equality equality-expr local-var-decl-stmt int-literal
public function main() {
    // This is an error because the type of (1 & 0xFF) is int:Unsigned8
    // and the intersection of that with 0x100 is empty.
    boolean b = (1 & 255) == 256; // @error
    _ = b;
}
