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
	"context"
	"encoding/pem"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	storeField      = "$ldapUserStore"
	memberUIDAttr   = "memberUid"
	defaultLdapPort = "389"
	defaultTLSPort  = "636"
)

// ldapUserStore is the native state of a ListenerLdapUserStoreBasicAuthProvider.
type ldapUserStore struct {
	rt                   *runtime.Runtime
	conn                 *goldap.Conn
	servers              []ldapServer
	connectTimeout       time.Duration
	readTimeout          time.Duration
	userSearchBases      []string
	userNameSearchFilter string
	groupSearchBases     []string
	groupNameAttribute   string
	groupNameListFilter  string
	membershipAttribute  string
}

// ldapServer is one entry of the connectionUrl list; tlsCfg is set for an ldaps:// URL.
type ldapServer struct {
	address string
	tlsCfg  *pal.TLSConfig
}

func initLdapConnectionExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		provider := args[0].(*values.Object)
		config := args[1].(*values.Map)

		store := &ldapUserStore{
			rt:                   rt,
			connectTimeout:       seconds(config, "connectionTimeout"),
			readTimeout:          seconds(config, "readTimeout"),
			userSearchBases:      strings.Split(stringField(config, "userSearchBase"), "#"),
			userNameSearchFilter: stringField(config, "userNameSearchFilter"),
			groupNameAttribute:   stringField(config, "groupNameAttribute"),
			groupNameListFilter:  stringField(config, "groupNameListFilter"),
			membershipAttribute:  stringField(config, "membershipAttribute"),
		}
		if v, ok := config.Get("groupSearchBase"); ok {
			list := v.(*values.List)
			for i := 0; i < list.Len(); i++ {
				store.groupSearchBases = append(store.groupSearchBases, strings.TrimSpace(list.Get(i).(string)))
			}
		}

		// As in jBallerina, a secure socket is validated even for an ldap:// URL, but only ldaps:// uses it.
		tlsCfg := &pal.TLSConfig{}
		if v, ok := config.Get("secureSocket"); ok {
			var err error
			if tlsCfg, err = trustConfig(rt, v.(*values.Map)); err != nil {
				return values.NewErrorWithMessage(err.Error()), nil
			}
		}
		// Like JNDI's provider URL, connectionUrl may list several space-separated URLs, tried in order.
		connectionURL := stringField(config, "connectionUrl")
		for _, u := range strings.Fields(connectionURL) {
			address, ldaps, ok := parseLdapURL(u)
			if !ok {
				return values.NewErrorWithMessage("Not an LDAP URL: " + connectionURL), nil
			}
			server := ldapServer{address: address}
			if ldaps {
				server.tlsCfg = tlsCfg
			}
			store.servers = append(store.servers, server)
		}
		if len(store.servers) == 0 {
			return values.NewErrorWithMessage("Not an LDAP URL: " + connectionURL), nil
		}

		// JNDI sends an empty connection password as an unauthenticated simple bind; go-ldap refuses it by default.
		bindRequest := goldap.NewSimpleBindRequest(stringField(config, "connectionName"), stringField(config, "connectionPassword"), nil)
		bindRequest.AllowEmptyPassword = true
		conn, err := store.dial(func(conn *goldap.Conn) error {
			_, err := conn.SimpleBind(bindRequest)
			return err
		})
		if err != nil {
			return values.NewErrorWithMessage(namingMessage(err)), nil
		}
		store.conn = conn
		provider.Put(storeField, store)
		return nil, nil
	}
}

func authenticateWithLdapExtern(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
	store := storeOf(args[0].(*values.Object))
	username := strings.TrimSpace(args[1].(string))
	password := args[2].(string)

	userDN, err := store.userDN(username)
	if err != nil {
		return values.NewErrorWithMessage(namingMessage(err)), nil
	}
	if userDN == "" {
		return values.NewErrorWithMessage("Username cannot be found in the directory."), nil
	}
	conn, err := store.dial(func(conn *goldap.Conn) error { return conn.Bind(userDN, password) })
	if err != nil {
		return values.NewErrorWithMessage(namingMessage(err)), nil
	}
	_ = conn.Close()
	return nil, nil
}

func getLdapGroupsExtern(strArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		store := storeOf(args[0].(*values.Object))
		groups, groupsErr := store.groups(args[1].(string))
		if groupsErr != nil {
			return groupsErr, nil
		}
		if len(groups) == 0 {
			return nil, nil
		}
		balGroups := make([]values.BalValue, len(groups))
		for i, g := range groups {
			balGroups[i] = g
		}
		atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), strArrTy)
		return values.NewList(strArrTy, atomic, false, nil, len(balGroups), balGroups), nil
	}
}

func storeOf(provider *values.Object) *ldapUserStore {
	v, _ := provider.Get(storeField)
	return v.(*ldapUserStore)
}

// dial connects and binds to the first server in the connectionUrl list that accepts both. As in JNDI, only
// invalid credentials stop it trying the next server; otherwise the last server's error is returned.
func (s *ldapUserStore) dial(bind func(*goldap.Conn) error) (*goldap.Conn, error) {
	var err error
	for _, server := range s.servers {
		var conn *goldap.Conn
		if conn, err = s.dialServer(server); err != nil {
			continue
		}
		if err = bind(conn); err == nil {
			return conn, nil
		}
		_ = conn.Close()
		if goldap.IsErrorWithCode(err, goldap.LDAPResultInvalidCredentials) {
			return nil, err
		}
	}
	return nil, err
}

func (s *ldapUserStore) dialServer(server ldapServer) (*goldap.Conn, error) {
	ctx := context.Background()
	if s.connectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.connectTimeout)
		defer cancel()
	}
	netConn, err := s.rt.Platform().Net.Dial(ctx, "tcp", server.address, "", server.tlsCfg)
	if err != nil {
		return nil, err
	}
	conn := goldap.NewConn(netConn, server.tlsCfg != nil)
	if s.readTimeout > 0 {
		conn.SetTimeout(s.readTimeout)
	}
	conn.Start()
	return conn, nil
}

// userDN returns the DN of the first entry matching the username search filter under any user search base,
// or "" when no entry matches.
func (s *ldapUserStore) userDN(username string) (string, error) {
	filter := strings.ReplaceAll(s.userNameSearchFilter, "?", escapeFilterValue(username))
	for _, base := range s.userSearchBases {
		result, err := s.conn.Search(goldap.NewSearchRequest(base, goldap.ScopeWholeSubtree, goldap.NeverDerefAliases,
			0, 0, false, filter, nil, nil))
		if err != nil {
			return "", err
		}
		if len(result.Entries) > 0 {
			return result.Entries[0].DN, nil
		}
	}
	return "", nil
}

func (s *ldapUserStore) groups(username string) ([]string, *values.Error) {
	userDN, err := s.userDN(username)
	if err != nil {
		return nil, values.NewErrorWithMessage(namingMessage(err))
	}
	if s.membershipAttribute == "" {
		return nil, values.NewErrorWithMessage("Membership attribute is not set in configurations.")
	}
	if userDN == "" {
		return nil, nil
	}
	var membershipValue string
	if strings.EqualFold(s.membershipAttribute, memberUIDAttr) {
		// A posixGroup's memberUid holds the user's leftmost RDN value, not the DN.
		dn, err := goldap.ParseDN(userDN)
		if err != nil {
			return nil, values.NewErrorWithMessage(err.Error())
		}
		membershipValue = goldap.EscapeFilter(dn.RDNs[0].Attributes[0].Value)
	} else {
		membershipValue = escapeDNForFilter(userDN)
	}
	filter := "(&" + s.groupNameListFilter + "(" + s.membershipAttribute + "=" + membershipValue + "))"

	var groups []string
	for _, base := range s.groupSearchBases {
		result, err := s.conn.Search(goldap.NewSearchRequest(base, goldap.ScopeWholeSubtree, goldap.NeverDerefAliases,
			0, 0, false, filter, []string{s.groupNameAttribute}, nil))
		if err != nil {
			return nil, values.NewErrorWithMessage(namingMessage(err))
		}
		for _, entry := range result.Entries {
			groups = append(groups, entry.GetEqualFoldAttributeValues(s.groupNameAttribute)...)
		}
	}
	return groups, nil
}

// parseLdapURL returns the host:port to dial and whether the URL selects LDAPS; ok is false for a non-LDAP URL.
func parseLdapURL(connectionURL string) (address string, ldaps bool, ok bool) {
	u, err := url.Parse(connectionURL)
	if err != nil || (!strings.EqualFold(u.Scheme, "ldap") && !strings.EqualFold(u.Scheme, "ldaps")) {
		return "", false, false
	}
	ldaps = strings.EqualFold(u.Scheme, "ldaps")
	host, port := u.Hostname(), u.Port()
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = defaultLdapPort
		if ldaps {
			port = defaultTLSPort
		}
	}
	return net.JoinHostPort(host, port), ldaps, true
}

// trustConfig builds the TLS trust from secureSocket.cert, a PEM or DER certificate file.
func trustConfig(rt *runtime.Runtime, secureSocket *values.Map) (*pal.TLSConfig, error) {
	cert, _ := secureSocket.Get("cert")
	path, ok := cert.(string)
	if !ok {
		return nil, errors.New("crypto:TrustStore is not yet supported for secureSocket.cert; use a certificate file path instead")
	}
	data, err := rt.Platform().FS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if block, _ := pem.Decode(data); block == nil {
		data = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	}
	return &pal.TLSConfig{CACertPEM: data}, nil
}

// namingMessage renders an LDAP result error in the form JNDI uses, so error causes read as in jBallerina.
func namingMessage(err error) string {
	var ldapErr *goldap.Error
	if errors.As(err, &ldapErr) && ldapErr.ResultCode != goldap.ErrorNetwork {
		text := ldapErr.Err.Error()
		if text == "" {
			text = goldap.LDAPResultCodeMap[ldapErr.ResultCode]
		}
		return "[LDAP: error code " + strconv.Itoa(int(ldapErr.ResultCode)) + " - " + text + "]"
	}
	return err.Error()
}

// escapeFilterValue escapes a username for a search filter as jBallerina does; `\*` is a literal `*`.
func escapeFilterValue(value string) string {
	return goldap.EscapeFilter(strings.ReplaceAll(value, `\*`, "*"))
}

// escapeDNForFilter escapes each RDN of a DN for use as a filter value, keeping its DN escapes as text.
func escapeDNForFilter(dn string) string {
	rdns := splitRDNs(dn)
	for i, rdn := range rdns {
		rdns[i] = goldap.EscapeFilter(rdn)
	}
	return strings.Join(rdns, ",")
}

// splitRDNs splits a DN at its unescaped, unquoted commas and trims the space around each RDN.
func splitRDNs(dn string) []string {
	var rdns []string
	start, quoted := 0, false
	for i := 0; i < len(dn); i++ {
		switch dn[i] {
		case '\\':
			i++
		case '"':
			quoted = !quoted
		case ',', ';':
			if !quoted {
				rdns = append(rdns, trimRDN(dn[start:i]))
				start = i + 1
			}
		}
	}
	return append(rdns, trimRDN(dn[start:]))
}

// trimRDN trims the space around an RDN, keeping a trailing space escaped with a backslash.
func trimRDN(rdn string) string {
	rdn = strings.TrimLeft(rdn, " ")
	trimmed := strings.TrimRight(rdn, " ")
	if len(trimmed) < len(rdn) && (len(trimmed)-len(strings.TrimRight(trimmed, `\`)))%2 == 1 {
		return trimmed + " "
	}
	return trimmed
}

func stringField(m *values.Map, name string) string {
	v, _ := m.Get(name)
	return v.(string)
}

func seconds(m *values.Map, name string) time.Duration {
	v, _ := m.Get(name)
	d := v.(*decimal.Decimal)
	return time.Duration(d.Float64() * float64(time.Second))
}
