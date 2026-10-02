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

service class MemberTypes {
    xml<int> firstField; // @error invalid XML constraint
    xml<int> secondField; // @error independently resolved field type
    function init(xml<int> value) { _ = value; } // @error independently resolved initializer signature
    function method() returns xml<int> {} // @error independently resolved method signature
    resource function get first() returns xml<int> {} // @error independently resolved resource signature
    resource function get second() returns xml<int> {} // @error independently resolved resource signature
}
