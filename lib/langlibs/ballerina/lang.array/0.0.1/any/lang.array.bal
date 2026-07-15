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

# Returns the number of members of an array.
#
# + arr - the array
# + return - number of members in `arr`
public isolated function length((any|error)[] arr) returns int = external;

# Returns a base64 representation of a byte array.
#
# + arr - the byte array
# + return - the base64 encoding of `arr`
public isolated function toBase64(byte[] arr) returns string = external;

# Returns a base16 representation of a byte array.
#
# + arr - the byte array
# + return - the hexadecimal encoding of `arr`
public isolated function toBase16(byte[] arr) returns string = external;

# Returns a byte array decoded from a base64 string.
#
# + str - the base64 encoded string
# + return - the decoded byte array or an error
public isolated function fromBase64(string str) returns byte[]|error = external;

# Returns a byte array decoded from a base16 (hex) string.
#
# + str - the hexadecimal encoded string
# + return - the decoded byte array or an error
public isolated function fromBase16(string str) returns byte[]|error = external;
