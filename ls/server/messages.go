// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"encoding/json"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

// Classification precedes lifecycle checks, and only complete frames reach here.
func (s *Server) handlePayload(payload []byte) error {
	if !json.Valid(payload) {
		return s.writeError(nil, rpcParseError, "parse error")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || fields == nil {
		return s.writeError(nil, rpcInvalidRequest, "invalid request")
	}
	rawMethod, hasMethod := fields["method"]
	if !hasMethod && (fields["result"] != nil || fields["error"] != nil) {
		return nil
	}
	var method string
	methodOK := len(rawMethod) > 0 && rawMethod[0] == '"' && json.Unmarshal(rawMethod, &method) == nil
	var version string
	versionOK := json.Unmarshal(fields["jsonrpc"], &version) == nil && version == "2.0"
	id, hasID := fields["id"]
	if methodOK && !hasID {
		if !versionOK {
			return nil
		}
		return s.handleMessage(protocol.Message{JSONRPC: version, Method: method, Params: fields["params"]})
	}
	if _, valid := requestIDKey(id); !valid {
		id = nil
	}
	if !methodOK || !versionOK || id == nil {
		return s.writeError(id, rpcInvalidRequest, "invalid request")
	}
	return s.handleMessage(protocol.Message{JSONRPC: version, ID: id, Method: method, Params: fields["params"]})
}
