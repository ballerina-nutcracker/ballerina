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
    decimal[] arr1 = [2, 2.3, 2.3e34d];
    io:println(arr1); // @output [2,2.3,2.3E+34]

    (decimal|int)[] arr2 = [1.2];
    io:println(arr2[0] is decimal); // @output true

    decimal[][] arr3 = [[1, 2], [3d, 3.1d], [4.33e34d]];
    io:println(arr3); // @output [[1,2],[3,3.1],[4.33E+34]]

    arr3[0].push(11e11d);
    io:println(arr3); // @output [[1,2,1.1E+12],[3,3.1],[4.33E+34]]
}
