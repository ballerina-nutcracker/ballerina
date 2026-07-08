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
    // The Ballerina spec leaves the decimal exponent range implementation-dependent
    // within the minimum bounds -6176 to 6111: https://ballerina.io/spec/lang/master/#section_5.2.4.2.
    // This value requires a quantize exponent outside the implementation-supported
    // range, so this implementation panics.
    _ = 1.23d.round(2147483649); // @panic invalid fractionDigits
}
