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


function flushBeforeSend() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    check flush a;
    1 -> a; // @error no flush or sync send follows the send
    check wait a;
}

function flushOtherPeer() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    worker b {
    }
    1 -> a; // @error flushing b doesn't cover the send to a
    flush b;
    check wait a;
    wait b;
}

function syncSendOtherPeer() returns error? {
    worker a returns error? {
        int _ = check <- function;
    }
    worker b returns error? {
        int _ = check <- function;
    }
    1 -> a; // @error a sync send to b doesn't cover the send to a
    check (2 ->> b);
    check wait a;
    check wait b;
}
