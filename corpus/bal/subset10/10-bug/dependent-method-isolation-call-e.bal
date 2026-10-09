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

class C {
    isolated function iso(typedesc<anydata> td = <>) returns td|error = external;
    function nonIso(typedesc<anydata> td = <>) returns td|error = external;
    public function pubNonIso(typedesc<anydata> td = <>) returns td|error = external;
    public isolated function pubIso(typedesc<anydata> td = <>) returns td|error = external;
}

isolated function f(C c) returns int|error {
    int a = check c.iso();
    int b = check c.nonIso(); // @error
    int d = check c.pubNonIso(); // @error
    int e = check c.pubIso();
    return a + b + d + e;
}

public function main() {
}
