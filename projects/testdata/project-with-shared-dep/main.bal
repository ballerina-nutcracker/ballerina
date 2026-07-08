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

import mockorg/middlepkg;
import mockorg/leafpkg;

public function main() {
    // Both imported directly; middlepkg also depends on leafpkg.
    // The dependency graph must record the middlepkg->leafpkg edge so that
    // topological sort places leafpkg before middlepkg.
    int _ = leafpkg:getValue();
    int _ = middlepkg:getDoubledValue();
}
