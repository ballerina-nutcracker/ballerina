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

public function main() {
    xml x = xml `<a><b>1</b><c>2</c><d>3</d><e>4</e></a>`;
    xml d = x/<d>;
    worker w1 returns xml {
        return x/<b>;
    }
    worker w2 returns xml {
        return x/<c>;
    }
    xml e = x/<e>;
    xml b = wait w1;
    xml c = wait w2;
    io:println(b + c + d + e); // @output <b>1</b><c>2</c><d>3</d><e>4</e>
}
