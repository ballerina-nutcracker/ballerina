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

type Cfg record {| string name; |};
annotation Cfg RemoteCfg on service remote function;
annotation RemoteMarker on service remote function;
annotation ObjectMarker on object function;
const annotation SourceRemote on source service remote function;
const annotation SourceObject on source object function;
const annotation MixedRemote on source service remote function, object function;
const annotation MixedObject on service remote function, source object function;
annotation Unrestricted;

service class Svc {
    @RemoteCfg {name: "ping"}
    @RemoteMarker
    @ObjectMarker
    @SourceRemote
    @SourceObject
    @MixedRemote
    @MixedObject
    @Unrestricted
    remote function ping() {}
}

public function main() {
    Svc svc = new;
    io:println(svc is service object {}); // @output true
}
