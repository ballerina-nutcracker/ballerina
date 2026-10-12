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

import ballerina/http;

// Each case is in its own function because the compiler reports only the first
// failing resource access per function.

// A named `targetType` binds to the method's own parameter, so it must be a
// `TargetType` typedesc rather than a query value.
function namedTargetTypeMustBeTypedesc(http:Client c) returns error? {
    http:Response _ = check c->/albums(targetType = "json"); // @error targetType is not a query parameter
}

// `head` has no `targetType` parameter, so its result cannot be bound.
function headCannotBind(http:Client c) returns error? {
    string _ = check c->/albums.head(); // @error head returns only http:Response
}

public function main() returns error? {
    http:Client c = check new ("https://example.com");
    check namedTargetTypeMustBeTypedesc(c);
    check headCannotBind(c);
}
