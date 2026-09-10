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

type Excluded record {|
    never x?;
    int...;
|};

type NeedsX record {|
    int x;
|};

public function main() {
    Excluded src = {"y": 1};
    NeedsX a = {...src}; // @error the source rest is not allowed by the closed target
    _ = a;

    // The exclusion does not suppress the inhabited rest, which can still duplicate a key.
    var b = {...src, y: 2}; // @error the source rest may already supply 'y'
    _ = b;
}
