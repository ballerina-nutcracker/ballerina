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

# Represents the generic error type of the module.
public type Error distinct error;

# Represents the errors occurs during constraint validations.
public type ValidationError distinct Error;

# Represents the errors occurs during the type conversion.
public type TypeConversionError distinct Error;

// Native code cannot construct values of a distinct error type directly, so it calls these helpers.
isolated function newError(string message) returns Error {
    return error Error(message);
}

isolated function newValidationError(string message, error? cause) returns ValidationError {
    return error ValidationError(message, cause);
}

isolated function newTypeConversionError(string message, error? cause) returns TypeConversionError {
    return error TypeConversionError(message, cause);
}
