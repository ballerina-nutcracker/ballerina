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

# Represents an error.
public type Error error;

# Defines the name of the JSON Object key.
#
# + value - The name of the JSON Object key
public type NameConfig record {|
    string value;
|};

# The annotation is used to overwrite the existing record field name.
public const annotation NameConfig Name on record field;                                             

# Represents options for data projection.
public type Options record {
    # Allows data projection with specific settings.
    #
    # This configuration can be either a record or false. 
    # If it is a record, it contains `nilAsOptionalField` and `absentAsNilableType` options.
    # If it is set to `false`, data projection is not allowed.
    record {
        # If `true`, nil values will be considered as optional fields in the projection.
        boolean nilAsOptionalField = false;
        # If `true`, absent fields will be considered as nilable types in the projection.
        boolean absentAsNilableType = false;
    }|false allowDataProjection = {};
    # Lines to skip during processing, specified either as an array of integers or a string.
    int[]|string skipLines = [];
    # If `true`, enables validation of constraints during processing.
    boolean enableConstraintValidation = true;  
    # If `true`, when the result is a list it will contain headers as the first row.
    boolean outputWithHeaders = false;
    # Specifies the fail-safe options for handling errors during processing.
    # If enabled, metadata is printed to console/files based on the output mode configuration.
    FailSafeOptions failSafe?;
};

# Represents the options for fail-safe mechanism during parsing.
public type FailSafeOptions record {|
    # Specifies enabling logging errors to the console
    boolean enableConsoleLogs = true;
    # Includes logging source data to the console
    boolean includeSourceDataInConsole = false;
    # Specifies the output mode for logging errors encountered during parsing
    FileOutputMode fileOutputMode?;
|};

# Represents the file output mode for logging errors.
public type FileOutputMode record {|
    # The file path where errors will be logged
    string filePath;
    # Controls the level of detail included in the error logs.
    ErrorLogContentType contentType = METADATA;
    # Configuration for writing to the error file
    FileWriteOption fileWriteOption = APPEND;
|};

# Represents the content type for error logging.
public enum ErrorLogContentType {
    # Logs only the metadata (timestamp, location, message) without source data rows
    METADATA,
    # Logs the source data rows
    RAW,
    # Logs both source data rows and metadata along with error messages
    RAW_AND_METADATA
};

# Represents an error log entry.
public type LogOutput record {|
    # The timestamp of the error occurrence
    string time?;
    # The location where the error occurred
    Location location?;
    # The error message
    string message?;
    # The source data row related to the error
    string offendingRow?;
|};

# Represents the location of an error.
public type Location record {|
    # The row number where the error occurred
    int row;
    # The column number where the error occurred
    int column;
|};

# Represents the options for writing data.
public enum FileWriteOption {
    # If the file already exists, new logs will be appended to the existing file
    APPEND,
    # When the error logging starts, if the file already exists, the file will be overwritten
    OVERWRITE
};

# Represents the options for parsing data.
public type ParseOptions record {|
    *Options;
    # The delimiter character used for separating fields in the data.
    string:Char delimiter = ",";
    # The character encoding of the data.
    string encoding = "UTF-8";
    # The locale used for parsing.
    string locale = "en_US";
    # The character used to enclose text fields.
    string:Char textEnclosure = "\"";
    # The character used for escaping.
    string:Char escapeChar = "\\";
    # The line terminator(s) used in the data.
    LineTerminator|LineTerminator[] lineTerminator = [LF, CRLF];
    # The value to represent nil.
    NilValue? nilValue = ();
    # The character used to indicate comments in the data.
    string:Char comment = "#";
    # Specifies whether the header is present and, if so, the number of header lines.
    int:Unsigned32? header = 0;
    # Custom headers for the data, if headers are absent.
    string[]? customHeadersIfHeadersAbsent = ();
|};

# Represents options for treating a list as a record.
public type ParseListOptions record {|
    *Options;
    # If `0`, all the source data will treat as data rows.
    # Otherwise specify the header rows(Starts from 1) in the source data.
    int:Unsigned32 headerRows = 0;
    # Specify the header names of the source data.
    # This field will overwrite the header values in the header rows.
    # This will be mandatory if the header row parameter is larger than one.
    string[] customHeaders?;
|};

# Represents options for treating a list as a record.
public type TransformOptions record {|
    *Options;
    # Specify the order of the headers in the source data.
    # If the expected type is a subset of `record {}[]` this parameter will be ignored.
    string[]? headerOrder = ();
|};

# Enum representing possible line terminators.
public enum LineTerminator {
    # Line Feed (LF) line terminator: `\n`
    LF = "\n",
    # Carriage Return and Line Feed (CRLF) line terminator: `\r\n`
    CRLF = "\r\n"
};

# Enum representing possible nil values.
public enum NilValue {
    # Represents a nil value as the string "null".
    NULL = "null",
    # Represents a nil value as "N/A".
    NOT_APPLICABLE = "N/A",
    # Represents an empty string as a nil value.
    EMPTY_STRING = "",
    # Represents a nil value as Ballerina nil value `()`.
    NIL = "()"
};
