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

# Represents a MQTT listener endpoint.
public isolated class Listener {

    private final Subscription[] & readonly mqttSubscriptions;

    # Creates a new `mqtt:Listener`.
    # ```ballerina
    # mqtt:Listener 'listener = check new(mqtt:DEFAULT_URL, "listener-unique-id", "mqtt/topic");
    # ```
    #
    # + serverUri - The URI of the remote MQTT server
    # + clientId - The unique client ID to identify the listener
    # + subscriptions - The topics to be subscribed to
    # + return - `mqtt:Error` if an error occurs while creating the listener or else `()`
    public isolated function init(string serverUri, string clientId, string|string[]|Subscription|Subscription[] subscriptions, *ListenerConfiguration config) returns Error? {
        self.mqttSubscriptions = check toReadOnlySubscriptions(subscriptions);
        check self.externInit(serverUri, clientId, config);
    }

    # Starts the registered services.
    # ```ballerina
    # mqtt:Error? result = 'listener.'start();
    # ```
    #
    # + return - A `error` if an error is encountered while starting the server or else `()`
    public isolated function 'start() returns Error? {
        check self.externStart(self.mqttSubscriptions);
    };

    # Stops the MQTT listener gracefully.
    # ```ballerina
    # mqtt:Error? result = 'listener.gracefulStop();
    # ```
    #
    # + return - A `error` if an error is encountered during the listener-stopping process or else `()`
    public isolated function gracefulStop() returns Error? = external;

    # Stops the mqtt listener immediately.
    # ```ballerina
    # mqtt:Error? result = 'listener.immediateStop();
    # ```
    #
    # + return - A `error` if an error is encountered during the listener-stopping process or else `()`
    public isolated function immediateStop() returns Error? = external;

    # Attaches a service to the listener.
    # ```ballerina
    # mqtt:Error? result = 'listener.attach(mqttService);
    # ```
    #
    # + 'service - The service to be attached
    # + name - Name of the service
    # + return - A `error` if an error is encountered while attaching the service or else `()`
    public isolated function attach(Service 'service, string[]|string? name = ()) returns Error? = external;

    # Detaches a consumer service from the listener.
    # ```ballerina
    # mqtt:Error? result = 'listener.detach(mqttService);
    # ```
    #
    # + 'service - The service to be detached
    # + return - A `error` if an error is encountered while detaching a service or else `()`
    public isolated function detach(Service 'service) returns Error? = external;

    private isolated function externInit(string serverUri, string clientId, *ListenerConfiguration config) returns Error? = external;

    private isolated function externStart(Subscription[] topics) returns Error? = external;
}
