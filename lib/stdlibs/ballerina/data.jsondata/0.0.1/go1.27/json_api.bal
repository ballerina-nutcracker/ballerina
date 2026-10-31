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

# Convert value of type `json` to subtype of `anydata`.
#
# + v - Source JSON value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns value belonging to the given target type, else returns an `jsondata:Error` value.
public isolated function parseAsType(json v, Options options = {}, typedesc<anydata> t = <>)
        returns t|Error = external;

# Converts JSON string to subtype of anydata.
#
# + s - Source JSON string value or byte[] or byte-block-stream
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `jsondata:Error` value.
public isolated function parseString(string s, Options options = {}, typedesc<anydata> t = <>)
        returns t|Error = external;

# Converts JSON byte[] to subtype of anydata.
#
# + s - Source JSON byte[]
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `jsondata:Error` value.
public isolated function parseBytes(byte[] s, Options options = {}, typedesc<anydata> t = <>)
        returns t|Error = external;

# Converts JSON byte-block-stream to subtype of anydata.
#
# + s - Source JSON byte-block-stream
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, value belonging to the given target type, else returns an `jsondata:Error` value.
public isolated function parseStream(stream<byte[], error?> s, Options options = {}, typedesc<anydata> t = <>)
        returns t|Error = external;

# Converts a value of type `anydata` to `json`.
#
# + v - Source anydata value
# + return - representation of `v` as value of type json
public isolated function toJson(anydata v) returns json {
    return checkpanic externToJson(v);
}

isolated function externToJson(anydata v) returns json|Error = external;

# Represent the options that can be used to modify the behaviour of the projection.
#
# + allowDataProjection - Enable or disable projection
# + enableConstraintValidation - Enable or disable constraint validation
public type Options record {
    record {
        # If `true`, nil values will be considered as optional fields in the projection.
        boolean nilAsOptionalField = false;
        # If `true`, absent fields will be considered as nilable types in the projection.
        boolean absentAsNilableType = false;
    }|false allowDataProjection = {};
    boolean enableConstraintValidation = true;
};

# Defines the name of the JSON Object key.
#
# + value - The name of the JSON Object key
public type NameConfig record {|
    string value;
|};

# The annotation is used to overwrite the existing record field name.
public const annotation NameConfig Name on record field;
