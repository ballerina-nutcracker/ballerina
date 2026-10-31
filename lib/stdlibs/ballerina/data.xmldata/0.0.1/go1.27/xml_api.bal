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

const EMPTY_STRING = "";

# Defines the name of the XML element.
public type NameConfig record {|
    # The name of the XML element
    string value;
|};

# The annotation is used to specify the new name of the existing record name or field name according to the XML format.
# When using `parseString`, `parseBytes`, `parseStream`, `parseAsType`, this annotation can be used to
# validate the name of the XML element with the record field or type.
# When using `toXml`, this annotation can be used to override the name of field or type.
public const annotation NameConfig Name on type, record field;

# Defines the namespace of the XML element.
public type NamespaceConfig record {|
    # The value of the prefix of the namespace
    string prefix?;
    # The value of the URI of the namespace
    string uri;
|};

# The annotation is used to specify the namespace's prefix and URI of the XML element.
# When using `parseString`, `parseBytes`, `parseStream`, `parseAsType`, this annotation can be used to
# validate the namespace of the XML element with the record field or type.
# When using `toXml`, this annotation can be used to add the namespace to the XML element.
public const annotation NamespaceConfig Namespace on type, record field;

# The annotation is used to denote the field that is considered an attribute.
# When using `parseString`, `parseBytes`, `parseStream`, `parseAsType`, this annotation can be used to
# indicate the record field as an attribute.
# When using `toXml`, this annotation can be used to add the attribute to the XML element.
public const annotation Attribute on record field;

# Represent the options that can be used to modify the behaviour of projection.
public type Options record {|
    # prefix added for attribute fields in the record
    string attributePrefix = EMPTY_STRING;
    # field name for the text field
    string textFieldName = "#content";
|};

# Represent the options that can be used to modify the behaviour of projection.
public type SourceOptions record {|
    *Options;
    # enable or disable projection
    boolean allowDataProjection = true;
    # choose between semantic and syntactic equality
    boolean useSemanticEquality = true;
    # enable or disable constraint validation
    boolean enableConstraintValidation = true;
|};

# Represents the error type of the ballerina/data.xmldata module. This error type represents any error that can occur
# during the execution of xmldata APIs.
public type Error distinct error;

isolated function createError(string message) returns Error {
    return error Error(message);
}

# Converts XML to record type with projection.
#
# + v - Source XML value
# + options - Options to be used for filtering in the projection
# + t - Target type to be used for filtering in the projection
# + return - On success, returns the given target type value, else returns an `xmldata:Error` value
public isolated function parseAsType(xml v, SourceOptions options = {}, typedesc<record {}> t = <>)
        returns t|Error = external;

# Converts XML string to record type with projection.
#
# + s - Source XML string value
# + options - Options to be used for filtering in the projection
# + t - Target type to be used for filtering in the projection
# + return - On success, returns the given target type value, else returns an `xmldata:Error` value
public isolated function parseString(string s, SourceOptions options = {}, typedesc<record {}> t = <>)
        returns t|Error = external;

# Converts XML byte[] to record type with projection.
#
# + s - Source XML  byte[]
# + options - Options to be used for filtering in the projection
# + t - Target type to be used for filtering in the projection
# + return - On success, returns the given target type value, else returns an `xmldata:Error` value
public isolated function parseBytes(byte[] s, SourceOptions options = {}, typedesc<record {}> t = <>)
        returns t|Error = external;

# Converts XML byte-block-stream to record type with projection.
#
# + s - Source XML byte-block-stream
# + options - Options to be used for filtering in the projection
# + t - Target type to be used for filtering in the projection
# + return - On success, returns the given target type value, else returns an `xmldata:Error` value
public isolated function parseStream(stream<byte[], error?> s, SourceOptions options = {}, typedesc<record {}> t = <>)
        returns t|Error = external;

# Validates an XML document against an XML schema.
#
# The schema can either be a file path to the `.xsd` file or a Ballerina record type that represents
# the schema defined by the XSD. The function checks if the XML value conforms to the specified schema.
#
# + schema - A string representing the file path to the `.xsd` file or a
#            Ballerina record type representing the schema defined by the XSD.
# + xmlValue - The XML document that needs to be validated against the schema.
# + return - Returns `()` if the XML value is valid according to the schema, otherwise returns `Error`.
public isolated function validate(xml xmlValue, string|typedesc<record {}> schema)
    returns Error? = external;
