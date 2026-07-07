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

// The default for `c` depends on two preceding parameters, so a lookup-key
// collision with a zero-parameter default function crashes at call time.
function sum(int a, int b, int c = a + b) returns int {
    return c;
}

function tagged(string tag = "t", int v = 9) returns int {
    return v + tag.length();
}
