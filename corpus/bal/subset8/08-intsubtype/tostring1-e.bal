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

type Wide int:Signed8|int:Unsigned32;

type Mixed -5|int:Unsigned8|256;

type Byte int:Unsigned8|256;

public function main() {
    9223372036854775807 max = 9223372036854775807;
    string _ = max; // @error
    Wide wide = 7;
    string _ = wide; // @error
    Mixed mixed = 7;
    string _ = mixed; // @error
    Byte b = 7;
    if b != 0 {
        string _ = b; // @error
    }
}
