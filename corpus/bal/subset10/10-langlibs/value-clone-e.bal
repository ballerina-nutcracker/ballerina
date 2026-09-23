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

import ballerina/lang.value;

class Counter {
    int n = 0;
}

public function main() {
    any a = 1;
    _ = a.clone(); // @error any is not a subtype of Cloneable

    Counter c = new;
    _ = c.clone(); // @error object values have no clone method

    int[] arr = [1];
    var strm = arr.toStream();
    _ = strm.clone(); // @error stream values have no clone method
    _ = value:clone(strm); // @error stream is not a subtype of Cloneable

    _ = value:clone(); // @error missing required argument
    _ = value:clone(arr, arr); // @error too many arguments
}
