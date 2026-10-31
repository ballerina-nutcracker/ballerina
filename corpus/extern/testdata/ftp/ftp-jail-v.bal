// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

import ballerina/ftp;
import ballerina/io;
import ballerina/lang.runtime;

function jailPort() returns int = external;

isolated string[] added = [];

service class LoopService {
    *ftp:Service;

    remote function onFileChange(ftp:WatchEvent & readonly event) {
        lock {
            foreach ftp:FileInfo f in event.addedFiles {
                added.push(f.pathDecoded);
            }
        }
    }
}

public function main() returns error? {
    ftp:Client jailed = check new ({
        host: "127.0.0.1",
        port: jailPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        userDirIsRoot: true
    });
    boolean exists = check jailed->exists("/x.txt");
    io:println(exists); // @output true
    string text = check jailed->getText("/x.txt");
    io:println(text); // @output outside
    boolean rootIsDir = check jailed->isDirectory("/");
    io:println(rootIsDir); // @output true
    boolean linkIsDir = check jailed->isDirectory("/incoming");
    io:println(linkIsDir); // @output true
    ftp:FileInfo[] entries = check jailed->list("/incoming");
    io:println(entries.map(e => e.name)); // @output ["in.txt"]
    text = check jailed->getText("/incoming/in.txt");
    io:println(text); // @output inside
    check jailed->putText("/new/dir/out.txt", "written");
    text = check jailed->getText("/new/dir/out.txt");
    io:println(text); // @output written
    check jailed->copy("/x.txt", "/copied/x.txt");
    text = check jailed->getText("/copied/x.txt");
    io:println(text); // @output outside
    int size = check jailed->size("/big.bin");
    io:println(size); // @output 3221225472
    ftp:FileInfo[] rootEntries = check jailed->list("/");
    int[] listedSizes = from ftp:FileInfo e in rootEntries where e.name == "big.bin" select e.size;
    io:println(listedSizes); // @output [3221225472]
    check jailed->close();

    ftp:Listener looped = check new ({
        host: "127.0.0.1",
        port: jailPort(),
        auth: {credentials: {username: "ballerina", password: "ballerina"}},
        userDirIsRoot: true,
        path: "/looped",
        pollingInterval: 3600
    });
    check looped.attach(new LoopService());
    check looped.poll();
    foreach int _ in 0 ..< 200 {
        lock {
            if added.length() > 0 {
                break;
            }
        }
        runtime:sleep(0.05);
    }
    lock {
        io:println(added.toString()); // @output ["/looped/a.txt"]
    }
    check looped.gracefulStop();
}
