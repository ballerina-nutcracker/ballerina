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
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package native

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// listenerSettings are the connection settings of an http:ListenerConfiguration.
type listenerSettings struct {
	timeout time.Duration
	// keepAliveNever closes every HTTP/1.x connection after its response.
	keepAliveNever      bool
	gracefulStopTimeout time.Duration
	http2               pal.ServerHTTP2Config
	socket              pal.ServerSocketConfig
}

// gracefulStopLimit bounds gracefulStop when both gracefulStopTimeout and the listener timeout are 0.
const gracefulStopLimit = 60 * time.Second

// gracefulStopContext bounds gracefulStop by gracefulStopTimeout, or else by the listener timeout, or
// else by gracefulStopLimit.
func (l listenerSettings) gracefulStopContext() (context.Context, context.CancelFunc) {
	limit := cmp.Or(l.gracefulStopTimeout, l.timeout, gracefulStopLimit)
	return context.WithTimeout(context.Background(), limit)
}

// readListenerSettings reads the connection settings of cfg, and rejects a negative timeout or
// gracefulStopTimeout, as jBallerina does, and HTTP/2 settings net/http cannot advertise.
func readListenerSettings(cfg *values.Map) (listenerSettings, error) {
	timeout := decimalField(cfg, "timeout")
	if timeout.Cmp(decimalZero) < 0 {
		return listenerSettings{}, errors.New("Idle timeout cannot be negative. If you want to disable the timeout please use value 0") //nolint:staticcheck // jBallerina's message verbatim
	}
	gracefulStopTimeout := decimalField(cfg, "gracefulStopTimeout")
	if gracefulStopTimeout.Cmp(decimalZero) < 0 {
		return listenerSettings{}, errors.New("gracefulStop timeout cannot be negative")
	}
	maxStreams := intField(cfg, "http2MaxActiveStreams")
	if maxStreams < 1 || maxStreams > math.MaxInt32 {
		return listenerSettings{}, fmt.Errorf("http2MaxActiveStreams must be between 1 and %d: %d", math.MaxInt32, maxStreams)
	}
	window := intField(cfg, "http2InitialWindowSize")
	if window < 1 || window > math.MaxInt32 {
		return listenerSettings{}, fmt.Errorf("http2InitialWindowSize must be between 1 and %d: %d", math.MaxInt32, window)
	}
	http1, _ := cfg.Get("http1Settings")
	keepAlive, _ := http1.(*values.Map).Get("keepAlive")
	socketVal, _ := cfg.Get("socketConfig")
	socket := socketVal.(*values.Map)
	return listenerSettings{
		timeout:             decimalToDuration(timeout),
		keepAliveNever:      keepAlive == "NEVER",
		gracefulStopTimeout: decimalToDuration(gracefulStopTimeout),
		http2: pal.ServerHTTP2Config{
			MaxConcurrentStreams: maxStreams,
			InitialWindowSize:    window,
		},
		socket: pal.ServerSocketConfig{
			ReceiveBufferSize: intField(socket, "receiveBufferSize"),
			SendBufferSize:    intField(socket, "sendBufferSize"),
			TCPNoDelay:        boolField(socket, "tcpNoDelay"),
			KeepAlive:         boolField(socket, "keepAlive"),
		},
	}, nil
}

func decimalField(m *values.Map, name string) *decimal.Decimal {
	v, _ := m.Get(name)
	return v.(*decimal.Decimal)
}

func intField(m *values.Map, name string) int {
	v, _ := m.Get(name)
	return int(v.(int64))
}

func boolField(m *values.Map, name string) bool {
	v, _ := m.Get(name)
	return v.(bool)
}
