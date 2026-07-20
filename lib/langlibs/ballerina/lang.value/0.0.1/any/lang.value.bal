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

# Constructs a value with a specified type by cloning another value.
#
# When `v` is a structural value, the inherent type of the constructed value comes from `t`.
# When `t` is a union with more than one structural type descriptor applicable to `v`'s basic type,
# the leftmost matching type descriptor is used as the inherent type.
# The operation is applied recursively to each member of `v` using the type required by the inherent type.
#
# Unlike `clone`:
# - the inherent type of structural values comes from `t`, not `v`
# - the read-only bit comes from `t`
# - the graph structure is not preserved (result is always a tree); an error is returned if `v` has cycles
# - immutable structural values are copied rather than returned as-is
# - numeric values may be converted via NumericConvert
#
# A record field missing from `v` is always an error, even when the field declares
# a default value in `t`; default-value injection is not yet supported.
#
# ```ballerina
# anydata rec = {name: "Alice", age: 30};
# type Person record {| string name; int age; |};
# Person p = check rec.cloneWithType();
# p ⇒ {name:"Alice",age:30}
# ```
#
# + v - the value to clone
# + t - the type for the clone to be constructed
# + return - a new value of type `t`, or an error if this cannot be done
public isolated function cloneWithType(anydata v, typedesc<anydata> t = <>) returns t|error = external;

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
public isolated function fromJsonWithType(json v, typedesc<anydata> t = <>) returns t|error  = external;
