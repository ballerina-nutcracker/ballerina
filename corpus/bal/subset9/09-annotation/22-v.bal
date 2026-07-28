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

type IntInfo record {|
    int value;
|};

type BooleanInfo record {|
    boolean value;
|};

type IntListInfo record {|
    int[] values;
|};

type StringInfo record {|
    string value;
|};

annotation IntInfo unaryInfo on type;
annotation BooleanInfo logicalInfo on type;
annotation IntInfo conversionInfo on type;
annotation IntListInfo listInfo on type;
annotation StringInfo templateInfo on type;

int runtimeInt = 5;
float runtimeFloat = 6.0;
int[] runtimeList = [7, 8];

function runtimeBoolean() returns boolean {
    return true;
}

@unaryInfo {value: -runtimeInt}
type UnaryTarget int;

@logicalInfo {value: true && runtimeBoolean()}
type LogicalTarget int;

@conversionInfo {value: <int>runtimeFloat}
type ConversionTarget int;

@listInfo {values: [0, ...runtimeList]}
type ListTarget int;

@templateInfo {value: string `runtime-${runtimeInt}`}
type TemplateTarget int;

@templateInfo {value: <string>"coverage-" + <string>"margin"}
type StringAdditionTarget int;

public function main() {
    IntInfo? unaryValue = UnaryTarget.@unaryInfo;
    if unaryValue is IntInfo {
        io:println(unaryValue.value); // @output -5
    }

    BooleanInfo? logicalValue = LogicalTarget.@logicalInfo;
    if logicalValue is BooleanInfo {
        io:println(logicalValue.value); // @output true
    }

    IntInfo? conversionValue = ConversionTarget.@conversionInfo;
    if conversionValue is IntInfo {
        io:println(conversionValue.value); // @output 6
    }

    IntListInfo? listValue = ListTarget.@listInfo;
    if listValue is IntListInfo {
        io:println(listValue.values[0]); // @output 0
        io:println(listValue.values[2]); // @output 8
    }

    StringInfo? templateValue = TemplateTarget.@templateInfo;
    if templateValue is StringInfo {
        io:println(templateValue.value); // @output runtime-5
    }

    StringInfo? stringAdditionValue = StringAdditionTarget.@templateInfo;
    if stringAdditionValue is StringInfo {
        io:println(stringAdditionValue.value); // @output coverage-margin
    }
}
