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

import ballerina/crypto;

// Provides a set of configurations to connect with a directory server.
//
// Fields:
//   hostName          - The host name of the LDAP server.
//   port              - The port of the LDAP server.
//   domainName        - The bind DN used for the simple bind. Despite the name, this is not an
//                        Active Directory DOMAIN\user string — it is passed verbatim as the bind DN.
//   password          - The password of the LDAP server.
//   clientSecureSocket - Client secure socket configurations.
public type ConnectionConfig record {|
    string hostName;
    int port;
    string domainName;
    string password;
    ClientSecureSocket clientSecureSocket?;
|};

// Provides configurations for facilitating secure communication with a remote LDAP server.
//
// Not supported: crypto:TrustStore (PKCS12) as `cert` — a PEM certificate file path string is
// fully supported instead; supplying a crypto:TrustStore value returns an Error.
//
// Fields:
//   enable         - Enable SSL validation.
//   cert           - A PEM certificate file path that the client trusts (crypto:TrustStore not supported).
//   verifyHostName - Enable/disable host name verification.
//   tlsVersions    - The TLS versions to be used.
public type ClientSecureSocket record {|
    boolean enable = true;
    crypto:TrustStore|string cert?;
    boolean verifyHostName = true;
    string[] tlsVersions = [];
|};

// LDAP response type.
//
// Fields:
//   matchedDN         - The matched DN from the response.
//   resultCode        - The operation status of the response.
//   diagnosticMessage - The diagnostic message from the response.
//   operationType     - The protocol operation type.
//   referral          - The referral URIs.
public type LdapResponse record {|
    string? matchedDN;
    Status resultCode;
    string? diagnosticMessage;
    string? operationType;
    string[]? referral;
|};

// LDAP search result type.
//
// Fields:
//   resultCode       - The result status of the response.
//   searchReferences - Search references.
//   entries          - The entries returned from the search.
public type SearchResult record {|
    Status resultCode;
    SearchReference[] searchReferences?;
    Entry[] entries?;
|};

// LDAP search reference type.
//
// Fields:
//   messageId - The message ID.
//   uris      - The referral URIs.
//   controls  - The controls.
public type SearchReference record {|
    int messageId;
    string[] uris;
    Control[] controls;
|};

// LDAP control type.
//
// Fields:
//   oid        - The OID of the control.
//   isCritical - The criticality of the control.
//   value      - The value of the control.
public type Control record {|
    string oid;
    boolean isCritical;
    string value;
|};

// Scope of the search operation.
public enum SearchScope {
    // Indicates that only the entry specified by the base DN should be considered.
    BASE,
    // Indicates that only entries that are immediate subordinates of the entry specified by the base
    // DN (but not the base entry itself) should be considered.
    ONE,
    // Indicates that the base entry itself and any subordinate entries (to any depth) should be considered.
    SUB,
    // Indicates that any subordinate entries (to any depth) below the entry specified by the base DN
    // should be considered, but the base entry itself should not be considered.
    SUBORDINATE_SUBTREE
};

// Attribute type of an LDAP entry.
public type AttributeType boolean|int|float|decimal|string|string[];

// LDAP entry type.
public type Entry record {|
    AttributeType...;
|};

// A record for an entry that represents a person.
//
// Fields:
//   objectClass     - object class of the person.
//   sn              - surname of the person.
//   cn              - common name of the person.
//   userPassword    - password of the person.
//   telephoneNumber - telephone number of the person.
public type Person record {
    string|string[]|ObjectClass|ObjectClass[] objectClass?;
    string sn?;
    string cn?;
    string userPassword?;
    string telephoneNumber?;
};

// Standard values for ObjectClass attribute type.
public enum ObjectClass {
    top,
    person,
    organizationalPerson,
    inetOrgPerson,
    organizationalRole,
    groupOfNames,
    groupOfUniqueNames,
    country,
    locality,
    organization,
    organizationalUnit,
    domainComponent,
    dcObject
};

// A record for an entry to contain domain component information.
//
// Fields:
//   dc - name of the domain component.
public type DcObject record {
    string dc;
};

// Represents the status of the operation.
//
// OTHER also covers client-side/connection-level failures that have no dedicated LDAP result code
// (e.g. connection closed, connect timeout) — see the README's Notable Behavioural Changes.
public enum Status {
    SUCCESS,
    OPERATIONS_ERROR = "OPERATIONS ERROR",
    PROTOCOL_ERROR = "PROTOCOL ERROR",
    TIME_LIMIT_EXCEEDED = "TIME LIMIT EXCEEDED",
    SIZE_LIMIT_EXCEEDED = "SIZE LIMIT EXCEEDED",
    COMPARE_FALSE = "COMPARE FALSE",
    COMPARE_TRUE = "COMPARE TRUE",
    AUTH_METHOD_NOT_SUPPORTED = "AUTH METHOD NOT SUPPORTED",
    STRONGER_AUTH_REQUIRED = "STRONGER AUTH REQUIRED",
    REFERRAL,
    ADMIN_LIMIT_EXCEEDED = "ADMIN LIMIT EXCEEDED",
    UNAVAILABLE_CRITICAL_EXTENSION = "UNAVAILABLE CRITICAL EXTENSION",
    CONFIDENTIALITY_REQUIRED = "CONFIDENTIALITY REQUIRED",
    SASL_BIND_IN_PROGRESS = "SASL BIND IN PROGRESS",
    NO_SUCH_ATTRIBUTE = "NO SUCH ATTRIBUTE",
    UNDEFINED_ATTRIBUTE_TYPE = "UNDEFINED ATTRIBUTE TYPE",
    INAPPROPRIATE_MATCHING = "INAPPROPRIATE MATCHING",
    CONSTRAINT_VIOLATION = "CONSTRAINT VIOLATION",
    ATTRIBUTE_OR_VALUE_EXISTS = "ATTRIBUTE OR VALUE EXISTS",
    INVALID_ATTRIBUTE_SYNTAX = "INVALID ATTRIBUTE SYNTAX",
    NO_SUCH_OBJECT = "NO SUCH OBJECT",
    ALIAS_PROBLEM = "ALIAS PROBLEM",
    INVALID_DN_SYNTAX = "INVALID DN SYNTAX",
    ALIAS_DEREFERENCING_PROBLEM = "ALIAS DEREFERENCING PROBLEM",
    INAPPROPRIATE_AUTHENTICATION = "INAPPROPRIATE AUTHENTICATION",
    INVALID_CREDENTIALS = "INVALID CREDENTIALS",
    INSUFFICIENT_ACCESS_RIGHTS = "INSUFFICIENT ACCESS RIGHTS",
    BUSY,
    UNAVAILABLE,
    UNWILLING_TO_PERFORM = "UNWILLING TO PERFORM",
    LOOP_DETECT = "LOOP DETECT",
    NAMING_VIOLATION = "NAMING VIOLATION",
    OBJECT_CLASS_VIOLATION = "OBJECT CLASS VIOLATION",
    NOT_ALLOWED_ON_NON_LEAF = "NOT ALLOWED ON NON LEAF",
    NOT_ALLOWED_ON_RDN = "NOT ALLOWED ON RDN",
    ENTRY_ALREADY_EXISTS = "ENTRY ALREADY EXISTS",
    OBJECT_CLASS_MODS_PROHIBITED = "OBJECT CLASS MODS PROHIBITED",
    AFFECTS_MULTIPLE_DSAS = "AFFECTS MULTIPLE DSAS",
    OTHER
}
