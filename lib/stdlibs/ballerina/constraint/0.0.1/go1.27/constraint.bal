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

# The annotation, which is used for the constraints of the `int` type.
public annotation IntConstraints Int on type, record field;

# The annotation, which is used for the constraints of the `float` type.
public annotation FloatConstraints Float on type, record field;

# The annotation, which is used for the constraints of the `int`, `float`, and `decimal` types.
public annotation NumberConstraints Number on type, record field;

# The annotation, which is used for the constraints of the `string` type.
public annotation StringConstraints String on type, record field;

# The annotation, which is used for the constraints of the `anydata[]` type.
public annotation ArrayConstraints Array on type, record field;

# The annotation, which is used for the constraints of the `Date` type, which is structurally equivalent
# to the following record:
# ```ballerina
# type Date record {
#    int year;
#    int month;
#    int day;
# };
# ```
# This annotation will enable validation on the date fields in the record. Additionally,
# the `option` field can be used to validate the date against the provided option.
public annotation DateConstraints Date on type, record field;

// This is a private type used to validate the usage of the `constraint:Date` annotation
// in the compiler plugin.

# Represents the constraint details as a record.
#
# + value - The value for the constraint
# + message - The message to be returned in case of the constraint violation
public type ConstraintRecord record {|
    anydata value;
    string message;
|};

# Represents the constraints associated with `int` type.
#
# + minValue - The inclusive lower bound of the constrained type
# + maxValue - The inclusive upper bound of the constrained type
# + minValueExclusive - The exclusive lower bound of the constrained type
# + maxValueExclusive - The exclusive upper bound of the constrained type
# + maxDigits - The maximum number of digits in the constrained type
public type IntConstraints record {|
    int|record{| *ConstraintRecord; int value; |} minValue?;
    int|record{| *ConstraintRecord; int value; |} maxValue?;
    int|record{| *ConstraintRecord; int value; |} minValueExclusive?;
    int|record{| *ConstraintRecord; int value; |} maxValueExclusive?;
    int|record{| *ConstraintRecord; int value; |} maxDigits?;
|};

# Represents the constraints associated with `float` type.
#
# + minValue - The inclusive lower bound of the constrained type
# + maxValue - The inclusive upper bound of the constrained type
# + minValueExclusive - The exclusive lower bound of the constrained type
# + maxValueExclusive - The exclusive upper bound of the constrained type
# + maxIntegerDigits - The maximum number of digits in the integer part of the constrained type
# + maxFractionDigits - The maximum number of digits in the fraction part of the constrained type
public type FloatConstraints record {|
    float|record{| *ConstraintRecord; float value; |} minValue?;
    float|record{| *ConstraintRecord; float value; |} maxValue?;
    float|record{| *ConstraintRecord; float value; |} minValueExclusive?;
    float|record{| *ConstraintRecord; float value; |} maxValueExclusive?;
    int|record{| *ConstraintRecord; int value; |} maxIntegerDigits?;
    int|record{| *ConstraintRecord; int value; |} maxFractionDigits?;
|};

# Represents the constraints associated with `int`, `float` and `decimal` types.
#
# + minValue - The inclusive lower bound of the constrained type
# + maxValue - The inclusive upper bound of the constrained type
# + minValueExclusive - The exclusive lower bound of the constrained type
# + maxValueExclusive - The exclusive upper bound of the constrained type
# + maxIntegerDigits - The maximum number of digits in the integer part of the constrained type
# + maxFractionDigits - The maximum number of digits in the fraction part of the constrained type
public type NumberConstraints record {|
    decimal|record{| *ConstraintRecord; decimal value; |} minValue?;
    decimal|record{| *ConstraintRecord; decimal value; |} maxValue?;
    decimal|record{| *ConstraintRecord; decimal value; |} minValueExclusive?;
    decimal|record{| *ConstraintRecord; decimal value; |} maxValueExclusive?;
    int|record{| *ConstraintRecord; int value; |} maxIntegerDigits?;
    int|record{| *ConstraintRecord; int value; |} maxFractionDigits?;
|};

# Represents the constraints associated with `string` type.
#
# + length - The number of characters of the constrained `string` type
# + minLength - The inclusive lower bound of the number of characters of the constrained `string` type
# + maxLength - The inclusive upper bound of the number of characters of the constrained `string` type
public type StringConstraints record {|
    int|record{| *ConstraintRecord; int value; |} length?;
    int|record{| *ConstraintRecord; int value; |} minLength?;
    int|record{| *ConstraintRecord; int value; |} maxLength?;
|};

# Represents the constraints associated with `anydata[]` type.
#
# + length - The number of members of the constrained type
# + minLength - The inclusive lower bound of the number of members of the constrained type
# + maxLength - The inclusive upper bound of the number of members of the constrained type
public type ArrayConstraints record {|
    int|record{| *ConstraintRecord; int value; |} length?;
    int|record{| *ConstraintRecord; int value; |} minLength?;
    int|record{| *ConstraintRecord; int value; |} maxLength?;
|};

# Represents the date option to be validated.
# + PAST - validates whether the date is in the past
# + PAST_OR_PRESENT - validates whether the date is in the past or present
# + FUTURE - validates whether the date is in the future
# + FUTURE_OR_PRESENT - validates whether the date is in the future or present
public enum DateOption {
   PAST,
   PAST_OR_PRESENT,
   FUTURE,
   FUTURE_OR_PRESENT
}

# Represents the constraints associated with `Date` type.
#
# + option - date option to be validated
# + message - The message to be returned in case of the constraint violation
public type DateConstraints record {
   DateOption|record{| *ConstraintRecord; DateOption value; |} option?;
   string message?;
};

# Validates the provided value against the configured annotations. Additionally, if the type of the value is different
# from the expected return type then the value will be cloned with the contextually expected type before the validation.
#
# + value - The `anydata` type value to be constrained
# + td - The type descriptor of the value to be constrained
# + return - The type descriptor of the value which is validated or else an `constraint:Error` in case of an error
public isolated function validate(anydata value, typedesc<anydata> td = <>) returns td|Error = external;
