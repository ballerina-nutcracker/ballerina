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

function foo() {
    x = value?.'_; // @error
    x = value?.mod:_; // @error
    x = value?.mod:'_; // @error
    x = ep->'_(); // @error
    C a = new(_ = 1); // @error
    C b = new C('_ = 1); // @error
    x = function(int _) returns int => 1; // @error
    x = (_)=>1; // @error
    x = 1;
}
function valid() {}
