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

type A record {| C? c; K k; |};
type Base object { A? a; };
type Other object { int n; };

class C {
    *Base;
    *Other;
    () a = ();
    int n = 1;
}

const boolean K = true ? true : ((<C?>()) == ()); // @error cycle through the first inclusion
public function main() {}
