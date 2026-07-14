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

// @productions float type-cast-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int i = 10;
    float f = <float>i;
    io:println(f); // @output 10.0

    i = 9223372036854775807;
    // xxx print in hex to be more portable 
    io:println(<float>i); // @output 9.223372036854776e18
}
