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
	"context"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

var tlsVersionByName = map[string]uint16{
	"TLSv1.0": 0x0301,
	"TLSv1.1": 0x0302,
	"TLSv1.2": 0x0303,
	"TLSv1.3": 0x0304,
}

// connOf reads the live *goldap.Conn stashed on a Client instance by
// initLdapConnection. Every remote method calls this first.
func connOf(self *values.Object) *goldap.Conn {
	v, ok := self.Get("$conn")
	if !ok {
		return nil
	}
	conn, _ := v.(*goldap.Conn)
	return conn
}

// buildSecureSocketTLSConfig translates ldap:ClientSecureSocket into a
// pal.TLSConfig. Returns (nil, nil) when secureSocket is absent/disabled
// (plaintext connection).
func buildSecureSocketTLSConfig(rt *runtime.Runtime, secureSocket *values.Map) (*pal.TLSConfig, *values.Error) {
	if secureSocket == nil {
		return nil, nil
	}
	if v, ok := secureSocket.Get("enable"); ok {
		if enable, ok := v.(bool); ok && !enable {
			return nil, nil
		}
	}

	var tlsCfg pal.TLSConfig
	if v, ok := secureSocket.Get("verifyHostName"); ok {
		if verify, ok := v.(bool); ok && !verify {
			tlsCfg.InsecureSkipVerify = true
		}
	}
	if v, ok := secureSocket.Get("cert"); ok && v != nil {
		switch cert := v.(type) {
		case string:
			if cert != "" {
				data, err := rt.Platform().FS.ReadFile(cert)
				if err != nil {
					return nil, ldapError("clientSecureSocket.cert: " + err.Error())
				}
				tlsCfg.CACertPEM = data
			}
		case *values.Map:
			return nil, ldapError("clientSecureSocket.cert: crypto:TrustStore is not yet supported; " +
				"use a PEM certificate file path string instead")
		}
	}
	if v, ok := secureSocket.Get("tlsVersions"); ok {
		if list, ok := v.(*values.List); ok {
			for i := 0; i < list.Len(); i++ {
				name, ok := list.Get(i).(string)
				if !ok {
					continue
				}
				ver, found := tlsVersionByName[name]
				if !found {
					continue
				}
				if tlsCfg.MinVersion == 0 || ver < tlsCfg.MinVersion {
					tlsCfg.MinVersion = ver
				}
				if ver > tlsCfg.MaxVersion {
					tlsCfg.MaxVersion = ver
				}
			}
		}
	}
	return &tlsCfg, nil
}

// dialAndBind opens the TCP(+TLS) connection via PAL, wraps it as a
// *goldap.Conn, and performs the simple bind. domainName is used verbatim as
// the bind DN (see ConnectionConfig doc comment).
func dialAndBind(rt *runtime.Runtime, hostName string, port int64, domainName, password string, tlsCfg *pal.TLSConfig) (*goldap.Conn, *values.Error) {
	address := fmt.Sprintf("%s:%d", hostName, port)
	netConn, err := rt.Platform().Net.Dial(context.Background(), "tcp", address, "", tlsCfg)
	if err != nil {
		return nil, ldapError("failed to connect to " + address + ": " + err.Error())
	}
	conn := goldap.NewConn(netConn, tlsCfg != nil)
	conn.Start()

	if _, err := conn.SimpleBind(goldap.NewSimpleBindRequest(domainName, password, nil)); err != nil {
		_ = conn.Close()
		return nil, ldapErrorFromErr(err)
	}
	return conn, nil
}

func registerClientFunctions(rt *runtime.Runtime) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.initLdapConnection",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			config := args[1].(*values.Map)

			hostName, _ := config.Get("hostName")
			port, _ := config.Get("port")
			domainName, _ := config.Get("domainName")
			password, _ := config.Get("password")

			var secureSocket *values.Map
			if v, ok := config.Get("clientSecureSocket"); ok {
				secureSocket, _ = v.(*values.Map)
			}
			tlsCfg, tlsErr := buildSecureSocketTLSConfig(rt, secureSocket)
			if tlsErr != nil {
				return tlsErr, nil
			}

			conn, connErr := dialAndBind(rt, hostName.(string), port.(int64), domainName.(string), password.(string), tlsCfg)
			if connErr != nil {
				return connErr, nil
			}
			self.Put("$conn", conn)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$close",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			if conn := connOf(self); conn != nil {
				_ = conn.Close()
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$isConnected",
		func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			conn := connOf(self)
			return conn != nil && !conn.IsClosing(), nil
		})
}
