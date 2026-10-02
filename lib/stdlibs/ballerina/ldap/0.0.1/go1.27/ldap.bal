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


# Consists of APIs to integrate with LDAP.
public isolated client class Client {

    # Gets invoked to initialize the LDAP client.
    #
    # + config - The configurations to be used when initializing the client
    # + return - An `ldap:Error` if client initialization failed
    public isolated function init(ConnectionConfig config) returns Error? {
        return self.initLdapConnection(config);
    }

    private isolated function initLdapConnection(ConnectionConfig config) returns Error? = external;

    # Creates an entry in a directory server.
    #
    # ```ballerina
    # anydata user = {
    #   "objectClass": "user",
    #   "sn": "New User",
    #   "cn": "New User"
    # };
    # ldap:LdapResponse result = check ldapClient->add(userDN, user);
    # ```
    #
    # + dN - The distinguished name of the entry
    # + entry - The information to add
    # + return - A `ldap:Error` if the operation fails or `ldap:LdapResponse` if successfully created
    remote isolated function add(string dN, Entry entry) returns LdapResponse|Error = external;

    # Removes an entry in a directory server.
    #
    # ```ballerina
    # ldap:LdapResponse result = check ldapClient->delete(userDN);
    # ```
    #
    # + dN - The distinguished name of the entry to remove
    # + return - A `ldap:Error` if the operation fails or `ldap:LdapResponse` if successfully removed
    remote isolated function delete(string dN) returns LdapResponse|Error = external;

    # Updates information of an entry.
    #
    # ```ballerina
    # anydata user = {
    #   "sn": "User",
    #   "givenName": "Updated User",
    #   "displayName": "Updated User"
    # };
    # ldap:LdapResponse result = check ldapClient->modify(userDN, user);
    # ```
    #
    # + dN - The distinguished name of the entry
    # + entry - The information to update
    # + return - A `ldap:Error` if the operation fails or `LdapResponse` if successfully updated
    remote isolated function modify(string dN, Entry entry) returns LdapResponse|Error = external;

    # Renames an entry in a directory server.
    #
    # ```ballerina
    # ldap:LdapResponse modifyDN = check ldapClient->modifyDn(userDN, "CN=Test User2", true);
    # ```
    #
    # + currentDn - The current distinguished name of the entry
    # + newRdn - The new relative distinguished name
    # + deleteOldRdn - A boolean value to determine whether to delete the old RDN
    # + return - A `ldap:Error` if the operation fails or `ldap:LdapResponse` if successfully renamed
    remote isolated function modifyDn(string currentDn, string newRdn, boolean deleteOldRdn = false)
        returns LdapResponse|Error = external;

    # Determines whether a given entry has a specified attribute value.
    #
    # ```ballerina
    # boolean compare = check ldapClient->compare(userDN, "givenName", "New User");
    # ```
    #
    # + dN - The distinguished name of the entry
    # + attributeName - The name of the target attribute for which the comparison is to be performed
    # + assertionValue - The assertion value to verify within the entry
    # + return - A `boolean` value indicating whether the values match, or an `ldap:Error` if the operation fails
    remote isolated function compare(string dN, string attributeName, string assertionValue)
        returns boolean|Error = external;

    # Gets information of an entry.
    #
    # ```ballerina
    # anydata value = check ldapClient->getEntry(userDN);
    # ```
    #
    # + dN - The distinguished name of the entry
    # + attributes - Optional array of attribute names to retrieve. If not provided, attributes are
    #                determined based on the target type
    # + targetType - Default parameter use to infer the user specified type
    # + return - An entry result with the given type or else `ldap:Error`
    remote isolated function getEntry(string dN, string[]? attributes = (), typedesc<anydata> targetType = <>)
        returns targetType|Error = external;

    # Returns a list of entries that match the given search parameters.
    #
    # ```ballerina
    # anydata[] value = check ldapClient->searchWithType("DC=ldap,DC=com", "(givenName=New User)", ldap:SUB);
    # ```
    #
    # + baseDn - The base distinguished name of the entry
    # + filter - The filter to be used in the search
    # + scope - The scope of the search
    # + attributes - Optional array of attribute names to retrieve. If not provided, attributes are
    #                determined based on the target type
    # + targetType - Default parameter use to infer the user specified type
    # + return - An array of entries with the given type or else `ldap:Error`
    remote isolated function searchWithType(string baseDn, string filter,
            SearchScope scope, string[]? attributes = (), typedesc<record {}[]> targetType = <>)
        returns targetType|Error = external;

    # Returns a record containing search result entries and references that match the given search parameters.
    #
    # ```ballerina
    # ldap:SearchResult value = check ldapClient->search("DC=ldap,DC=windows", "(givenName=New User)", ldap:SUB);
    # ```
    #
    # + baseDn - The base distinguished name of the entry
    # + filter - The filter to be used in the search
    # + scope - The scope of the search
    # + attributes - Optional array of attribute names to retrieve. If not provided, all attributes are retrieved
    # + return - An `ldap:SearchResult` if successful, or else `ldap:Error`
    remote isolated function search(string baseDn, string filter, SearchScope scope, string[]? attributes = ())
        returns SearchResult|Error = external;

    # Unbinds from the server and closes the LDAP connection.
    #
    # ```ballerina
    # ldapClient->close();
    # ```
    remote isolated function close() = external;

    # Determines whether the client is connected to the server.
    #
    # ```ballerina
    # boolean isConnected = ldapClient->isConnected();
    # ```
    #
    # + return - A boolean value indicating the connection status
    remote isolated function isConnected() returns boolean = external;
}



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


// Represents any error related to the Ballerina LDAP module.
// Note: distinct error types are not yet supported; Error is currently an alias for error.
public type Error error;

// The error details type for the Ballerina LDAP module.
//
// Fields:
//   resultCode - The status of the error. Not currently populated: this interpreter's lang.error
//                does not implement error:detail() yet, so the LDAP result code is only available
//                via error:message() text.
public type ErrorDetails record {|
    string resultCode?;
|};
