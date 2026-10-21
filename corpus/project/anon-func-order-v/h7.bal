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

function h7() returns function () returns int {
    function () returns int g = function() returns int { return 7; };
    _ = 1 + g();
    _ = 2 + g();
    _ = 3 + g();
    _ = 4 + g();
    _ = 5 + g();
    _ = 6 + g();
    _ = 7 + g();
    _ = 8 + g();
    _ = 9 + g();
    _ = 10 + g();
    _ = 11 + g();
    _ = 12 + g();
    _ = 13 + g();
    _ = 14 + g();
    _ = 15 + g();
    return g;
}
