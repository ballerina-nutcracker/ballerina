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

isolated function processSubscriptions(string|string[]|Subscription|Subscription[] subscriptions) returns Subscription[] {
    Subscription[] mqttSubscriptions = [];
    if subscriptions is Subscription {
        mqttSubscriptions = [subscriptions];
    } else if subscriptions is string {
        mqttSubscriptions = [{topic: subscriptions}];
    } else if subscriptions is string[] {
        foreach string topic in subscriptions {
            mqttSubscriptions.push({topic: topic});
        }
    } else if subscriptions is Subscription[] {
        mqttSubscriptions = checkpanic subscriptions.cloneWithType();
    }
    return mqttSubscriptions;
}

isolated function createError(string message, int? reasonCode, string? causeMessage) returns Error {
    error? cause = causeMessage is string ? error(causeMessage) : ();
    if reasonCode is int {
        return error Error(message, cause, reasonCode = reasonCode);
    }
    return error Error(message, cause);
}

isolated function createMessage(byte[] payload, int qos, boolean retained, boolean duplicate, int messageId,
        string topic, string? responseTopic, byte[]? correlationData) returns Message {
    MessageProperties properties = {};
    if responseTopic is string {
        properties.responseTopic = responseTopic;
    }
    if correlationData is byte[] {
        properties.correlationData = correlationData;
    }
    return {payload, qos, retained, duplicate, messageId, topic, properties};
}

isolated function createDeliveryToken(int messageId, string topic) returns DeliveryToken {
    return {messageId, topic};
}

function createCaller() returns Caller {
    return new;
}

function createStreamIterator() returns StreamIterator {
    return new;
}

isolated function serviceParamTypes() returns [typedesc<Message>, typedesc<Caller>, typedesc<Error>, typedesc<DeliveryToken>] {
    return [Message, Caller, Error, DeliveryToken];
}
