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

package native

import (
	"fmt"
)

// xmlError is an error jBallerina throws as a module error; it aborts the conversion unless a union member
// attempt catches it.
type xmlError struct {
	message string
}

func (e *xmlError) Error() string {
	return e.message
}

func newXMLError(format string, args ...any) *xmlError {
	return &xmlError{message: fmt.Sprintf(format, args...)}
}

func invalidTypeError(expected, found string) *xmlError {
	return newXMLError("invalid type expected '%s' but found '%s'", expected, found)
}

func xmlRootMissingError() *xmlError {
	return newXMLError("xml root element is missing")
}

func invalidRestTypeError(typeName string) *xmlError {
	return newXMLError("invalid rest type '%s'", typeName)
}

func arraySizeMismatchError() *xmlError {
	return newXMLError("array size is not compatible with the expected size")
}

func requiredFieldNotPresentError(fieldName string) *xmlError {
	return newXMLError("required field '%s' not present in XML", fieldName)
}

func requiredAttributeNotPresentError(fieldName string) *xmlError {
	return newXMLError("required attribute '%s' not present in XML", fieldName)
}

func duplicateFieldError(fieldName string) *xmlError {
	return newXMLError("duplicate field '%s'", fieldName)
}

func foundArrayForNonArrayTypeError(typeName, fieldName string) *xmlError {
	return newXMLError("expected '%s' value for the field '%s' found 'array' value", typeName, fieldName)
}

func expectedAnydataOrJSONError() *xmlError {
	return newXMLError("incompatible type expected anydata or json")
}

func namespaceMismatchError(typeName string) *xmlError {
	return newXMLError("namespace mismatched for the type: '%s'", typeName)
}

func typeNameMismatchError(name, localName string) *xmlError {
	return newXMLError("record type name '%s' mismatch with given XML name '%s'", name, localName)
}

func unsupportedTypeError() *xmlError {
	return newXMLError("unsupported input type")
}

func recoverAsParseError(err *error) {
	if r := recover(); r != nil {
		*err = newXMLError("failed to parse xml: %v", r)
	}
}

func xmlParseError(message string) *xmlError {
	return newXMLError("failed to parse xml: '%s'", message)
}

func undefinedFieldError(fieldName, recordName string) *xmlError {
	return newXMLError("undefined field '%s' in record '%s'", fieldName, recordName)
}

func cannotConvertSourceError(typeName string) *xmlError {
	return newXMLError("source value cannot be converted into '%s'", typeName)
}

func fieldCannotConvertError(fieldName, typeName string) *xmlError {
	return newXMLError("field '%s' cannot be converted into the type '%s'", fieldName, typeName)
}

// unionFieldCannotConvertError is the same diagnostic raised with only the type argument: Java's
// MessageFormat leaves the missing placeholder in the text.
func unionFieldCannotConvertError(typeName string) *xmlError {
	return newXMLError("field '%s' cannot be converted into the type '{1}'", typeName)
}

func invalidXMLError(message string) *xmlError {
	return newXMLError("Invalid XML found: '%s'", message)
}

func cannotConvertAttributeToArrayError(fieldName, typeName string) *xmlError {
	return newXMLError("attribute '%s' cannot be converted into the array type '%s'", fieldName, typeName)
}

func cannotConvertError(sourceType, value, typeName string) *xmlError {
	return newXMLError("'%s' value '%s' cannot be converted to '%s'", sourceType, value, typeName)
}
