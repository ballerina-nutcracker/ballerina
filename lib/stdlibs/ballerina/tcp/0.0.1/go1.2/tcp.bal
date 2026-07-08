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

# Initializes the TCP connection client based on the provided configurations.
public isolated client class Client {

    # Initializes the TCP client based on the provided configurations.
    # ```ballerina
    # tcp:Client socketClient = check new ("www.remote.com", 80, {localHost: "localHost"});
    # ```
    #
    # + remoteHost - The hostname or the IP address of the remote host
    # + remotePort - The port number of the remote host
    # + config - Connection-oriented client-related configurations. Note: unlike jBallerina's
    #            `*ClientConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(string remoteHost, int remotePort, ClientConfiguration config = {}) returns Error? {
        return self.initTcpConnection(remoteHost, remotePort, config);
    }

    private isolated function initTcpConnection(string remoteHost, int remotePort, ClientConfiguration config)
        returns Error? = external;

    # Sends the given data to the connected remote host.
    # ```ballerina
    # tcp:Error? result = socketClient->writeBytes("msg".toBytes());
    # ```
    #
    # + data - The data that needs to be sent to the connected remote host
    # + return - `()` or else a `tcp:Error` if the given data cannot be sent
    isolated remote function writeBytes(byte[] data) returns Error? = external;

    # Reads data from the connected remote host.
    # ```ballerina
    # (readonly & byte[])|tcp:Error result = socketClient->readBytes();
    # ```
    #
    # + return - The `readonly & byte[]` or else a `tcp:Error` if the data cannot be read from the remote host
    isolated remote function readBytes() returns (readonly & byte[])|Error = external;

    # Closes the connection.
    # ```ballerina
    # tcp:Error? closeResult = socketClient->close();
    # ```
    #
    # + return - A `tcp:Error` if it cannot close the connection or else `()`
    isolated remote function close() returns Error? = external;
}

// Configurations for the connection-oriented TCP client.
//
// Fields:
//   localHost    - Local binding interface hostname or IP address.
//   timeout      - The socket read timeout, in seconds. Defaults to 300 seconds (5 minutes).
//   writeTimeout - The socket write timeout, in seconds. Defaults to 300 seconds (5 minutes).
//   secureSocket - The TLS configurations for the client.
public type ClientConfiguration record {|
    string localHost?;
    decimal timeout = 300;
    decimal writeTimeout = 300;
    ClientSecureSocket secureSocket?;
|};
