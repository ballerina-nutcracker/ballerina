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
    int[] xs = [1, 2, 3];
    var doubled = from var x in xs
        let int y = x * 2
        collect y;

    var afterGroup = from var x in [1, 2, 1, 3]
        group by x
        collect x;

    var afterGroupNonKey = from var x in [1, 2, 1, 3]
        let int y = x + 10
        group by x
        collect y;

    decimal[] contextualSelect = from var _ in xs
        select 1;

    decimal contextualCollect = from var _ in xs
        collect 1;

    io:println(doubled); // @output [2,4,6]
    io:println(afterGroup); // @output [1,2,3]
    io:println(afterGroupNonKey); // @output [11,11,12,13]
    io:println(contextualSelect[0] is decimal); // @output true
    io:println(contextualCollect is decimal); // @output true
}
