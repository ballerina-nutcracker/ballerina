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

function ordinary() returns int => 1;
int missingInitializer = missingGlobal + 1; // @error
Unknown abandoned = ordinary(); // @error

function analysis() {
    int broken = missingLocal + ordinary(); // @error
    if missingCondition { // @error
        ordinary();
    }
    int later = "wrong"; // @error independent semantic analysis
    var defaultError = function(int x = "wrong") returns int => missingBody; // @error default and body
    var signatureError = isolated function(int x = ordinary()) returns int => missingSignatureBody; // @error independent isolated default
    var isolatedBody = isolated function() returns int => ordinary() + missingIsolated; // @error no isolation cascade
    var nested = isolated function() returns int {
        var inner = function() returns int => ordinary() + missingNested; // @error no nested isolation cascade
        if missingNestedCondition { // @error skip dependent isolation checks
            ordinary();
        }
        int isolatedDefault = "wrong"; // @error independent nested semantic analysis
        return 1;
    };
    int finalSibling = "wrong"; // @error independent semantic analysis
}
