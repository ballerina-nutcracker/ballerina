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

public function main() {
    boolean b = true;
    int i = 10;
    float f = 1.5;
    decimal d = 2.25d;
    string s = "hi";
    io:println(string `b=${b}, i=${i}, f=${f}, d=${d}, s=${s}`); // @output b=true, i=10, f=1.5, d=2.25, s=hi
}
