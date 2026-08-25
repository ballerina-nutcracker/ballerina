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

public function main() returns error? {
    http:Client c = check new ("https://example.com");
    Album album = {id: 1};

    // A computed path segment must be a member of `PathParamType`; a record is not.
    http:Response _ = check c->/albums/[album]; // @error record is not a PathParamType

    // `QueryParams` declares headers, targetType, message and mediaType as `never`,
    // so none of them can be supplied as a query parameter.
    http:Response _ = check c->/albums(targetType = "json"); // @error targetType is a never field
    http:Response _ = check c->/albums(message = "body"); // @error message is a never field
    http:Response _ = check c->/albums(mediaType = "text/plain"); // @error mediaType is a never field

    return;
}
