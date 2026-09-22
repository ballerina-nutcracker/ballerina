// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

# Represents a context object used to pass the headers together with a stream of
# `map<anydata>` content, representing the protobuf well-known type
# ``google.protobuf.Struct``.
#
# + content - The `map<anydata>` content stream
# + headers - The headers map
public type ContextStructStream record {|
    stream<map<anydata>, error?> content;
    map<string|string[]> headers;
|};

# Represents a context object used to pass the headers together with `map<anydata>`
# content, representing the protobuf well-known type ``google.protobuf.Struct``.
#
# + content - The `map<anydata>` content
# + headers - The headers map
public type ContextStruct record {|
    map<anydata> content;
    map<string|string[]> headers;
|};
