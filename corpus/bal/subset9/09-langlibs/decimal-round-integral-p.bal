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

public function main() {
    // Ballerina decimal is based on IEEE 754 decimal128 and the spec limits the
    // coefficient to 34 decimal digits: https://ballerina.io/spec/lang/master/#section_5.2.4.2.
    // Rounding this value to exponent 0 would require an integer coefficient
    // with more than 34 digits, so it must panic even though jBallerina doesn't.
    _ = (9.999999999999999999999999999999999E6144d).round(0); // @panic decimal coefficient exceeds 34 digits
}
