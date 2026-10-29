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

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "mqtt"
)

type mqttModule struct {
	rt        *runtime.Runtime
	byteArrTy semtypes.SemType
}

func initMqttModule(rt *runtime.Runtime) {
	byteArrDef := semtypes.NewListDefinition()
	m := &mqttModule{
		rt:        rt,
		byteArrTy: byteArrDef.Define(rt.GetTypeEnv(), nil, semtypes.ListRest(semtypes.Byte)),
	}
	m.registerClient()
	m.registerListener()
}

func init() {
	runtime.RegisterModuleInitializer(initMqttModule)
}

// callHelper invokes a private Ballerina function of this module, which builds values of types
// declared in mqtt (Error, Message, DeliveryToken, ...) so they carry their declared types.
func callHelper(ctx *extern.Context, name string, args ...values.BalValue) (values.BalValue, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, name)
	if !ok {
		return nil, fmt.Errorf("mqtt: internal helper function %q not found", name)
	}
	return ctx.InvokeFunction(handle, args)
}

func toBalError(ctx *extern.Context, err *mqttError) (values.BalValue, error) {
	var reasonCode, cause values.BalValue
	if err.reasonCode != nil {
		reasonCode = int64(*err.reasonCode)
	}
	if err.cause != nil {
		cause = err.cause.Error()
	}
	return callHelper(ctx, "createError", err.message, reasonCode, cause)
}

// result converts an optional mqttError into the `Error?` an extern returns.
func result(ctx *extern.Context, err *mqttError) (values.BalValue, error) {
	if err != nil {
		return toBalError(ctx, err)
	}
	return nil, nil
}

func (m *mqttModule) bytes(ctx *extern.Context, data []byte) *values.List {
	return values.ByteSliceToList(m.byteArrTy, ctx.TypeEnv(), data)
}

func (m *mqttModule) toBalMessage(ctx *extern.Context, msg receivedMessage) (values.BalValue, error) {
	p := msg.publish
	var responseTopic, correlationData values.BalValue
	if p.Properties != nil {
		if p.Properties.ResponseTopic != "" {
			responseTopic = p.Properties.ResponseTopic
		}
		if p.Properties.CorrelationData != nil {
			correlationData = m.bytes(ctx, p.Properties.CorrelationData)
		}
	}
	return callHelper(ctx, "createMessage", m.bytes(ctx, p.Payload), int64(p.QoS), p.Retain, p.Duplicate(),
		int64(p.PacketID), p.Topic, responseTopic, correlationData)
}

func deliveryToken(ctx *extern.Context, packetID uint16, topic string) (values.BalValue, error) {
	return callHelper(ctx, "createDeliveryToken", int64(packetID), topic)
}

// printError writes an error the way jBallerina's BError.printStackTrace() starts:
// "error: <message> <detail>". Errors here carry no call stack, so no frames follow.
func (m *mqttModule) printError(v values.BalValue, err error) {
	var line string
	switch {
	case err != nil:
		line = "error: " + err.Error()
	case v != nil:
		balErr, ok := v.(*values.Error)
		if !ok {
			return
		}
		line = "error: " + balErr.Message
		if balErr.Detail != nil && balErr.Detail.Len() > 0 {
			line += " " + values.String(balErr.Detail, map[uintptr]bool{})
		}
	default:
		return
	}
	_, _ = m.rt.Platform().IO.Stderr([]byte(line + "\n"))
}
