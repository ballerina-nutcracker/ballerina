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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import testorg/signature66.lib;
public function main() {
    sibling(1, "sibling");
    lib:imported(1, "imported");
    lib:Alias aliasValue = "ok";
    lib:aliasCall(aliasValue);
    lib:Box box = new lib:Box(1);
    box.put(1, "tag");
    _ = box;
}
