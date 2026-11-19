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
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	imapStateField       = "$imap"
	imapConnectPrefix    = "Error while connecting to the IMAP store: "
	imapPropertiesPrefix = "Error while initializing IMAP client properties: "
	imapStartTLSMissing  = "STARTTLS required but not supported by server"
	imapConnectFailed    = "Connect failed"
	imapConnectionLost   = "Lost connection to the IMAP server"
	imapReadOnly         = "Cannot open in desired mode"
	imapNoBody           = "The server returned no body for the message"
	// imapMaxLineLength bounds each line read before the session starts.
	imapMaxLineLength = 16 << 10
)

var errLineTooLong = errors.New("line too long")

// imapClient runs the IMAP sessions of an email:ImapClient.
type imapClient struct {
	*mailbox
}

func initImapClientExtern(rt *runtime.Runtime) extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		m, err := newMailbox(rt, args, imapPropertiesPrefix)
		if err != nil {
			return err, nil
		}
		// Like jBallerina, init connects so that bad hosts and credentials fail here.
		session, err := imapClient{m}.connect(rt.Platform())
		if err != nil {
			err.Message = imapConnectPrefix + err.Message
			return err, nil
		}
		logout(session)
		args[0].(*values.Object).Put(imapStateField, m)
		return nil, nil
	}
}

// imapReadExtern reads in one session, opened at the first check and kept selected while the
// read waits.
func imapReadExtern(rt *runtime.Runtime, strArrTy semtypes.SemType) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		client := imapClient{mailboxOf(args[0], imapStateField)}
		folder, _ := args[1].(string)
		timeout, _ := args[2].(*decimal.Decimal)
		if client.closed.Load() {
			return emailError(closedError), nil
		}
		platform := rt.Platform()
		var session *imapclient.Client
		// LOGOUT without CLOSE, which would expunge the messages other clients flagged \Deleted.
		defer func() {
			if session != nil {
				logout(session)
			}
		}()
		return client.receive(ctx, platform, strArrTy, timeout, func() ([]byte, func(bool), *values.Error) {
			if session == nil {
				opened, err := client.open(platform, folder)
				if err != nil {
					return nil, nil, err
				}
				session = opened
			}
			uid, data, err := firstUnseen(session)
			if err != nil || data == nil {
				return nil, nil, err
			}
			return data, func(received bool) {
				if received {
					// A failed STORE leaves the message unseen, so the next read returns it again.
					if err := session.Store(uid, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagSeen}}, nil).Close(); err != nil {
						client.kept.Store(true)
					}
				}
			}, nil
		})
	}
}

// open returns a session with folder selected for reading and writing.
func (c imapClient) open(platform pal.Platform, folder string) (*imapclient.Client, *values.Error) {
	session, err := c.connect(platform)
	if err != nil {
		return nil, err
	}
	selected, selectErr := session.Select(folder, nil).Wait()
	if selectErr != nil {
		defer logout(session)
		return nil, folderFailure(session, folder, selectErr)
	}
	if selected.ReadOnly {
		logout(session)
		return nil, emailError(imapReadOnly)
	}
	return session, nil
}

// firstUnseen returns the UID and content of the oldest message that is neither \Seen nor
// \Deleted, fetched without setting \Seen, or nil content when there is none.
func firstUnseen(session *imapclient.Client) (imap.UIDSet, []byte, *values.Error) {
	criteria := &imap.SearchCriteria{NotFlag: []imap.Flag{imap.FlagSeen, imap.FlagDeleted}}
	var options *imap.SearchOptions
	if caps := session.Caps(); caps.Has(imap.CapESearch) || caps.Has(imap.CapIMAP4rev2) {
		options = &imap.SearchOptions{ReturnMin: true}
	}
	search, err := session.UIDSearch(criteria, options).Wait()
	if err != nil {
		return nil, nil, imapFailure(err)
	}
	first := imap.UID(search.Min)
	if options == nil {
		if uids := search.AllUIDs(); len(uids) > 0 {
			first = slices.Min(uids)
		}
	}
	if first == 0 {
		return nil, nil, nil
	}
	uid := imap.UIDSetNum(first)
	section := &imap.FetchItemBodySection{Peek: true}
	messages, err := session.Fetch(uid, &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{section}}).Collect()
	if err != nil {
		return nil, nil, imapFailure(err)
	}
	if len(messages) == 0 {
		// Another client expunged the message after the search.
		return nil, nil, nil
	}
	data := messages[0].FindBodySection(section)
	if data == nil {
		return nil, nil, emailError(imapNoBody)
	}
	return uid, data, nil
}

// connect opens an authenticated session, with implicit TLS or STARTTLS as the security mode selects.
func (c imapClient) connect(platform pal.Platform) (*imapclient.Client, *values.Error) {
	conn, err := c.dial(platform)
	if err != nil {
		return nil, err
	}
	var greeting io.Reader
	if c.startTLS() {
		if conn, greeting, err = c.negotiateTLS(platform, conn); err != nil {
			return nil, err
		}
	}
	session := imapclient.New(replayConn{Conn: withTimeout(platform, conn), r: greeting}, nil)
	sessionErr := session.WaitGreeting()
	if sessionErr == nil {
		sessionErr = c.authenticate(session)
	}
	if sessionErr != nil {
		_ = session.Close()
		if _, rejected := errors.AsType[*imap.Error](sessionErr); rejected {
			return nil, imapFailure(sessionErr)
		}
		return nil, emailErrorWithCause(imapConnectFailed, sessionErr)
	}
	return session, nil
}

// authenticate uses AUTHENTICATE PLAIN when the server offers it and LOGIN otherwise, as
// JavaMail does.
func (c imapClient) authenticate(session *imapclient.Client) error {
	if session.Caps().Has(imap.AuthCap(sasl.Plain)) {
		return session.Authenticate(sasl.NewPlainClient("", c.username, c.password))
	}
	return session.Login(c.username, c.password).Wait()
}

// negotiateTLS runs STARTTLS when the server offers it and returns the connection for the
// session with the reader the session reads from: a greeting followed by the connection's data.
func (c imapClient) negotiateTLS(platform pal.Platform, conn net.Conn) (net.Conn, io.Reader, *values.Error) {
	// imapclient upgrades with crypto/tls directly, so STARTTLS runs here to upgrade through PAL.
	timed := withTimeout(platform, conn)
	r := bufio.NewReaderSize(timed, imapMaxLineLength)
	fail := func(err error) (net.Conn, io.Reader, *values.Error) {
		_ = conn.Close()
		return nil, nil, emailErrorWithCause(imapConnectFailed, err)
	}
	greeting, err := readLine(r)
	if err != nil {
		return fail(err)
	}
	capabilities, known := capabilityList(greeting)
	if !known {
		lines, err := command(timed, r, "S1", "CAPABILITY")
		if err != nil {
			return fail(err)
		}
		for _, line := range lines {
			if list, ok := capabilityList(line); ok {
				capabilities = list
			}
		}
		if fields := strings.Fields(greeting); len(fields) > 1 && strings.EqualFold(fields[1], "OK") {
			greeting = "* OK [CAPABILITY " + strings.Join(capabilities, " ") + "] ready"
		}
	}
	if !slices.ContainsFunc(capabilities, func(capability string) bool { return strings.EqualFold(capability, "STARTTLS") }) {
		if c.security == securityStartTLSAlways {
			_ = conn.Close()
			return nil, nil, emailError(imapStartTLSMissing)
		}
		return conn, io.MultiReader(strings.NewReader(greeting+"\r\n"), r), nil
	}
	lines, err := command(timed, r, "S2", "STARTTLS")
	if err != nil {
		return fail(err)
	}
	// Like JavaMail, a refused STARTTLS fails the connection in both STARTTLS modes.
	if fields := strings.Fields(lines[len(lines)-1]); len(fields) < 2 || !strings.EqualFold(fields[1], "OK") {
		_ = conn.Close()
		return nil, nil, emailError(strings.Join(fields[min(2, len(fields)):], " "))
	}
	tlsConn, tlsErr := c.upgradeTLS(platform, conn)
	if tlsErr != nil {
		return nil, nil, tlsErr
	}
	return tlsConn, io.MultiReader(strings.NewReader("* OK TLS negotiation completed\r\n"), withTimeout(platform, tlsConn)), nil
}

// command sends a tagged command and returns the response lines up to the tagged status line.
func command(w io.Writer, r *bufio.Reader, tag, name string) ([]string, error) {
	if _, err := fmt.Fprintf(w, "%s %s\r\n", tag, name); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
		if strings.HasPrefix(line, tag+" ") {
			return lines, nil
		}
	}
}

// readLine reads a line without its line break, failing for a line longer than r's buffer.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		return "", errLineTooLong
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(line), "\r\n"), nil
}

// capabilityList returns the capabilities in an untagged CAPABILITY response or in a CAPABILITY
// response code; IMAP keywords are case-insensitive.
func capabilityList(line string) ([]string, bool) {
	if _, code, found := strings.Cut(line, "["); found {
		code, _, _ = strings.Cut(code, "]")
		if fields := strings.Fields(code); len(fields) > 0 && strings.EqualFold(fields[0], "CAPABILITY") {
			return fields[1:], true
		}
	}
	if fields := strings.Fields(line); len(fields) > 1 && fields[0] == "*" && strings.EqualFold(fields[1], "CAPABILITY") {
		return fields[2:], true
	}
	return nil, false
}

// replayConn reads from r, when set, instead of the connection; r ends with the connection's data.
type replayConn struct {
	net.Conn
	r io.Reader
}

func (c replayConn) Read(p []byte) (int, error) {
	if c.r == nil {
		return c.Conn.Read(p)
	}
	return c.r.Read(p)
}

func logout(session *imapclient.Client) {
	_ = session.Logout().Wait()
	_ = session.Close()
}

// folderFailure reports a folder that does not exist as JavaMail does, and any other SELECT
// failure with the server's text.
func folderFailure(session *imapclient.Client, folder string, err error) *values.Error {
	list, listErr := session.List("", folder, nil).Collect()
	if listErr == nil && !slices.ContainsFunc(list, func(data *imap.ListData) bool {
		return data.Mailbox == folder || strings.EqualFold(folder, defaultFolder) && strings.EqualFold(data.Mailbox, folder)
	}) {
		return emailError(folder + " not found")
	}
	return imapFailure(err)
}

// imapFailure reports a negative response with its code and text, as JavaMail does, and any
// other failure as a lost connection with the error as its cause.
func imapFailure(err error) *values.Error {
	if response, ok := errors.AsType[*imap.Error](err); ok {
		if response.Code != "" {
			return emailError(fmt.Sprintf("[%s] %s", response.Code, response.Text))
		}
		return emailError(response.Text)
	}
	return emailErrorWithCause(imapConnectionLost, err)
}
