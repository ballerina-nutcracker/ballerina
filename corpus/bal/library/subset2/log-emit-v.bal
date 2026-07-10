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

// INFO/WARN/ERROR records are emitted (DEBUG is filtered at the default INFO
// level). They are written to stderr in LOGFMT; the integration golden captures
// the stderr stream with the non-deterministic timestamp normalized to <TIME>.
import ballerina/log;

public function main() {
    log:printInfo("server started");
    log:printWarn("low disk space", diskPct = 12);
    log:printError("request failed", 'error = error("disk full"));
}
