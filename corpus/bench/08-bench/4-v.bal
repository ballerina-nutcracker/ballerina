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

const int MagicMod = 100000;

type ComparableValue record {
    int intVal;
    float floatVal;
    string stringVal;
    int? optionalVal;
};

function buildValueArray(int count) returns ComparableValue[] {
    ComparableValue[] values = [];
    foreach int i in 0 ..< count {
        int iVal = (i * 13) % 1000;
        int iMod = iVal % 100;
        float fVal = <float>iMod / 10.0;
        string sVal = "val";
        int? oVal = ();
        if i % 3 == 0 {
            oVal = iVal;
        }
        ComparableValue val = {
            intVal: iVal,
            floatVal: fVal,
            stringVal: sVal,
            optionalVal: oVal
        };
        values.push(val);
    }
    return values;
}

function computeChecksum(ComparableValue[] values) returns int {
    int checksum = 0;
    int len = values.length();
    
    foreach int i in 0 ..< len {
        ComparableValue val = values[i];
        
        if val.intVal > 500 {
            checksum = (checksum + val.intVal) % MagicMod;
        } else if val.intVal < 200 {
            checksum = (checksum + val.intVal * 2) % MagicMod;
        }
        
        int? lifted = val.intVal + val.optionalVal;
        if lifted is int {
            checksum = (checksum + lifted) % MagicMod;
        } else {
            checksum = (checksum + 1) % MagicMod;
        }
        
        int? bitwise = val.intVal & val.optionalVal;
        if bitwise is int {
            checksum = (checksum + bitwise) % MagicMod;
        }
    }
    
    return checksum;
}

function repeatedComputeLoop(ComparableValue[] values, int iterations) returns int {
    int result = 0;
    foreach int _ in 0 ..< iterations {
        int partial = computeChecksum(values);
        result = (result + partial) % MagicMod;
    }
    return result;
}

public function main() {
    ComparableValue[] values = buildValueArray(100);
    io:println(values.length()); // @output 100
    
    int checksum = computeChecksum(values);
    io:println(checksum); // @output 76380
    
    int loopResult = repeatedComputeLoop(values, 50);
    io:println(loopResult); // @output 19000
}
