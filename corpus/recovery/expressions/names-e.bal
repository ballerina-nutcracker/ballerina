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

function names() {
    x = mod:_; // @error
    x = mod:'_; // @error formerly silent malformed quoted identifier
    int '_ = 1; // @error formerly silent malformed local identifier
    x = mod:; // @error
    x = _:name; // @error
    x = mod:_(1); // @error
    x = foo(_ = 1); // @error
    x = foo(=1); // @error
    x = value._; // @error
    x = value?._; // @error
    x = ep->_(); // @error
    x = ep->method(_ = 1); // @error
    x = Target.@mod:_; // @error
    C c = new C(_ = 1); // @error
    C d = new('_ = 1); // @error
    int later = 3; // @error unused surviving sibling
}
function valid() {}
