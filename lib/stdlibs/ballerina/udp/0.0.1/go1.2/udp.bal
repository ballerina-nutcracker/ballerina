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

// A self-contained, independent entity of data carrying sufficient information
// to be routed from the source to the destination nodes without reliance
// on earlier exchanges between the nodes and the transporting network.
//
// Fields:
//   remoteHost - The hostname or the IP address of the remote host.
//   remotePort - The port number of the remote host.
//   data       - The content which needs to be transported to the remote host.
public type Datagram record {|
    string remoteHost;
    int remotePort;
    byte[] data;
|};

# Initializes the UDP connectionless client based on the provided configurations.
public isolated client class Client {

    # Initializes the UDP connectionless client based on the provided configurations.
    # ```ballerina
    # udp:Client socketClient = check new ({localHost: "localhost"});
    # ```
    #
    # + config - Connectionless client-related configurations. Note: unlike jBallerina's
    #            `*ClientConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(ClientConfiguration config = {}) returns Error? {
        return self.initClient(config);
    }

    private isolated function initClient(ClientConfiguration config) returns Error? = external;

    # Sends the given data to the specified remote host.
    # ```ballerina
    # udp:Error? result = socketClient->sendDatagram({remoteHost: "localhost", remotePort: 48826, data: "msg".toBytes()});
    # ```
    #
    # + datagram - Contains the data to be sent to the remote host and the address of the remote host
    # + return - `()` or else a `udp:Error` if the given data cannot be sent
    isolated remote function sendDatagram(Datagram datagram) returns Error? = external;

    # Reads data from the remote host.
    # ```ballerina
    # udp:Datagram|udp:Error result = socketClient->receiveDatagram();
    # ```
    #
    # + return - A `readonly & udp:Datagram` or else a `udp:Error` if the data cannot be read from the remote host
    isolated remote function receiveDatagram() returns (readonly & Datagram)|Error = external;

    # Closes the client and frees up the occupied socket.
    # ```ballerina
    # udp:Error? closeResult = socketClient->close();
    # ```
    #
    # + return - A `udp:Error` if it can't close the socket or else `()`
    isolated remote function close() returns Error? = external;
}

// Configurations for the connectionless UDP client.
//
// Fields:
//   localHost - Local binding of the interface.
//   timeout   - The socket-reading timeout value in seconds. Defaults to 300 seconds (5 minutes).
public type ClientConfiguration record {|
    string localHost?;
    decimal timeout = 300;
|};
