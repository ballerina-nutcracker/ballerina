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
    int|float x = 5;
    var _ = -x; // @error mixed basic numeric types
    byte|float bf = 5;
    var _ = -bf; // @error mixed basic numeric types
    int|decimal id = 5;
    var _ = -id; // @error mixed basic numeric types
    float|decimal fd = 5.0;
    var _ = -fd; // @error mixed basic numeric types
    int|float? nullable = 5;
    var _ = -nullable; // @error nil removal still leaves mixed basic numeric types
}
