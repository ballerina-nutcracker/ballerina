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

// A minimal, hand-rolled LDAPv3 protocol server used only to exercise the
// ldap:Client corpus fixtures against real wire traffic. It supports exactly
// the operations the fixtures use (Bind/Add/Delete/Modify/ModifyDN/Compare/
// Search with simple equality/and/or/not/present filters) and nothing more —
// this is test infrastructure, not a spec-complete LDAP server.
//
// github.com/jimlambrt/gldap was evaluated first but its wire dispatcher
// doesn't decode ModifyDN/Compare request PDUs at all (connection is closed
// on receipt), so it can't exercise the full ldap:Client surface; this file
// is built directly on go-asn1-ber instead (already a transitive dependency
// of go-ldap).

package extern_test

import (
	"crypto/tls"
	"net"
	"strings"
	"sync"

	ber "github.com/go-asn1-ber/asn1-ber"
	goldap "github.com/go-ldap/ldap/v3"
)

type fakeLdapEntry struct {
	dn    string
	attrs map[string][]string // lowercased attribute name -> values
}

type fakeLdapServer struct {
	mu      sync.Mutex
	entries []*fakeLdapEntry
	bindDN  string
	bindPW  string
	ln      net.Listener
}

func newFakeLdapServer(t interface{ Fatalf(string, ...any) }, bindDN, bindPW string, tlsConfig *tls.Config) *fakeLdapServer {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	if tlsConfig != nil {
		ln = tls.NewListener(ln, tlsConfig)
	}
	s := &fakeLdapServer{bindDN: bindDN, bindPW: bindPW, ln: ln}
	go s.acceptLoop()
	return s
}

func (s *fakeLdapServer) addr() string {
	return s.ln.Addr().String()
}

func (s *fakeLdapServer) close() {
	_ = s.ln.Close()
}

func (s *fakeLdapServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *fakeLdapServer) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	for {
		msg, err := ber.ReadPacket(conn)
		if err != nil {
			return
		}
		if len(msg.Children) < 2 {
			return
		}
		messageID, _ := msg.Children[0].Value.(int64)
		op := msg.Children[1]
		if !s.dispatch(conn, messageID, op) {
			return
		}
	}
}

// dispatch handles one request PDU. Returns false to close the connection
// (UnbindRequest, or an operation type this fake server doesn't support).
func (s *fakeLdapServer) dispatch(conn net.Conn, messageID int64, op *ber.Packet) bool {
	switch op.Tag {
	case ber.Tag(goldap.ApplicationBindRequest):
		s.handleBind(conn, messageID, op)
	case ber.Tag(goldap.ApplicationUnbindRequest):
		return false
	case ber.Tag(goldap.ApplicationAddRequest):
		s.handleAdd(conn, messageID, op)
	case ber.Tag(goldap.ApplicationDelRequest):
		s.handleDelete(conn, messageID, op)
	case ber.Tag(goldap.ApplicationModifyRequest):
		s.handleModify(conn, messageID, op)
	case ber.Tag(goldap.ApplicationModifyDNRequest):
		s.handleModifyDN(conn, messageID, op)
	case ber.Tag(goldap.ApplicationCompareRequest):
		s.handleCompare(conn, messageID, op)
	case ber.Tag(goldap.ApplicationSearchRequest):
		s.handleSearch(conn, messageID, op)
	default:
		return false
	}
	return true
}

// packetString returns a packet's content as a string regardless of tag
// class: universal-tagged leaves (e.g. plain LDAPDN/AttributeValue OCTET
// STRINGs) get their content decoded into .Value by the ber decoder, but
// context-tagged leaves (e.g. Bind's [0] simple password, ModifyDN's [0]
// newSuperior, Filter's [3] equalityMatch operands) only get raw bytes into
// .Data — see go-asn1-ber's readPacket, which only populates .Value inside
// its `if p.ClassType == ClassUniversal` branch.
func packetString(p *ber.Packet) string {
	if s, ok := p.Value.(string); ok {
		return s
	}
	if p.Data != nil {
		return p.Data.String()
	}
	return string(p.ByteValue)
}

func sendMessage(conn net.Conn, messageID int64, op *ber.Packet) {
	envelope := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "LDAP Response")
	envelope.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, messageID, "MessageID"))
	envelope.AppendChild(op)
	_, _ = conn.Write(envelope.Bytes())
}

func ldapResultOp(appTag ber.Tag, resultCode uint16, matchedDN, diagMsg string) *ber.Packet {
	op := ber.Encode(ber.ClassApplication, ber.TypeConstructed, appTag, nil, "")
	op.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagEnumerated, int64(resultCode), "resultCode"))
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, matchedDN, "matchedDN"))
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, diagMsg, "diagnosticMessage"))
	return op
}

func (s *fakeLdapServer) findEntry(dn string) *fakeLdapEntry {
	for _, e := range s.entries {
		if strings.EqualFold(e.dn, dn) {
			return e
		}
	}
	return nil
}

func (s *fakeLdapServer) handleBind(conn net.Conn, messageID int64, op *ber.Packet) {
	name := packetString(op.Children[1])
	password := packetString(op.Children[2])

	s.mu.Lock()
	ok := strings.EqualFold(name, s.bindDN) && password == s.bindPW
	s.mu.Unlock()

	code := uint16(goldap.LDAPResultInvalidCredentials)
	if ok {
		code = goldap.LDAPResultSuccess
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationBindResponse), code, "", ""))
}

// decodePartialAttributes decodes an AttributeList/PartialAttributeList
// (SEQUENCE OF SEQUENCE { type, vals SET OF }), the shared shape used by
// AddRequest's attributes and ModifyRequest's Change.modification.
func decodePartialAttributes(seq *ber.Packet) map[string][]string {
	attrs := map[string][]string{}
	for _, partial := range seq.Children {
		name := strings.ToLower(packetString(partial.Children[0]))
		vals := make([]string, len(partial.Children[1].Children))
		for i, v := range partial.Children[1].Children {
			vals[i] = packetString(v)
		}
		attrs[name] = vals
	}
	return attrs
}

func (s *fakeLdapServer) handleAdd(conn net.Conn, messageID int64, op *ber.Packet) {
	dn := packetString(op.Children[0])
	attrs := decodePartialAttributes(op.Children[1])

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findEntry(dn) != nil {
		sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationAddResponse), goldap.LDAPResultEntryAlreadyExists, "", "entry already exists"))
		return
	}
	s.entries = append(s.entries, &fakeLdapEntry{dn: dn, attrs: attrs})
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationAddResponse), goldap.LDAPResultSuccess, "", ""))
}

func (s *fakeLdapServer) handleDelete(conn net.Conn, messageID int64, op *ber.Packet) {
	dn := packetString(op)

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if strings.EqualFold(e.dn, dn) {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationDelResponse), goldap.LDAPResultSuccess, "", ""))
			return
		}
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationDelResponse), goldap.LDAPResultNoSuchObject, "", "no such object"))
}

func (s *fakeLdapServer) handleModify(conn net.Conn, messageID int64, op *ber.Packet) {
	dn := packetString(op.Children[0])

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.findEntry(dn)
	if entry == nil {
		sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationModifyResponse), goldap.LDAPResultNoSuchObject, "", "no such object"))
		return
	}
	for _, change := range op.Children[1].Children {
		// change.Children[0] = operation (0=add,1=delete,2=replace); this
		// fixture set only ever exercises replace, so every operation
		// replaces the named attribute for simplicity.
		mod := change.Children[1]
		name := strings.ToLower(packetString(mod.Children[0]))
		vals := make([]string, len(mod.Children[1].Children))
		for i, v := range mod.Children[1].Children {
			vals[i] = packetString(v)
		}
		entry.attrs[name] = vals
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationModifyResponse), goldap.LDAPResultSuccess, "", ""))
}

func (s *fakeLdapServer) handleModifyDN(conn net.Conn, messageID int64, op *ber.Packet) {
	dn := packetString(op.Children[0])
	newRDN := packetString(op.Children[1])

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.findEntry(dn)
	if entry == nil {
		sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationModifyDNResponse), goldap.LDAPResultNoSuchObject, "", "no such object"))
		return
	}
	_, parentDN, found := strings.Cut(dn, ",")
	if !found {
		parentDN = ""
	}
	if parentDN == "" {
		entry.dn = newRDN
	} else {
		entry.dn = newRDN + "," + parentDN
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationModifyDNResponse), goldap.LDAPResultSuccess, "", ""))
}

func (s *fakeLdapServer) handleCompare(conn net.Conn, messageID int64, op *ber.Packet) {
	dn := packetString(op.Children[0])
	attrName := strings.ToLower(packetString(op.Children[1].Children[0]))
	assertion := packetString(op.Children[1].Children[1])

	s.mu.Lock()
	entry := s.findEntry(dn)
	s.mu.Unlock()
	if entry == nil {
		sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationCompareResponse), goldap.LDAPResultNoSuchObject, "", "no such object"))
		return
	}
	matched := false
	for _, v := range entry.attrs[attrName] {
		if strings.EqualFold(v, assertion) {
			matched = true
			break
		}
	}
	code := uint16(goldap.LDAPResultCompareFalse)
	if matched {
		code = goldap.LDAPResultCompareTrue
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationCompareResponse), code, "", ""))
}

// evalFilter walks a compiled RFC 4511 Filter packet (the same shape
// ldap.CompileFilter produces client-side — this is the raw wire encoding,
// decoded as-is) against one entry's attributes. Supports and/or/not,
// equality, and present; other filter choices are treated as non-matching
// since the fixtures don't exercise them.
func evalFilter(pkt *ber.Packet, attrs map[string][]string) bool {
	switch pkt.Tag {
	case ber.Tag(goldap.FilterAnd):
		for _, c := range pkt.Children {
			if !evalFilter(c, attrs) {
				return false
			}
		}
		return true
	case ber.Tag(goldap.FilterOr):
		for _, c := range pkt.Children {
			if evalFilter(c, attrs) {
				return true
			}
		}
		return false
	case ber.Tag(goldap.FilterNot):
		return len(pkt.Children) == 1 && !evalFilter(pkt.Children[0], attrs)
	case ber.Tag(goldap.FilterEqualityMatch):
		name := strings.ToLower(packetString(pkt.Children[0]))
		val := packetString(pkt.Children[1])
		for _, v := range attrs[name] {
			if strings.EqualFold(v, val) {
				return true
			}
		}
		return false
	case ber.Tag(goldap.FilterPresent):
		_, ok := attrs[strings.ToLower(packetString(pkt))]
		return ok
	default:
		return false
	}
}

func (s *fakeLdapServer) handleSearch(conn net.Conn, messageID int64, op *ber.Packet) {
	baseDN := packetString(op.Children[0])
	scope, _ := op.Children[1].Value.(int64)
	filterPkt := op.Children[6]
	var reqAttrs []string
	for _, a := range op.Children[7].Children {
		reqAttrs = append(reqAttrs, strings.ToLower(packetString(a)))
	}

	s.mu.Lock()
	var matches []*fakeLdapEntry
	for _, e := range s.entries {
		if !dnInScope(e.dn, baseDN, int(scope)) {
			continue
		}
		if evalFilter(filterPkt, e.attrs) {
			matches = append(matches, e)
		}
	}
	s.mu.Unlock()

	for _, e := range matches {
		sendMessage(conn, messageID, searchResultEntryOp(e, reqAttrs))
	}
	sendMessage(conn, messageID, ldapResultOp(ber.Tag(goldap.ApplicationSearchResultDone), goldap.LDAPResultSuccess, "", ""))
}

// dnInScope approximates LDAP scope semantics using plain suffix matching on
// the fixture's flat DN strings — sufficient for the small, hand-built
// directories these tests populate.
func dnInScope(dn, baseDN string, scope int) bool {
	dn, baseDN = strings.ToLower(dn), strings.ToLower(baseDN)
	switch scope {
	case goldap.ScopeBaseObject:
		return dn == baseDN
	case goldap.ScopeSingleLevel:
		if dn == baseDN || !strings.HasSuffix(dn, ","+baseDN) {
			return false
		}
		rest := strings.TrimSuffix(dn, ","+baseDN)
		return !strings.Contains(rest, ",")
	default: // ScopeWholeSubtree, ScopeChildren
		return dn == baseDN || strings.HasSuffix(dn, ","+baseDN)
	}
}

func searchResultEntryOp(e *fakeLdapEntry, reqAttrs []string) *ber.Packet {
	op := ber.Encode(ber.ClassApplication, ber.TypeConstructed, ber.Tag(goldap.ApplicationSearchResultEntry), nil, "SearchResultEntry")
	op.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, e.dn, "objectName"))
	attrsSeq := ber.NewSequence("PartialAttributeList")
	for name, vals := range e.attrs {
		if len(reqAttrs) > 0 && !containsFold(reqAttrs, name) {
			continue
		}
		partial := ber.NewSequence("PartialAttribute")
		partial.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, name, "type"))
		valsSet := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSet, nil, "vals")
		for _, v := range vals {
			valsSet.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, v, "AttributeValue"))
		}
		partial.AppendChild(valsSet)
		attrsSeq.AppendChild(partial)
	}
	op.AppendChild(attrsSeq)
	return op
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
