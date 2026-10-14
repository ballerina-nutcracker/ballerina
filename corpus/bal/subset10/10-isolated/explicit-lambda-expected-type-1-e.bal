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

int g = 1;

type IF isolated function (int) returns int;

// An explicit lambda's type is its written signature, which is not isolated,
// so it is not a subtype of the isolated function type it is assigned to.
function pureBody() {
    IF h = function(int y) returns int { // @error not isolated
        return y;
    };
    _ = h(1);
}

function readsMutableModuleVar() {
    IF h = function(int y) returns int { // @error not isolated
        return y + g;
    };
    _ = h(1);
}

public function main() {
    pureBody();
    readsMutableModuleVar();
}
