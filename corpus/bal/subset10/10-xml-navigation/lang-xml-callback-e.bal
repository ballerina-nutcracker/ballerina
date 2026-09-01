// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

int state = 0;

function nonIsolated(xml:Element item) returns xml:Element {
    state += 1;
    return item;
}

isolated function invalidIsolation() returns xml<xml:Element> {
    xml:Element element = xml `<x/>`;
    return element.map(nonIsolated); // @error
}

public function main() {
    xml:Element element = xml `<x/>`;
    xml _ = element.map(function(xml:Element item) returns int { // @error
        _ = item;
        return 1;
    });
    xml _ = element.filter(function(xml:Element item) returns string { // @error
        _ = item;
        return "yes";
    });
    element.forEach(function(xml:Element item) returns int { // @error
        _ = item;
        return 1;
    });
}
