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

package extern_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/platform/palnative"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/test_util/testharness"
)

// ldapPal wraps the default in-memory TestPal but overrides Net.Dial with the
// real platform dialer, so the ldap:Client under test performs actual TCP(+TLS)
// I/O against the in-process fakeLdapServer started per test.
type ldapPal struct {
	testharness.TestPal
}

func newLdapPal() *ldapPal {
	return &ldapPal{TestPal: testharness.NewTestPal()}
}

func (p *ldapPal) Platform() pal.Platform {
	base := p.TestPal.Platform()
	base.Net = pal.Net{Dial: palnative.Dial}
	return base
}

const (
	fakeLdapBindDN = "cn=admin,dc=example,dc=com"
	fakeLdapBindPW = "adminpassword"
)

func TestLdapClientBasic(t *testing.T) {
	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, nil)
	defer server.close()

	host, port, err := net.SplitHostPort(server.addr())
	if err != nil {
		t.Fatalf("splitting server addr: %v", err)
	}

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/ldap;

type Person record {|
    string sn?;
    string cn?;
    string telephoneNumber?;
|};

public function main() returns error? {
    ldap:Client ldapClient = check new ({
        hostName: %q,
        port: %s,
        domainName: %q,
        password: %q
    });
    boolean connected = check ldapClient->isConnected();
    io:println(connected); // @output true

    string userDN = "cn=John Doe,dc=example,dc=com";
    ldap:LdapResponse addResult = check ldapClient->add(userDN, {
        "objectClass": ["top", "person"],
        "sn": "Doe",
        "cn": "John Doe",
        "telephoneNumber": "1234567890"
    });
    io:println(addResult.resultCode); // @output SUCCESS

    Person entry = check ldapClient->getEntry(userDN);
    io:println(entry.sn); // @output Doe
    io:println(entry.cn); // @output John Doe

    ldap:LdapResponse modifyResult = check ldapClient->modify(userDN, {"telephoneNumber": "5551234"});
    io:println(modifyResult.resultCode); // @output SUCCESS

    boolean compareMatch = check ldapClient->compare(userDN, "telephoneNumber", "5551234");
    io:println(compareMatch); // @output true
    boolean compareNoMatch = check ldapClient->compare(userDN, "telephoneNumber", "0000000");
    io:println(compareNoMatch); // @output false

    ldap:SearchResult searchResult = check ldapClient->search("dc=example,dc=com", "(cn=John Doe)", ldap:SUB);
    io:println(searchResult.resultCode); // @output SUCCESS
    ldap:Entry[]? entries = searchResult?.entries;
    io:println(entries is ldap:Entry[] && entries.length() == 1); // @output true

    Person[] typedResults = check ldapClient->searchWithType("dc=example,dc=com", "(sn=Doe)", ldap:SUB);
    io:println(typedResults.length()); // @output 1
    io:println(typedResults[0].cn); // @output John Doe

    string newUserDN = "cn=Jane Doe,dc=example,dc=com";
    ldap:LdapResponse renameResult = check ldapClient->modifyDn(userDN, "cn=Jane Doe", false);
    io:println(renameResult.resultCode); // @output SUCCESS

    ldap:LdapResponse deleteResult = check ldapClient->delete(newUserDN);
    io:println(deleteResult.resultCode); // @output SUCCESS

    Person|ldap:Error afterDelete = ldapClient->getEntry(newUserDN);
    io:println(afterDelete is ldap:Error); // @output true

    ldapClient->close();
    boolean stillConnected = check ldapClient->isConnected();
    io:println(stillConnected); // @output false
    return;
}
`, host, port, fakeLdapBindDN, fakeLdapBindPW)

	tmpDir := t.TempDir()
	tmpBalFile := filepath.Join(tmpDir, "ldap-client-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "ldap-client-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "ldap-client-v.txtar"),
	}
	runExtern(t, tc, newLdapPal(), nil)
}

func TestLdapClientTLS(t *testing.T) {
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)

	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}

	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
	})
	defer server.close()

	host, port, err := net.SplitHostPort(server.addr())
	if err != nil {
		t.Fatalf("splitting server addr: %v", err)
	}

	tmpDir := t.TempDir()
	caCertFile := filepath.Join(tmpDir, "ca.pem")
	if err := os.WriteFile(caCertFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("writing ca cert: %v", err)
	}

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/ldap;

public function main() returns error? {
    ldap:Client ldapClient = check new ({
        hostName: %q,
        port: %s,
        domainName: %q,
        password: %q,
        clientSecureSocket: {
            cert: %q,
            verifyHostName: false
        }
    });
    boolean connected = check ldapClient->isConnected();
    io:println(connected); // @output true

    string userDN = "cn=TLS User,dc=example,dc=com";
    ldap:LdapResponse addResult = check ldapClient->add(userDN, {
        "objectClass": ["top", "person"],
        "sn": "User",
        "cn": "TLS User"
    });
    io:println(addResult.resultCode); // @output SUCCESS

    ldap:SearchResult searchResult = check ldapClient->search("dc=example,dc=com", "(cn=TLS User)", ldap:SUB);
    io:println(searchResult.resultCode); // @output SUCCESS

    ldapClient->close();
    return;
}
`, host, port, fakeLdapBindDN, fakeLdapBindPW, filepath.ToSlash(caCertFile))

	tmpBalFile := filepath.Join(tmpDir, "ldap-client-tls-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "ldap-client-tls-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "ldap-client-tls-v.txtar"),
	}
	runExtern(t, tc, newLdapPal(), nil)
}
