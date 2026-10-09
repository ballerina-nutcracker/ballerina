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

type Album record {|
    int id;
|};

// The compiler reports only the first failing client resource access in a
// function, so this check lives in its own test rather than alongside the
// other client resource method errors.
public function main() returns error? {
    http:Client c = check new ("https://example.com");
    Album album = {id: 1};

    // A query parameter value must be a `QueryParamType`.
    http:Response _ = check c->/albums(tag = album); // @error record is not a QueryParamType

    return;
}
