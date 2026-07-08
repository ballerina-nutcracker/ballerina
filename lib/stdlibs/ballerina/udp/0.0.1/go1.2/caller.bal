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

# Represents the caller object passed to udp service remote methods.
#
# + remoteHost - The hostname or the IP address of the remote host that sent the datagram
# + remotePort - The port number of the remote host that sent the datagram
public isolated client class Caller {

    public final string remoteHost;
    public final int remotePort;

    // Package-level private init to prevent object creation from user code.
    // Every Caller instance is constructed by the listener's native code.
    isolated function init(string remoteHost, int remotePort) {
        self.remoteHost = remoteHost;
        self.remotePort = remotePort;
    }

    # Sends the given data to the same remote host that sent the datagram this `Caller` was passed for.
    # ```ballerina
    # udp:Error? result = caller->sendBytes("msg".toBytes());
    # ```
    #
    # + data - The data that needs to be sent to the remote host
    # + return - `()` or else a `udp:Error` if the given data cannot be sent
    isolated remote function sendBytes(byte[] data) returns Error? = external;

    # Sends the given data to a remote destination as specified in the datagram.
    # ```ballerina
    # udp:Error? result = caller->sendDatagram({remoteHost: "localhost", remotePort: 48826, data: "msg".toBytes()});
    # ```
    #
    # + datagram - Contains the data to be sent to the remote host and the address of the remote host
    # + return - `()` or else a `udp:Error` if the given data cannot be sent
    isolated remote function sendDatagram(Datagram datagram) returns Error? = external;
}
