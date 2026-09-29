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

const DATA_KEY_SEPARATOR = "#";
const MODULE_SEPARATOR = ":";
const DOT = ".";
const WILDCARD = "*";
const SINGLE_QUOTE = "'";

// Workaround for a confirmed interpreter bug (see TODO.md): the unused-variable
// check is scoped per-file, not per-module. filter.bal/execute.bal/report.bal
// (separate files, P8.6-P8.8) will genuinely use these constants, but that
// won't clear this file's own "unused variable" errors due to the bug. Never
// called; only exists to give each constant a same-file reference. Remove once
// the compiler bug is fixed.
function retainModuleLevelConstants() {
    _ = DATA_KEY_SEPARATOR;
    _ = MODULE_SEPARATOR;
    _ = DOT;
    _ = WILDCARD;
    _ = SINGLE_QUOTE;
}
