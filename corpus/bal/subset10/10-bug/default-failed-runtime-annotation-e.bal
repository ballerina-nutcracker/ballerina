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
isolated function seed() returns int { return 1; }

isolated function value(isolated function () returns Info next) returns int {
    return next().other;
}

type Info record {|
    int code = seed().missing; // @error invalid default body
    int other = value(isolated function () returns Info {
        return {code: 1, other: 2};
    });
|};

const annotation Info info on type;

@info {}
type Target int;

public function main() {}
