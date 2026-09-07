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

import ballerina/time;

# The prefix used to construct the `typeUrl` field of an `Any` value.
const string ANY_TYPE_URL_PREFIX = "type.googleapis.com/google.protobuf.";

# The set of value shapes that can be packed into, or unpacked from, an `Any` value.
public type ValueType int|float|string|boolean|time:Utc|time:Seconds|record {}|()|byte[]|map<anydata>;

# A `typedesc` constrained to the `ValueType` shapes.
public type ValueTypeDesc typedesc<ValueType>;

# Represents the error type of the `protobuf.types.any` module.
#
# In jBallerina this is `distinct protobuf:Error`, chaining into the package's root error
# type. That chain isn't reproducible here: a module cannot currently import its own
# package's default/root module by name (see
# https://github.com/ballerina-nutcracker/ballerina/issues/777), which `protobuf.types.any`
# would need to reference `protobuf:Error`. Declared as a standalone `distinct error`
# instead; catching the root `protobuf:Error` type will not also catch this module's errors.
public type Error distinct error;

# Represents the error that occurs when the value packed into an `Any` does not match the
# type requested when unpacking it.
public type TypeMismatchError distinct Error;

# Represents the protobuf well-known type ``google.protobuf.Any``.
#
# + typeUrl - A URL identifying the packed value's type
# + value - The packed value
public type Any record {|
    string typeUrl = "";
    ValueType value = ();
|};

# Represents a context object used to pass the headers together with a stream of `Any` content.
#
# + content - The `Any` content stream
# + headers - The headers map
public type ContextAnyStream record {|
    stream<Any, error?> content;
    map<string|string[]> headers;
|};

# Represents a context object used to pass the headers together with `Any` content.
#
# + content - The `Any` content
# + headers - The headers map
public type ContextAny record {|
    Any content;
    map<string|string[]> headers;
|};

# Packs the given value into an `Any` value.
#
# + message - The value to pack
# + return - The packed `Any` value, or an `Error` if the value could not be packed
public isolated function pack(ValueType message) returns Any|Error {
    string suffix = typeUrlSuffix(message);
    string hexValue = check serializeToHex(message, suffix);
    return {typeUrl: ANY_TYPE_URL_PREFIX + suffix, value: hexValue};
}

# Unpacks the value held by the given `Any` value into the target type.
#
# + anyValue - The `Any` value to unpack
# + targetTypeOfAny - The `typedesc` of the type to unpack into
# + return - The unpacked value, or an `Error` if the packed value does not match the target type
public isolated function unpack(Any anyValue, ValueTypeDesc targetTypeOfAny = <>) returns targetTypeOfAny|Error = external;

# Determines the ``google.protobuf.*`` well-known-type name suffix for a `ValueType` value's shape.
#
# An arbitrary `record {}` message value (as opposed to a genuine `map<anydata>` value) is not
# distinguishable from `map<anydata>` by an `is` check — any record whose field types are all
# `anydata` structurally satisfies `map<anydata>` too — so it is packed as a `Struct` rather than
# resolved via its own registered proto descriptor. See the module README for details. The one
# record shape that stays distinguishable is the closed empty record, which needs no descriptor
# and so keeps jBallerina's `Empty` mapping.
isolated function typeUrlSuffix(ValueType message) returns string {
    if message is () {
        return "Empty";
    } else if message is boolean {
        return "BoolValue";
    } else if message is int {
        return "Int64Value";
    } else if message is float {
        return "FloatValue";
    } else if message is string {
        return "StringValue";
    } else if message is byte[] {
        return "BytesValue";
    } else if message is time:Utc {
        return "Timestamp";
    } else if message is time:Seconds {
        return "Duration";
    } else if message is record {||} {
        return "Empty";
    } else {
        return "Struct";
    }
}

isolated function serializeToHex(ValueType message, string typeUrlSuffix) returns string|Error = external;

// TODO: newAnyError and newTypeMismatchError below exist only because native code has no way
// to construct a value of a specific distinct error type directly — it must call back into a
// Ballerina-level constructor to get the real distinct identity. Remove both once the runtime
// gains an API for native code to build a distinct-typed error directly (see the matching TODO
// on callModuleErrorConstructor in native/any.go).

# Constructs an `Error` carrying the module's real distinct-error identity.
# Called from native code so errors satisfy `is Error`.
isolated function newAnyError(string message) returns Error {
    return error Error(message);
}

# Constructs a `TypeMismatchError` carrying the module's real distinct-error identity.
# Called from native code so `unpack` can raise an error that satisfies `is TypeMismatchError`.
isolated function newTypeMismatchError(string message) returns TypeMismatchError {
    return error TypeMismatchError(message);
}
