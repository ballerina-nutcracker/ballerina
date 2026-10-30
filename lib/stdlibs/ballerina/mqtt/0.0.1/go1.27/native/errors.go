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

package native

import (
	"fmt"
	"strings"
)

// Reason codes and messages mirror the Paho Java MQTTv5 client (messages.properties) so that
// mqtt:Error values carry the same message and reasonCode as on jBallerina.
const (
	reasonClientException       = 0
	reasonClientTimeout         = 32000
	reasonClientConnected       = 32100
	reasonClientDisconnected    = 32101
	reasonClientDisconnecting   = 32102
	reasonServerConnectError    = 32103
	reasonClientNotConnected    = 32104
	reasonSocketFactoryMismatch = 32105
	reasonConnectionLost        = 32109
	reasonConnectInProgress     = 32110
	reasonClientClosed          = 32111
	reasonServerDisconnected    = 32204
)

var reasonMessages = map[int]string{
	1:     "Invalid protocol version",
	2:     "Invalid client ID",
	3:     "Broker unavailable",
	4:     "Bad user name or password",
	5:     "Not authorized to connect",
	6:     "Unexpected error",
	32000: "Timed out waiting for a response from the server",
	32001: "Internal error, caused by no new message IDs being available",
	32002: "Timed out while waiting to write messages to the server",
	32100: "Client is connected",
	32101: "Client is disconnected",
	32102: "Client is currently disconnecting",
	32103: "Unable to connect to server",
	32104: "Client is not connected",
	32105: "The specified SocketFactory type does not match the broker URI",
	32106: "SSL configuration error",
	32107: "Disconnecting is not allowed from a callback method",
	32108: "Unrecognized packet",
	32109: "Connection lost",
	32110: "Connect already in progress",
	32111: "Client is closed",
	32200: "Persistence already in use",
	32201: "Token already in use",
	32202: "Too many publishes in progress",
	32204: "The Server Disconnected the client.",
	32301: "An invalid topic alias combination was received.",
	50000: "Invalid Message Property Identifier",
	50001: "Invalid Return code",
	50002: "Malformed Packet",
	50003: "Unsupported Protocol",
	50004: "An invalid topic alias combination was received in an incoming message.",
	50005: "Duplicate property in Packet",
	51001: "Incoming packet too large.",
	51002: "Outgoing packet too large.",
	16:    "No matching subscribers.",
	17:    "No subscription existed.",
	24:    "Continue authentication.",
	25:    "Re-authenticate.",
	128:   "Unspecified error.",
	129:   "Malformed packet.",
	130:   "Protocol error.",
	131:   "Implementation specific error.",
	132:   "Unsupported protocol version.",
	133:   "Client identifier not valid.",
	134:   "Bad User Name or Password.",
	135:   "Not authorized.",
	136:   "Server unavailable.",
	137:   "Server busy.",
	138:   "Banned.",
	139:   "Server shutting down.",
	140:   "Bad authentication method.",
	141:   "Keep Alive timeout.",
	142:   "Session taken over.",
	143:   "Topic Filter invalid.",
	144:   "Topic Name invalid.",
	145:   "Packet identifier in use.",
	146:   "Packet identifier not found.",
	147:   "Receive Maximum exceeded.",
	148:   "Topic Alias invalid.",
	149:   "Packet too large.",
	150:   "Message rate too high.",
	151:   "Quota exceeded.",
	152:   "Administrative action.",
	153:   "Payload format invalid.",
	154:   "Retain not supported.",
	155:   "QoS not supported",
	156:   "Use another server.",
	157:   "Server moved.",
	158:   "Shared Subscriptions not supported.",
	159:   "Connection rate exceeded.",
	160:   "Maximum connect time.",
	161:   "Subscription Identifiers not supported.",
}

// mqttError is the Go-side form of an mqtt:Error; it is converted to the Ballerina value at
// the extern boundary by toBalError.
type mqttError struct {
	message    string
	reasonCode *int
	cause      error
}

func (e *mqttError) Error() string {
	return e.message
}

func newError(message string) *mqttError {
	return &mqttError{message: message}
}

func newErrorWithCause(message string, cause error) *mqttError {
	return &mqttError{message: message, cause: cause}
}

func newReasonError(reasonCode int, cause error) *mqttError {
	rc := reasonCode
	return &mqttError{message: reasonMessage(reasonCode), reasonCode: &rc, cause: cause}
}

func reasonMessage(reasonCode int) string {
	if msg, ok := reasonMessages[reasonCode]; ok {
		return msg
	}
	return fmt.Sprintf("Untranslated MqttException - RC: %d", reasonCode)
}

func serverDisconnectedError(disconnectReason byte) *mqttError {
	rc := reasonServerDisconnected
	return &mqttError{
		message:    fmt.Sprintf("%s Disconnect RC: %d", reasonMessages[reasonServerDisconnected], disconnectReason),
		reasonCode: &rc,
	}
}

func validateQoS(qos int64) *mqttError {
	if qos < 0 || qos > 2 {
		return newError(fmt.Sprintf("Invalid QoS value: %d. Allowed values are 0, 1 and 2", qos))
	}
	return nil
}

// validateTopic applies the Paho Java MqttTopicValidator rules: topic names must not contain
// wildcards, and topic filters may only use '#' as the last level and '+' as a whole level.
func validateTopic(topic string, wildcardAllowed bool) *mqttError {
	if len(topic) < 1 || len(topic) > 65535 {
		return newError("Invalid topic length, should be in range[1, 65535]!")
	}
	if !wildcardAllowed {
		if strings.ContainsAny(topic, "#+") {
			return newError("The topic name MUST NOT contain any wildcard characters (#+)")
		}
		return nil
	}
	if topic == "#" || topic == "+" {
		return nil
	}
	if strings.Count(topic, "#") > 1 || (strings.Contains(topic, "#") && !strings.HasSuffix(topic, "/#")) {
		return newError("Invalid usage of multi-level wildcard in topic string: " + topic)
	}
	for i := 0; i < len(topic); i++ {
		if topic[i] != '+' {
			continue
		}
		if (i > 0 && topic[i-1] != '/') || (i < len(topic)-1 && topic[i+1] != '/') {
			return newError(fmt.Sprintf("Invalid usage of single-level wildcard in topic string '%s'!", topic))
		}
	}
	return nil
}
