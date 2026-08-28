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

# Returns the length of the string.
#
# + str - the string
# + return - the number of characters (code points) in `str`
public isolated function length(string str) returns int = external;

# Finds the first occurrence of one string in another string.
#
# + str - the string in which to search
# + substr - the string to search for
# + startIndex - index to start searching from
# + return - index of the first occurrence of parameter `substr` in parameter `str` that is >= parameter `startIndex`,
#     or `()` if there is no such occurrence
public isolated function indexOf(string str, string substr, int startIndex = 0) returns int? = external;

# Tests whether a string includes another string.
#
# + str - the string in which to search
# + substr - the string to search for
# + startIndex - index to start searching from
# + return - `true` if there is an occurrence of parameter `substr` in parameter `str` at an index >= parameter `startIndex`,
#     or `false` otherwise
public isolated function includes(string str, string substr, int startIndex = 0) returns boolean = external;

# Returns a byte array for a string using UTF-8 encoding.
#
# + str - the string value
# + return - a byte array representation of `str`
public isolated function toBytes(string str) returns byte[] = external;

# Returns a string from a byte array using UTF-8 decoding.
#
# + bytes - the byte array to decode
# + return - the decoded string, or an error if the bytes are not valid UTF-8
public isolated function fromBytes(byte[] bytes) returns string|error = external;
