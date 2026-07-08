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

# Represents the caller object passed to tcp service remote methods.
#
# + remoteHost - The hostname or the IP address of the remote host
# + remotePort - The port number of the remote host
# + localHost - The bound hostname
# + localPort - The port number to which the socket is bound
# + id - The unique ID associated with the connection
public isolated client class Caller {

    public final string remoteHost;
    public final int remotePort;
    public final string localHost;
    public final int localPort;
    public final string id;

    // Package-level private init to prevent object creation from user code.
    // Every Caller instance is constructed by the listener's native code.
    isolated function init(string remoteHost, int remotePort, string localHost, int localPort, string id) {
        self.remoteHost = remoteHost;
        self.remotePort = remotePort;
        self.localHost = localHost;
        self.localPort = localPort;
        self.id = id;
    }

    # Sends the given data to the connected remote host.
    #
    # + data - The data to send to the remote host
    # + return - `()` or else a `tcp:Error` if the given data cannot be sent
    isolated remote function writeBytes(byte[] data) returns Error? = external;

    # Closes the connection.
    #
    # + return - `()` or else a `tcp:Error` if the connection cannot be properly closed
    isolated remote function close() returns Error? = external;
}
