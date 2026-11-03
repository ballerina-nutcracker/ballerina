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

# Represents the client that is used to publish messages to the server.
public client isolated class Client {

    # Creates a new `mqtt:Client`.
    # ```ballerina
    # mqtt:Client 'client = check new(mqtt:DEFAULT_URL, "client-unique-id");
    # ```
    #
    # + serverUri - URI of the server to connect to
    # + clientId - Unique ID of the client
    # + config - Optional configuration values to use for the client
    # + return - `mqtt:Error` if an error occurs while creating the client or else `()`
    public isolated function init(string serverUri, string clientId, *ClientConfiguration config) returns Error? {
        check self.externInit(serverUri, clientId, config);
    }

    # Publishes a message to a topic.
    # ```ballerina
    # mqtt:DeliveryToken token = check 'client->publish("mqtt/topic", message);
    # ```
    #
    # + topic - Topic to publish the message to
    # + message - Message to publish
    # + return - `mqtt:DeliveryToken` or else `mqtt:Error` if an error occurs while publishing
    isolated remote function publish(string topic, Message message) returns DeliveryToken|Error {
        return self.externPublish(topic, message);
    }

    # Subscribes to a given topic in the request response scenario.
    # ```ballerina
    # check 'client->subscribe([{topic: "mqtt/topic1", qos: 0}, {topic: "mqtt/topic2", qos: 1}]);
    # ```
    #
    # + subscriptions - The topics to be subscribed to
    # + return - `mqtt:Error` if an error occurs while subscribing or else `()`
    isolated remote function subscribe(string|string[]|Subscription|Subscription[] subscriptions) returns Error? {
        check self.externSubscribe(processSubscriptions(subscriptions));
    }

    # Receives messages from the server.
    # ```ballerina
    # stream<mqtt:Message, error?> responseStream = check 'client->receive();
    # ```
    # 
    # + T - Type of the stream to return
    # + return - `stream<Message, error?>` or else`mqtt:Error` if an error occurs while receiving the response
    isolated remote function receive(typedesc<stream<Message, error?>> T = <>) returns T|Error = external;

    # Closes the connection to the server.
    # ```ballerina
    # check 'client->close();
    # ```
    #
    # + return - `mqtt:Error` if an error occurs while closing or else `()`
    isolated remote function close() returns Error? {
        check self.externClose();
    }

    # Checks if the client is connected to the server.
    # ```ballerina
    # boolean isConnected = check 'client->isConnected;
    # ```
    #
    # + return - `true` if the client is connected, `mqtt:Error` if an error occurs in the process
    isolated remote function isConnected() returns boolean|Error {
        return self.externIsConnected();
    }

    # Disconnects the client from the server.
    # ```ballerina
    # check 'client->disconnect();
    # ```
    #
    # + return - `mqtt:Error` if an error occurs while disconnecting or else `()`
    isolated remote function disconnect() returns Error? {
        check self.externDisconnect();
    }

    # Reconnects the client to the server.
    # ```ballerina
    # check 'client->reconnect();
    # ```
    #
    # + return - `mqtt:Error` if an error occurs while reconnecting or else `()`
    isolated remote function reconnect() returns Error? {
        check self.externReconnect();
    }

    private isolated function externInit(string serverUri, string clientId, *ClientConfiguration config) returns Error? = external;

    private isolated function externPublish(string topic, Message message) returns DeliveryToken|Error = external;

    private isolated function externSubscribe(Subscription[] subscriptions) returns Error? = external;

    private isolated function externClose() returns Error? = external;

    private isolated function externIsConnected() returns boolean|Error = external;

    private isolated function externDisconnect() returns Error? = external;

    private isolated function externReconnect() returns Error? = external;
}
