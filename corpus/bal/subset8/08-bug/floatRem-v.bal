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
    float a = getFloat(5, 100000);
    float b = getFloat(89, 20000);
    io:println(a % b); // @output 124449.35999998456
    io:println(b % a); // @output 124471.21999999792
}

// We need to make sure llvm can't optimize this function to a constant value, if that happens llvm could convert
// the `%` operation to a constant as well.
function getFloat(int seed, int iterations) returns float {
    float[] buffer = [1.0, 5.5, 3.3, 7.5, 9.5, 10.54];
    float currentVal = 0;
    int bufferSize = buffer.length();
    int currentIndex = (seed + seed) % bufferSize;
    foreach int _ in 0 ..< iterations {
        currentVal += buffer[currentIndex];
        currentIndex = (currentIndex + seed) % bufferSize;
    }
    return currentVal;
}
