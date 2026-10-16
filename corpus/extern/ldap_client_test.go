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

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
	"github.com/ballerina-nutcracker/ballerina/test_util"
	"github.com/ballerina-nutcracker/ballerina/test_util/testharness"
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
	t.Parallel()
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

type IntSurname record {|
    int sn;
|};

type RequiresMail record {|
    string sn;
    string mail;
|};

type StringUnions record {|
    string|boolean flag;
    anydata a;
    int|string code;
|};

type ObjectClasses record {|
    string[] objectClass;
    string[] sn;
|};

type NameOnly record {|
    string cn;
    string sn;
|};

type DomainOnly record {|
    string dc;
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
    io:println(addResult); // @output {"matchedDN":null,"resultCode":"SUCCESS","diagnosticMessage":null,"operationType":"ADD","referral":[]}

    Person entry = check ldapClient->getEntry(userDN);
    io:println(entry.sn); // @output Doe
    io:println(entry.cn); // @output John Doe

    ldap:Entry genericEntry = check ldapClient->getEntry(userDN);
    io:println(genericEntry); // @output {"cn":"John Doe","objectclass":["top","person"],"sn":"Doe","telephonenumber":"1234567890"}
    genericEntry["mail"] = ["john@example.com", "doe@example.com"];
    io:println(genericEntry["mail"]); // @output ["john@example.com","doe@example.com"]

    ObjectClasses classes = check ldapClient->getEntry(userDN);
    io:println(classes); // @output {"objectClass":["top","person"],"sn":["Doe"]}

    IntSurname|ldap:Error intSurname = ldapClient->getEntry(userDN);
    if intSurname is ldap:Error {
        io:println(intSurname.message()); // @output field 'sn' should be of type 'int', found 'Doe'
    }

    RequiresMail|ldap:Error requiresMail = ldapClient->getEntry(userDN);
    if requiresMail is ldap:Error {
        io:println(requiresMail.message()); // @output missing required field 'mail' of type 'string'
    }

    string unionsDN = "cn=Unions,dc=example,dc=com";
    _ = check ldapClient->add(unionsDN, {"objectClass": "top", "cn": "Unions", "flag": "1", "a": "t", "code": "007"});
    StringUnions unions = check ldapClient->getEntry(unionsDN);
    io:println(unions); // @output {"a":"t","code":"007","flag":"1"}
    _ = check ldapClient->delete(unionsDN);

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

    ldap:Entry[] genericResults = check ldapClient->searchWithType("dc=example,dc=com", "(sn=Doe)", ldap:SUB);
    io:println(genericResults); // @output [{"cn":"John Doe","objectclass":["top","person"],"sn":"Doe","telephonenumber":"5551234"}]

    string newUserDN = "cn=Jane Doe,dc=example,dc=com";
    ldap:LdapResponse renameResult = check ldapClient->modifyDn(userDN, "cn=Jane Doe", false);
    io:println(renameResult.resultCode); // @output SUCCESS

    ldap:LdapResponse deleteResult = check ldapClient->delete(newUserDN);
    io:println(deleteResult.resultCode); // @output SUCCESS

    Person|ldap:Error afterDelete = ldapClient->getEntry(newUserDN);
    io:println(afterDelete is ldap:Error); // @output true

    ldap:SearchResult|ldap:Error noMatch = ldapClient->search("dc=example,dc=com", "(cn=Nobody)", ldap:SUB);
    if noMatch is ldap:Error {
        io:println(noMatch.message()); // @output Entry is not found for DN: 'dc=example,dc=com'
    }

    string encodedDN = "cn=Encoded,dc=example,dc=com";
    _ = check ldapClient->add(encodedDN, {
        "objectClass": "top",
        "cn": "Encoded",
        "sn": "S",
        "description": "tab\there",
        "telephoneNumber": "del\u{7F}char",
        "objectSid": "\u{01}\u{01}\u{01}\u{00}\u{00}\u{00}\u{00}\u{05}\u{07}\u{00}\u{00}\u{00}"
    });
    ldap:Entry encoded = check ldapClient->getEntry(encodedDN, ["description", "telephoneNumber", "objectSid"]);
    io:println(encoded); // @output {"description":"dGFiCWhlcmU=","objectsid":"S-1-0x10000000005-7","telephonenumber":"ZGVsf2NoYXI="}

    NameOnly|DomainOnly|ldap:Error unionTarget = ldapClient->getEntry(encodedDN, ["cn", "sn"]);
    io:println(unionTarget is NameOnly); // @output true
    readonly & NameOnly|ldap:Error readonlyTarget = ldapClient->getEntry(encodedDN, ["cn", "sn"]);
    io:println(readonlyTarget is readonly & NameOnly); // @output true
    string|ldap:Error stringTarget = ldapClient->getEntry(encodedDN);
    io:println(stringTarget is ldap:Error); // @output true
    _ = check ldapClient->delete(encodedDN);

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
	t.Parallel()
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

func TestLdapClientTLSConfig(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	untrustedCACertPEM, _, _, _, _ := generateTestCerts(t)

	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}

	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, &tls.Config{
		Certificates: []tls.Certificate{serverTLSCert},
	})
	defer server.close()

	_, port, err := net.SplitHostPort(server.addr())
	if err != nil {
		t.Fatalf("splitting server addr: %v", err)
	}

	tmpDir := t.TempDir()
	caCertFile := filepath.Join(tmpDir, "ca.pem")
	if err := os.WriteFile(caCertFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("writing ca cert: %v", err)
	}
	untrustedCACertFile := filepath.Join(tmpDir, "untrusted-ca.pem")
	if err := os.WriteFile(untrustedCACertFile, untrustedCACertPEM, 0o600); err != nil {
		t.Fatalf("writing untrusted ca cert: %v", err)
	}

	balContent := fmt.Sprintf(`
import ballerina/io;
import ballerina/ldap;

const PORT = %s;
const BIND_DN = %q;
const BIND_PW = %q;
const CA_CERT = %q;
const UNTRUSTED_CA_CERT = %q;

function connect(string hostName, ldap:ClientSecureSocket secureSocket) returns ldap:Client|ldap:Error {
    return new ({
        hostName,
        port: PORT,
        domainName: BIND_DN,
        password: BIND_PW,
        clientSecureSocket: secureSocket
    });
}

public function main() returns error? {
    // The server certificate is issued for 127.0.0.1 only.
    ldap:Client|ldap:Error hostMismatch = connect("localhost", {cert: CA_CERT});
    io:println(hostMismatch is ldap:Error); // @output true

    ldap:Client hostCheckSkipped = check connect("localhost", {cert: CA_CERT, verifyHostName: false});
    boolean hostCheckSkippedConnected = check hostCheckSkipped->isConnected();
    io:println(hostCheckSkippedConnected); // @output true
    hostCheckSkipped->close();

    ldap:Client|ldap:Error untrustedChain = connect("127.0.0.1", {cert: UNTRUSTED_CA_CERT, verifyHostName: false});
    io:println(untrustedChain is ldap:Error); // @output true

    ldap:Client enableFalse = check connect("127.0.0.1", {enable: false, cert: CA_CERT});
    boolean enableFalseConnected = check enableFalse->isConnected();
    io:println(enableFalseConnected); // @output true
    enableFalse->close();

    ldap:Client tlsV1Name = check connect("127.0.0.1", {cert: CA_CERT, tlsVersions: ["TLSv1", "TLSv1.2"]});
    boolean tlsV1NameConnected = check tlsV1Name->isConnected();
    io:println(tlsV1NameConnected); // @output true
    tlsV1Name->close();
    return;
}
`, port, fakeLdapBindDN, fakeLdapBindPW, filepath.ToSlash(caCertFile), filepath.ToSlash(untrustedCACertFile))

	tmpBalFile := filepath.Join(tmpDir, "ldap-client-tls-config-v.bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}

	tc := test_util.TestCase{
		Name:         "ldap-client-tls-config-v",
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, "ldap-client-tls-config-v.txtar"),
	}
	runExtern(t, tc, newLdapPal(), nil)
}
