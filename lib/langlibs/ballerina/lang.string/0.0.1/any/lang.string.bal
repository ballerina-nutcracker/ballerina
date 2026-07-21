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

# Returns a substring of `str`.
#
# + str - source string
# + startIndex - the starting index, inclusive
# + endIndex - the ending index, exclusive
# + return - substring consisting of characters with index startIndex up to endIndex-1
public isolated function substring(string str, int startIndex, int endIndex = length(str)) returns string = external;

# Tests whether `str1` and `str2` are equal, ignoring the case of ASCII characters.
#
# + str1 - the first string to be compared
# + str2 - the second string to be compared
# + return - true if `str1` and `str2` are equal when any ASCII uppercase characters are
#   converted to lowercase
public isolated function equalsIgnoreCaseAscii(string str1, string str2) returns boolean = external;

# Converts occurrences of a character in the ASCII range to lowercase.
#
# + str - the string to be converted
# + return - `str` with any occurrences of an ASCII uppercase character replaced with the
#   equivalent lowercase character
public isolated function toLowerAscii(string str) returns string = external;

# Converts occurrences of a character in the ASCII range to uppercase.
#
# + str - the string to be converted
# + return - `str` with any occurrences of an ASCII lowercase character replaced with the
#   equivalent uppercase character
public isolated function toUpperAscii(string str) returns string = external;

# Removes ASCII white space characters from the start and end of a string.
#
# + str - the string to be trimmed
# + return - `str` with leading or trailing ASCII white space characters removed
public isolated function trim(string str) returns string = external;

# Represents `str` as an array of bytes using UTF-8.
#
# + str - the string to be converted
# + return - UTF-8 byte array representation of `str`
public isolated function toBytes(string str) returns byte[] = external;

# Constructs a string from its UTF-8 representation in a byte array.
#
# + bytes - UTF-8 byte array representation of a string
# + return - `bytes` converted to a string, or an error if `bytes` is not valid UTF-8
public isolated function fromBytes(byte[] bytes) returns string|error = external;
