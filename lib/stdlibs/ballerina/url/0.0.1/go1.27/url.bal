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

# Encodes the given string value into a `string` using the provided charset.
# ```ballerina
# string value = "param1=http://xyz.com/?a=12&b=55&param2=99";
# string encoded = check url:encode(value, "UTF-8");
# ```
#
# + value - String value to be encoded
# + charset - The name of a supported character encoding
# + return - The `string` value of the encoded string or an error if an error occurred
public isolated function encode(string value, string charset) returns string|error = external;

# Decodes the given string value into a `string` using the provided charset.
# ```ballerina
# string value = "http://www.domain.com/?param1=http%3A%2F%2Fxyz.com%2F%3Fa%3D12%26b%3D55&param2=99";
# string decoded = check url:decode(value, "UTF-8");
# ```
#
# + value - String value to be decoded
# + charset - The name of a supported character encoding
# + return - The `string` value of the decoded string or an error if an error occurred
public isolated function decode(string value, string charset) returns string|error = external;
