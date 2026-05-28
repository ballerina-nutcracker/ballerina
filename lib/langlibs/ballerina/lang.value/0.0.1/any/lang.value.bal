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

// This module currently exposes no symbols; it exists so that the lang.value
// langlib resolves as a real bundle.

# Converts a value of type json to a user-specified type.
#
# This works the same as function `cloneWithType`,
# except that it also does the inverse of the conversions done by `toJson`.
#
# ```ballerina
# json arr = [1, 2, 3, 4];
# int[] intArray = check arr.fromJsonWithType();
# intArray ⇒ [1,2,3,4]
#
# type Vowels string:Char[];
#
# json vowels = ["a", "e", "i", "o", "u"];
# vowels.fromJsonWithType(Vowels) ⇒ ["a","e","i","o","u"]
#
# vowels.fromJsonWithType(string) ⇒ error
# ```
#
# + v - json value
# + t - type to convert to
# + return - value belonging to type parameter `t` or error if this cannot be done
public isolated function fromJsonWithType(json v, typedesc<anydata> t = <>) = external;
