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

# Consists of APIs to integrate with LDAP.
public isolated client class Client {

    # Gets invoked to initialize the LDAP client.
    #
    # + config - The configurations to be used when initializing the client
    # + return - An `ldap:Error` if client initialization failed
    public isolated function init(*ConnectionConfig config) returns Error? {
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
