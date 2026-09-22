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

import ballerina/io;

// Parameter names are part of the public interface, so these calls have to
// compile here as they do in jBallerina. These langlib functions are declared
// in .bal with real signatures, so named arguments resolve normally. The
// opaque-symbol functions cannot do this yet — see langlib-named-args-fv.bal.
public function main() {
    io:println(int:fromString(s = "42")); // @output 42
    io:println("hello".substring(startIndex = 1, endIndex = 3)); // @output el
}
