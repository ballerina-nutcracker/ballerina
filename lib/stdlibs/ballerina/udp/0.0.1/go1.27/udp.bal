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


# Initializes the UDP connection-oriented client based on the provided configurations.
public isolated client class ConnectClient {

    # Initializes the UDP connect client based on the provided configurations.
    # ```ballerina
    # udp:ConnectClient socketClient = check new ("www.remote.com", 80, {localHost: "localHost"});
    # ```
    #
    # + remoteHost - The hostname or the IP address of the remote host
    # + remotePort - The port number of the remote host
    # + config - Connection-oriented client-related configurations. Note: unlike jBallerina's
    #            `*ConnectClientConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(string remoteHost, int remotePort, ConnectClientConfiguration config = {}) returns Error? {
        return self.initConnectClient(remoteHost, remotePort, config);
    }

    private isolated function initConnectClient(string remoteHost, int remotePort, ConnectClientConfiguration config)
        returns Error? = external;

    # Sends the given data to the connected remote host.
    # ```ballerina
    # udp:Error? result = socketClient->writeBytes("msg".toBytes());
    # ```
    #
    # + data - The data that needs to be sent to the connected remote host
    # + return - `()` or else a `udp:Error` if the given data cannot be sent
    isolated remote function writeBytes(byte[] data) returns Error? = external;

    # Reads data only from the connected remote host.
    # ```ballerina
    # (readonly & byte[])|udp:Error result = socketClient->readBytes();
    # ```
    #
    # + return - A `readonly & byte[]` or else a `udp:Error` if the data cannot be read from the remote host
    isolated remote function readBytes() returns (readonly & byte[])|Error = external;

    # Closes the connection and frees up the occupied socket.
    # ```ballerina
    # udp:Error? closeResult = socketClient->close();
    # ```
    #
    # + return - A `udp:Error` if it can't close the connection or else `()`
    isolated remote function close() returns Error? = external;
}

// Configurations for the connection-oriented UDP client.
//
// Fields:
//   localHost - Local binding of the interface.
//   timeout   - The socket-reading timeout value in seconds. Defaults to 300 seconds (5 minutes).
public type ConnectClientConfiguration record {|
    string localHost?;
    decimal timeout = 300;
|};


// Represents udp module related errors.
// Note: distinct error types are not yet supported; Error is currently an alias for error.
public type Error error;


# Used for creating UDP server endpoints. A UDP server endpoint is capable of responding to
# remote callers. The `Listener` is responsible for initializing the endpoint using the
# provided configurations.
public isolated class Listener {

    # Initializes the UDP listener based on the provided configurations.
    # ```ballerina
    # udp:Listener udpServer = check new (8080);
    # ```
    #
    # + localPort - The port number to listen on
    # + config - Configurations related to the `udp:Listener`. Note: unlike jBallerina's
    #            `*ListenerConfiguration` (an included-record parameter, allowing named-arg-style
    #            `localHost = "x"` at the call site), this is a plain default-valued record
    #            parameter — pass a record literal (`{localHost: "x"}`) instead. See the
    #            README's Notable Behavioural Changes for why.
    public isolated function init(int localPort, ListenerConfiguration config = {}) returns Error? {
        return self.initListener(localPort, config);
    }

    private isolated function initListener(int localPort, ListenerConfiguration config) returns Error? = external;

    # Binds a service to the `udp:Listener`. Only one service may be attached at a time.
    #
    # + s - The service to attach
    # + name - Ignored; `udp:Listener` has no path-based routing
    # + return - `()` or else a `udp:Error` if a service is already attached
    public isolated function attach(Service s, string[]|string? name = ()) returns error? = external;

    # Starts the registered service.
    #
    # + return - An `error` if the listener fails to bind
    public isolated function 'start() returns error? = external;

    # Stops the service listener gracefully.
    #
    # + return - An `error` if the listener fails to stop
    public isolated function gracefulStop() returns error? = external;

    # Stops the service listener immediately.
    #
    # + return - An `error` if the listener fails to stop
    public isolated function immediateStop() returns error? = external;

    # Detaches the given service from the `udp:Listener`.
    #
    # + s - The service to detach
    # + return - `()` or else a `udp:Error` if the given service isn't the one currently attached
    public isolated function detach(Service s) returns error? = external;
}

// Configurations for the UDP listener.
//
// Fields:
//   localHost - The hostname or IP address to bind to; defaults to all interfaces.
public type ListenerConfiguration record {|
    string localHost?;
|};


// Represents the UDP listener service type.
// A udp:Service may declare the following optional remote methods:
//   remote function onBytes(readonly & byte[] data, Caller caller) returns byte[]|Datagram|Error?;
//   remote function onDatagram(readonly & Datagram datagram, Caller caller) returns byte[]|Datagram|Error?;
//   remote function onError(readonly & Error err);
public type Service service object {
};
