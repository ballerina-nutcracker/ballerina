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

const NEW_LINE_CHARACTER = "\n";

# Converts YAML string to subtype of anydata.
#
# ```ballerina
# json value = yaml:parseString("name: Ballerina");
# value ⇒ {"name": "Ballerina"}
# ```
# 
# + s - Source string value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns the given target type value, else returns an `yaml:Error`
public isolated function parseString(string s,
        Options options = {}, typedesc<anydata> t = <>)
    returns t|Error = external;

# Converts YAML byte[] to subtype of anydata.
#
# ```ballerina
# byte[] content = "name: Ballerina".toBytes();
# json value = yaml:parseBytes(content);
# value ⇒ {"name": "Ballerina"}
# ```
# 
# + s - Source byte[] value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns the given target type value, else returns an `yaml:Error`
public isolated function parseBytes(byte[] s,
        Options options = {}, typedesc<anydata> t = <>)
    returns t|Error = external;

# Converts YAML byte-block-stream to subtype of anydata.
#
# ```ballerina
# stream<byte[], error?> content = getStream();
# json value = yaml:parseStream(content);
# value ⇒ {"name": "Ballerina"}
# ```
# 
# + s - Source byte-block-stream value
# + options - Options to be used for filtering in the projection
# + t - Target type
# + return - On success, returns the given target type value, else returns an `yaml:Error`
public isolated function parseStream(stream<byte[], error?> s,
        Options options = {}, typedesc<anydata> t = <>)
    returns t|Error = external;

# Converts anydata YAML value to a string.
#
# ```ballerina
# yaml:toYamlString({"name": "Ballerina"}) ⇒ "name: Ballerina"
# ```
# 
# + yamlValue - Input yaml value
# + config - Options used to get desired toString representation
# + return - On success, returns to string value, else returns an `yaml:Error`
public isolated function toYamlString(anydata yamlValue, WriteConfig config = {}) returns string|Error {
    string[] lines = check toYamlStringArray(yamlValue, config);
    string result = "";
    foreach int i in 0 ..< lines.length() {
        if i > 0 {
            result += NEW_LINE_CHARACTER;
        }
        result += lines[i];
    }
    return result;
}

isolated function toYamlStringArray(anydata yamlValue, WriteConfig config = {})
    returns string[]|Error = external;

# Represents the YAML schema available for the parser.
#
# + FAILSAFE_SCHEMA - Generic schema that works for any YAML document
# + JSON_SCHEMA - Schema supports all the basic JSON types
# + CORE_SCHEMA - An extension of JSON schema that allows more human-readable presentation
public enum YAMLSchema {
    FAILSAFE_SCHEMA,
    JSON_SCHEMA,
    CORE_SCHEMA
}

# Represent the options that can be used for filtering in the projection.
public type Options record {|
    # schema - field description
    YAMLSchema schema = CORE_SCHEMA;
    # allowAnchorRedefinition - field description
    boolean allowAnchorRedefinition = true;
    # allowMapEntryRedefinition - field description
    boolean allowMapEntryRedefinition = false;
    # allowDataProjection - Enable or disable projection
    record {
        # If `true`, nil values will be considered as optional fields in the projection.
        boolean nilAsOptionalField = false;
        # If `true`, absent fields will be considered as nilable types in the projection.
        boolean absentAsNilableType = false;
        # If `true`, yaml document to top level tuple value will re-ordered.
        boolean enableYamlStreamReorder = false;
    }|false allowDataProjection = {};
    # enableConstraintValidation - Enable or disable constraint validation
    boolean enableConstraintValidation = true;
|};

# Configurations for writing a YAML document.
public type WriteConfig record {|
    # indentationPolicy - Number of whitespace for an indentation
    int indentationPolicy = 2;
    # blockLevel - The maximum depth level for a block collection
    int blockLevel = 1;
    # canonical - If set, the tags are written along with the nodes
    boolean canonical = false;
    # useSingleQuotes - If set, single quotes are used to surround scalars
    boolean useSingleQuotes = false;
    # forceQuotes - If set, all the scalars are surrounded by quotes
    boolean forceQuotes = false;
    # schema - YAML schema used for writing
    YAMLSchema schema = CORE_SCHEMA;
    # isStream - If set, the parser will write a stream of YAML documents
    boolean isStream = false;
    # flowStyle - If set, mappings and sequences will output in flow style
    boolean flowStyle = false;
|};

# Represents the error type of the ballerina/data.yaml module. This error type represents any error that can occur
# during the execution of data.yaml APIs.
public type Error distinct error;

# Defines the name of the JSON Object key.
public type NameConfig record {|
    # value - The name of the JSON Object key
    string value;
|};

# The annotation is used to overwrite the existing record field name.
public const annotation NameConfig Name on record field;

isolated function createError(string message) returns Error {
    return error Error(message);
}
