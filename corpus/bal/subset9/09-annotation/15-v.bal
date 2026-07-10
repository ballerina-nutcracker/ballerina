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

type Meta readonly & record {|
    string name?;
|};

public const annotation Meta annotationMeta on source annotation;
annotation Meta classMeta on class;
annotation Meta serviceMeta on service;
annotation Meta fieldMeta on object field;
annotation Meta methodMeta on object function;
annotation parameterMeta on parameter;
annotation returnMeta on return;
annotation recordFieldMeta on record field;
annotation objectFieldMeta on object field;
annotation anyFieldMeta on field;

@annotationMeta {name: "type annotation"}
annotation Meta typeMeta on type;

@classMeta {name: "counter"}
class Counter {
    @fieldMeta {name: "value"}
    private int value = 0;

    @methodMeta {name: "add"}
    function add(@parameterMeta int amount, @parameterMeta int... extras) returns @returnMeta int {
        self.value += amount;
        foreach int extra in extras {
            self.value += extra;
        }
        return self.value;
    }
}

@typeMeta {name: "result"}
type Result int;

type RecordWithMetadata record {|
    @recordFieldMeta
    @anyFieldMeta
    int value;
|};

type ObjectWithMetadata object {
    @objectFieldMeta
    @anyFieldMeta
    public int value;
};

client class MetadataClient {
    resource function get value() returns @returnMeta int {
        return 1;
    }
}

@serviceMeta {name: "service"}
service class MetadataService {
}

public function main() {
    Counter counter = new;
    Result result = counter.add(1, 2, 3);
    io:println(result); // @output 6

    Meta? serviceInfo = MetadataService.@serviceMeta;
    if serviceInfo is Meta {
        io:println(serviceInfo.name); // @output service
    }
}
