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

	"github.com/ballerina-nutcracker/ballerina/test_util"
)

const authLdapConfig = `
function ldapConfig(string url, string membershipAttribute = "member", string userSearchBase = "ou=Users,dc=example,dc=com",
        string password = %[2]q, decimal readTimeout = 60, auth:SecureSocket? secureSocket = ())
        returns auth:LdapUserStoreConfig {
    auth:LdapUserStoreConfig config = {
        domainName: "example.com",
        connectionUrl: url,
        connectionName: %[1]q,
        connectionPassword: password,
        userSearchBase: userSearchBase,
        userEntryObjectClass: "inetOrgPerson",
        userNameAttribute: "uid",
        userNameSearchFilter: "(&(objectClass=inetOrgPerson)(uid=?))",
        userNameListFilter: "(objectClass=inetOrgPerson)",
        groupSearchBase: [" ou=Groups,dc=example,dc=com "],
        groupEntryObjectClass: "groupOfNames",
        groupNameAttribute: "cn",
        groupNameSearchFilter: "(&(objectClass=groupOfNames)(cn=?))",
        groupNameListFilter: "(|(objectClass=groupOfNames)(objectClass=posixGroup))",
        membershipAttribute: membershipAttribute,
        readTimeout: readTimeout
    };
    if secureSocket is auth:SecureSocket {
        config.secureSocket = secureSocket;
    }
    return config;
}

function show(auth:UserDetails|auth:Error result) {
    if result is auth:Error {
        error? cause = result.cause();
        io:println(result.message(), " | ", cause is error ? cause.message() : "no cause");
    } else {
        string[]? scopes = result?.scopes;
        io:println(result.username, " ", scopes is () ? "no scopes" : scopes.toString());
    }
}

function credential(string usernameAndPassword) returns string => usernameAndPassword.toBytes().toBase64();
`

func seedAuthLdapDirectory(s *fakeLdapServer) {
	s.addEntry("uid=alice,ou=Users,dc=example,dc=com", map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {"alice"}, "userPassword": {"s3cret"}})
	s.addEntry(`uid=bob\\,ou=Users,dc=example,dc=com`, map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {`bob\`}, "userPassword": {"pw"}})
	s.addEntry("uid=carol,ou=Users,dc=example,dc=com", map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {"carol"}, "userPassword": {"pw"}})
	s.addEntry("cn=admin,ou=Groups,dc=example,dc=com", map[string][]string{"objectClass": {"groupOfNames"}, "cn": {"admin"}, "member": {"uid=alice,ou=Users,dc=example,dc=com"}})
	s.addEntry("cn=dev,ou=Groups,dc=example,dc=com", map[string][]string{"objectClass": {"groupOfNames"}, "cn": {"dev"}, "member": {"uid=alice,ou=Users,dc=example,dc=com", `uid=bob\\,ou=Users,dc=example,dc=com`}})
	s.addEntry("cn=staff,ou=Groups,dc=example,dc=com", map[string][]string{"objectClass": {"posixGroup"}, "cn": {"staff"}, "memberUid": {"carol"}})
	s.addEntry("uid=*,ou=Users,dc=example,dc=com", map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {"*"}, "userPassword": {"pw"}})
	s.addEntry("uid=st*r,ou=Users,dc=example,dc=com", map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {"st*r"}, "userPassword": {"pw"}})
	s.addEntry("cn=stars,ou=Groups,dc=example,dc=com", map[string][]string{"objectClass": {"groupOfNames"}, "cn": {"stars"}, "member": {"uid=st*r,ou=Users,dc=example,dc=com"}})
	s.addEntry(`uid=foo\\*,ou=Users,dc=example,dc=com`, map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {`foo\*`}, "userPassword": {"pw"}})
	s.addEntry(`cn=sp\ ,ou=Users,dc=example,dc=com`, map[string][]string{"objectClass": {"inetOrgPerson"}, "uid": {"sp"}, "userPassword": {"pw"}})
	s.addEntry("cn=escaped,ou=Groups,dc=example,dc=com", map[string][]string{"objectClass": {"groupOfNames"}, "cn": {"escaped"}, "member": {`uid=foo\\*,ou=Users,dc=example,dc=com`, `cn=sp\ ,ou=Users,dc=example,dc=com`}})
	s.addEntry("ou=Empty,dc=example,dc=com", map[string][]string{"objectClass": {"organizationalUnit"}, "ou": {"Empty"}})
}

func runAuthLdapCase(t *testing.T, name, balContent string) {
	t.Helper()
	tmpBalFile := filepath.Join(t.TempDir(), name+".bal")
	if err := os.WriteFile(tmpBalFile, []byte(balContent), 0o644); err != nil {
		t.Fatalf("writing bal file: %v", err)
	}
	tc := test_util.TestCase{
		Name:         name,
		InputPath:    tmpBalFile,
		ExpectedPath: filepath.Join(expectedDir, name+".txtar"),
	}
	runExtern(t, tc, newLdapPal(), nil)
}

func TestAuthLdapUserStore(t *testing.T) {
	t.Parallel()
	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, nil)
	defer server.close()
	seedAuthLdapDirectory(server)

	balContent := `
import ballerina/auth;
import ballerina/io;
` + fmt.Sprintf(authLdapConfig, fakeLdapBindDN, fakeLdapBindPW) + fmt.Sprintf(`
public function main() {
    string url = "ldap://%s";
    auth:ListenerLdapUserStoreBasicAuthProvider provider = new (ldapConfig(url));
    show(provider.authenticate(credential("alice:s3cret"))); // @output alice ["admin","dev"]
    show(provider.authenticate(credential("carol:pw"))); // @output carol no scopes
    show(provider.authenticate(credential("alice:wrong"))); // @output Failed to authenticate username 'alice' with LDAP user store. | [LDAP: error code 49 - Invalid Credentials]
    show(provider.authenticate(credential("dave:pw"))); // @output Failed to authenticate username 'dave' with LDAP user store. | Username cannot be found in the directory.
    show(provider.authenticate(credential("a*:pw"))); // @output Failed to authenticate username 'a*' with LDAP user store. | Username cannot be found in the directory.
    show(provider.authenticate("")); // @output Credential cannot be empty. | no cause
    show(provider.authenticate(credential("alice"))); // @output Incorrect credential format. Format should be username:password | no cause
    // jBallerina panics with StringIndexOutOfBoundsException for this DN: https://github.com/ballerina-platform/ballerina-library/issues/9708
    show(provider.authenticate(credential("bob\\:pw"))); // @output bob\ ["dev"]

    auth:ListenerBasicAuthProvider multiBase = new auth:ListenerLdapUserStoreBasicAuthProvider(
        ldapConfig(url, userSearchBase = "ou=Empty,dc=example,dc=com#ou=Users,dc=example,dc=com"));
    show(multiBase.authenticate(credential("alice:s3cret"))); // @output alice ["admin","dev"]

    auth:ListenerLdapUserStoreBasicAuthProvider posix = new (ldapConfig(url, membershipAttribute = "memberUid"));
    show(posix.authenticate(credential("carol:pw"))); // @output carol ["staff"]
    // jBallerina leaves both membership values unescaped, so * gets other users' groups and st*r none of its own: https://github.com/ballerina-platform/ballerina-library/issues/9717
    show(posix.authenticate(credential("*:pw"))); // @output * no scopes
    show(provider.authenticate(credential("st*r:pw"))); // @output st*r ["stars"]
    // jBallerina drops the backslash before the DN's star, so this user gets no groups: https://github.com/ballerina-platform/ballerina-library/issues/9726
    show(provider.authenticate(credential("foo\\\\*:pw"))); // @output foo\\* ["escaped"]
    show(provider.authenticate(credential("sp:pw"))); // @output sp ["escaped"]
    // jBallerina matches memberUid case-sensitively: https://github.com/ballerina-platform/ballerina-library/issues/9718
    auth:ListenerLdapUserStoreBasicAuthProvider posixLower = new (ldapConfig(url, membershipAttribute = "memberuid"));
    show(posixLower.authenticate(credential("carol:pw"))); // @output carol ["staff"]


    auth:ListenerLdapUserStoreBasicAuthProvider noMembership = new (ldapConfig(url, membershipAttribute = ""));
    show(noMembership.authenticate(credential("alice:s3cret"))); // @output Failed to get groups for the username 'alice' from LDAP user store. | Membership attribute is not set in configurations.

    auth:ListenerLdapUserStoreBasicAuthProvider|error wrongAdmin = trap new (ldapConfig(url, password = "wrong"));
    io:println(wrongAdmin is auth:Error ? wrongAdmin.message() : "connected"); // @output [LDAP: error code 49 - Invalid Credentials]
    auth:ListenerLdapUserStoreBasicAuthProvider|error notLdap = trap new (ldapConfig("http://localhost:389"));
    io:println(notLdap is auth:Error ? notLdap.message() : "connected"); // @output Not an LDAP URL: http://localhost:389
    auth:ListenerLdapUserStoreBasicAuthProvider|error trustStore = trap new (
        ldapConfig(url, secureSocket = {cert: {path: "truststore.p12", password: "ballerina"}}));
    io:println(trustStore is auth:Error ? trustStore.message() : "connected"); // @output crypto:TrustStore is not yet supported for secureSocket.cert; use a certificate file path instead
    auth:ListenerLdapUserStoreBasicAuthProvider|error missingCert = trap new (ldapConfig(url, secureSocket = {cert: "missing-ca.pem"}));
    io:println(missingCert is auth:Error); // @output true
}
`, server.addr())
	runAuthLdapCase(t, "auth-ldap-v", balContent)
}

// jBallerina truncates fractional timeouts, so 0.2s means no timeout there: https://github.com/ballerina-platform/ballerina-library/issues/9707
func TestAuthLdapUserStoreReadTimeout(t *testing.T) {
	t.Parallel()
	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, nil)
	defer server.close()
	seedAuthLdapDirectory(server)
	server.mu.Lock()
	server.stallSearch = true
	server.mu.Unlock()

	balContent := `
import ballerina/auth;
import ballerina/io;
` + fmt.Sprintf(authLdapConfig, fakeLdapBindDN, fakeLdapBindPW) + fmt.Sprintf(`
public function main() {
    auth:ListenerLdapUserStoreBasicAuthProvider provider = new (ldapConfig("ldap://%s", readTimeout = 0.2));
    auth:UserDetails|auth:Error result = provider.authenticate(credential("alice:s3cret"));
    io:println(result is auth:Error ? result.message() : "authenticated"); // @output Failed to authenticate username 'alice' with LDAP user store.
}
`, server.addr())
	runAuthLdapCase(t, "auth-ldap-timeout-v", balContent)
}

// jBallerina ignores secureSocket.cert for ldaps:// and fails the handshake: https://github.com/ballerina-platform/ballerina-library/issues/9706
func TestAuthLdapUserStoreTLS(t *testing.T) {
	t.Parallel()
	caCertPEM, serverCertPEM, serverKeyPEM, _, _ := generateTestCerts(t)
	serverTLSCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("creating server TLS cert: %v", err)
	}
	server := newFakeLdapServer(t, fakeLdapBindDN, fakeLdapBindPW, &tls.Config{Certificates: []tls.Certificate{serverTLSCert}})
	defer server.close()
	seedAuthLdapDirectory(server)

	caCertFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caCertFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("writing ca cert: %v", err)
	}
	_, port, err := net.SplitHostPort(server.addr())
	if err != nil {
		t.Fatalf("splitting server addr: %v", err)
	}

	balContent := `
import ballerina/auth;
import ballerina/io;
` + fmt.Sprintf(authLdapConfig, fakeLdapBindDN, fakeLdapBindPW) + fmt.Sprintf(`
public function main() {
    string url = "LDAPS://127.0.0.1:%s";
    auth:ListenerLdapUserStoreBasicAuthProvider provider = new (ldapConfig(url, secureSocket = {cert: %q}));
    show(provider.authenticate(credential("alice:s3cret"))); // @output alice ["admin","dev"]
    show(provider.authenticate(credential("alice:wrong"))); // @output Failed to authenticate username 'alice' with LDAP user store. | [LDAP: error code 49 - Invalid Credentials]

    auth:ListenerLdapUserStoreBasicAuthProvider|error untrusted = trap new (ldapConfig(url));
    io:println(untrusted is auth:Error); // @output true
}
`, port, caCertFile)
	runAuthLdapCase(t, "auth-ldap-tls-v", balContent)
}
