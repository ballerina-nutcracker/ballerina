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

type ReturnedFunction function (int returnedArg = 10) returns int;
type ReturnedAlias ReturnedFunction;
type DeclaredFunction function (int declaredArg = 20) returns int;

function getFunction(int outerArg = 1) returns ReturnedAlias {
    int _ = outerArg;
    return function (int implementationArg) returns int {
        return implementationArg;
    };
}

class Factory {
    function create(int methodArg = 2) returns function (int methodReturnedArg = 30) returns int {
        int _ = methodArg;
        return function (int implementationArg) returns int {
            return implementationArg;
        };
    }
}

client class RemoteFactory {
    remote function create(int remoteOuterArg = 5) returns function (int remoteReturnedArg = 50) returns int {
        int _ = remoteOuterArg;
        return function (int implementationArg) returns int {
            return implementationArg;
        };
    }
}

var moduleFunction = getFunction();

public function main() {
    var inferred = getFunction();
    io:println(inferred()); // @output 10
    io:println(inferred(returnedArg = 11)); // @output 11

    var grouped = (getFunction());
    io:println(grouped(returnedArg = 12)); // @output 12

    var inferredAgain = getFunction(outerArg = 3);
    io:println(inferredAgain(returnedArg = 13)); // @output 13
    var inferredDefaultOuter = getFunction();
    io:println(inferredDefaultOuter(returnedArg = 14)); // @output 14

    Factory factory = new;
    var fromMethod = factory.create();
    io:println(fromMethod()); // @output 30
    io:println(fromMethod(methodReturnedArg = 31)); // @output 31
    var fromMethodAgain = factory.create(methodArg = 4);
    io:println(fromMethodAgain(methodReturnedArg = 32)); // @output 32

    RemoteFactory remoteFactory = new;
    var fromRemoteMethod = remoteFactory->create();
    io:println(fromRemoteMethod()); // @output 50
    io:println(fromRemoteMethod(remoteReturnedArg = 51)); // @output 51
    var fromRemoteMethodAgain = remoteFactory->create(remoteOuterArg = 6);
    io:println(fromRemoteMethodAgain(remoteReturnedArg = 52)); // @output 52

    io:println(moduleFunction()); // @output 10
    io:println(moduleFunction(returnedArg = 15)); // @output 15

    DeclaredFunction explicitlyTyped = getFunction();
    io:println(explicitlyTyped()); // @output 20
    io:println(explicitlyTyped(declaredArg = 21)); // @output 21

    ReturnedAlias|string positiveNarrowed = getFunction();
    if positiveNarrowed is ReturnedAlias {
        io:println(positiveNarrowed(returnedArg = 16)); // @output 16
    }

    ReturnedAlias|string negatedNarrowed = getFunction();
    if negatedNarrowed !is ReturnedAlias {
        io:println("unexpected");
    } else {
        io:println(negatedNarrowed()); // @output 10
        io:println(negatedNarrowed(returnedArg = 17)); // @output 17
    }
}
