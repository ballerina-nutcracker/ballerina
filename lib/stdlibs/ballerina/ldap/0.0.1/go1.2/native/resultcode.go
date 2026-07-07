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

package native

import (
	"errors"
	"fmt"

	goldap "github.com/go-ldap/ldap/v3"

	"ballerina-lang-go/values"
)

// statusByCode maps go-ldap's numeric LDAP result codes to the exact string
// literal declared by ldap:Status. Two codes are intentionally corrected
// relative to jBallerina's own (buggy) output — see README "Notable
// Behavioural Changes": StrongAuthRequired and NotAllowedOnNonLeaf.
var statusByCode = map[uint16]string{
	goldap.LDAPResultSuccess:                      "SUCCESS",
	goldap.LDAPResultOperationsError:              "OPERATIONS ERROR",
	goldap.LDAPResultProtocolError:                "PROTOCOL ERROR",
	goldap.LDAPResultTimeLimitExceeded:            "TIME LIMIT EXCEEDED",
	goldap.LDAPResultSizeLimitExceeded:            "SIZE LIMIT EXCEEDED",
	goldap.LDAPResultCompareFalse:                 "COMPARE FALSE",
	goldap.LDAPResultCompareTrue:                  "COMPARE TRUE",
	goldap.LDAPResultAuthMethodNotSupported:       "AUTH METHOD NOT SUPPORTED",
	goldap.LDAPResultStrongAuthRequired:           "STRONGER AUTH REQUIRED",
	goldap.LDAPResultReferral:                     "REFERRAL",
	goldap.LDAPResultAdminLimitExceeded:           "ADMIN LIMIT EXCEEDED",
	goldap.LDAPResultUnavailableCriticalExtension: "UNAVAILABLE CRITICAL EXTENSION",
	goldap.LDAPResultConfidentialityRequired:      "CONFIDENTIALITY REQUIRED",
	goldap.LDAPResultSaslBindInProgress:           "SASL BIND IN PROGRESS",
	goldap.LDAPResultNoSuchAttribute:              "NO SUCH ATTRIBUTE",
	goldap.LDAPResultUndefinedAttributeType:       "UNDEFINED ATTRIBUTE TYPE",
	goldap.LDAPResultInappropriateMatching:        "INAPPROPRIATE MATCHING",
	goldap.LDAPResultConstraintViolation:          "CONSTRAINT VIOLATION",
	goldap.LDAPResultAttributeOrValueExists:       "ATTRIBUTE OR VALUE EXISTS",
	goldap.LDAPResultInvalidAttributeSyntax:       "INVALID ATTRIBUTE SYNTAX",
	goldap.LDAPResultNoSuchObject:                 "NO SUCH OBJECT",
	goldap.LDAPResultAliasProblem:                 "ALIAS PROBLEM",
	goldap.LDAPResultInvalidDNSyntax:              "INVALID DN SYNTAX",
	goldap.LDAPResultAliasDereferencingProblem:    "ALIAS DEREFERENCING PROBLEM",
	goldap.LDAPResultInappropriateAuthentication:  "INAPPROPRIATE AUTHENTICATION",
	goldap.LDAPResultInvalidCredentials:           "INVALID CREDENTIALS",
	goldap.LDAPResultInsufficientAccessRights:     "INSUFFICIENT ACCESS RIGHTS",
	goldap.LDAPResultBusy:                         "BUSY",
	goldap.LDAPResultUnavailable:                  "UNAVAILABLE",
	goldap.LDAPResultUnwillingToPerform:           "UNWILLING TO PERFORM",
	goldap.LDAPResultLoopDetect:                   "LOOP DETECT",
	goldap.LDAPResultNamingViolation:              "NAMING VIOLATION",
	goldap.LDAPResultObjectClassViolation:         "OBJECT CLASS VIOLATION",
	goldap.LDAPResultNotAllowedOnNonLeaf:          "NOT ALLOWED ON NON LEAF",
	goldap.LDAPResultNotAllowedOnRDN:              "NOT ALLOWED ON RDN",
	goldap.LDAPResultEntryAlreadyExists:           "ENTRY ALREADY EXISTS",
	goldap.LDAPResultObjectClassModsProhibited:    "OBJECT CLASS MODS PROHIBITED",
	goldap.LDAPResultAffectsMultipleDSAs:          "AFFECTS MULTIPLE DSAS",
	goldap.LDAPResultOther:                        "OTHER",
}

// statusForCode returns the ldap:Status literal for an LDAP result code.
// Client-side-only codes (connection failures, timeouts, etc.) have no
// dedicated Status member in either jBallerina or this port, so they fall
// back to OTHER — see README "Notable Behavioural Changes".
func statusForCode(code uint16) string {
	if s, ok := statusByCode[code]; ok {
		return s
	}
	return "OTHER"
}

// ldapError builds an ldap:Error from a plain message, with no cause and no
// detail record. resultCode is not surfaced structurally (this interpreter's
// lang.error has no detail() yet — see README); callers that need it should
// rely on the LDAP result code already embedded in the message text below.
func ldapError(msg string) *values.Error {
	return values.NewErrorWithMessage(msg)
}

// ldapErrorFromErr converts a go-ldap operation error (or any other Go error,
// e.g. a network/dial failure) into an ldap:Error. When err wraps a
// *goldap.Error, its Error() text already embeds the numeric result code and
// the server's diagnostic message; the ldap:Status literal is appended so the
// exact enum string is still grep-able from message() text even though it
// isn't surfaced structurally (see the ldapError doc comment above).
func ldapErrorFromErr(err error) *values.Error {
	var ldapErr *goldap.Error
	if errors.As(err, &ldapErr) {
		return values.NewErrorWithMessage(fmt.Sprintf("%s (%s)", err.Error(), statusForCode(ldapErr.ResultCode)))
	}
	return values.NewErrorWithMessage(err.Error())
}
