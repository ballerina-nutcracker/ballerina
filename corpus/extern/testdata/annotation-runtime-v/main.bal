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
import annotationruntime.lst;
import annotationruntime.meta;

function serviceName() returns string {
    return "runtime-service";
}

function parameterName() returns string {
    return "runtime-count";
}

listener lst:Listener l = new ();

@meta:serviceMeta {name: serviceName()}
service /annotated on l {
    resource function get items(
        @meta:parameterMeta {name: parameterName()} int count,
        @meta:parameterMeta {name: "constant-header"} string header,
        @meta:marker string... extras
    ) returns int {
        var _ = header;
        var _ = extras;
        return count;
    }
}

public function main() returns error? {
    check l.inspect();
    // @output runtime-service
    // @output runtime-count
    // @output constant-header
    // @output rest-marker
    error? result = trap l.invokeWithoutArgs();
    io:println(result); // @output error("not enough arguments")
}
