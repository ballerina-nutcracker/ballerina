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

// Represents Random module related errors.
public type Error error;

// Represents the arithmetic error.
// Note: distinct error types are not yet supported; ArithmeticError is currently an alias for Error.
public type ArithmeticError error;

# Generates a random decimal number between 0.0 and 1.0.
# ```ballerina
# float randomValue = random:createDecimal();
# ```
#
# + return - The random decimal number generated
public isolated function createDecimal() returns float {
    return externCreateDecimal();
}

# Generates a random number between the given start(inclusive) and end(exclusive) values.
# Please note that the generated number is not cryptographically secured.
# ```ballerina
# int randomInteger = check random:createIntInRange(1, 100);
# ```
#
# + startRange - The start range value
# + endRange - The end range value
# + return - The random number generated within the given range, or an error if the end range value is less than or equal to the start range value
public isolated function createIntInRange(int startRange, int endRange) returns int|Error {
    return externCreateIntInRange(startRange, endRange);
}

isolated function externCreateDecimal() returns float = external;
isolated function externCreateIntInRange(int startRange, int endRange) returns int|Error = external;
