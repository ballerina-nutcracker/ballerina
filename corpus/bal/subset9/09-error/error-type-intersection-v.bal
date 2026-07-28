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

import ballerina/io;

type InputErrorDetail record {|
    int|string value;
|};

type NumericErrorDetail record {|
    int|float value;
|};

type InputError error<InputErrorDetail>;

type NumericError error<NumericErrorDetail>;

type NumericInputError InputError & NumericError;

type DistinctInputError distinct error<InputErrorDetail>;

type DistinctNumericError distinct error<NumericErrorDetail>;

type DistinctNumericInputError DistinctInputError & DistinctNumericError;

public function main() {
    NumericInputError e1 = error("Numeric input error", value = 5);
    io:println(e1 is InputError); // @output true
    io:println(e1 is DistinctInputError); // @output false

    DistinctNumericInputError e2 = error("Distinct numeric input error", value = 5);
    io:println(e2 is InputError); // @output true
    io:println(e2 is DistinctInputError); // @output true
}
