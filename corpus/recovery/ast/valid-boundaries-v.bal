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

int x = 1 + 2;
const int y = 3;
type A mod:Name;
// doc
function foo(int x) {
	return;
}
function references() {
    x = mod:name;
    x = value?.member;
    x = ep->method();
    C a = new C(value = 1);
    C b = new(value = 1);
    int _ = 1;
}
function valid() {}
