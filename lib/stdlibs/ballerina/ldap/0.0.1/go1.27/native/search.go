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
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// scopeFromString maps an ldap:SearchScope literal to the go-ldap numeric scope.
func scopeFromString(scope string) int {
	switch scope {
	case "BASE":
		return goldap.ScopeBaseObject
	case "ONE":
		return goldap.ScopeSingleLevel
	case "SUBORDINATE_SUBTREE":
		return goldap.ScopeChildren
	default: // "SUB"
		return goldap.ScopeWholeSubtree
	}
}

// needsBase64Encoding decides whether an LDAP attribute value must be
// base64-encoded when surfaced as text. Like UnboundID, it treats every ASCII
// control byte as unsafe, not only RFC 2849's NUL/LF/CR.
func needsBase64Encoding(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	if raw[0] == ' ' || raw[0] == ':' || raw[0] == '<' {
		return true
	}
	if raw[len(raw)-1] == ' ' {
		return true
	}
	for _, b := range raw {
		if b < 0x20 || b >= 0x7f {
			return true
		}
	}
	return false
}

// objectGUIDToString formats a 16-byte Active Directory objectGUID value as a
// standard hyphenated GUID string, byte-swapping the first three components
// (they are stored little-endian).
func objectGUIDToString(b []byte) (string, error) {
	if len(b) != 16 {
		return "", fmt.Errorf("objectGUID must be 16 bytes, got %d", len(b))
	}
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[3], b[2], b[1], b[0], b[5], b[4], b[7], b[6], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15]), nil
}

// objectSidToString decodes a Windows SID binary value into its canonical
// S-1-<authority>-<subauth1>-... string form.
func objectSidToString(b []byte) (string, error) {
	if len(b) < 8 || b[0] != 1 {
		return "", fmt.Errorf("invalid objectSid: bad revision")
	}
	subAuthCount := int(b[1])
	if len(b) != 8+4*subAuthCount {
		return "", fmt.Errorf("invalid objectSid: length mismatch")
	}
	var authority uint64
	for i := 2; i < 8; i++ {
		authority = (authority << 8) | uint64(b[i])
	}
	sid := fmt.Sprintf("S-1-%d", authority)
	if authority >= 1<<32 {
		sid = fmt.Sprintf("S-1-0x%X", authority)
	}
	for i := 0; i < subAuthCount; i++ {
		off := 8 + i*4
		subAuth := uint32(b[off]) | uint32(b[off+1])<<8 | uint32(b[off+2])<<16 | uint32(b[off+3])<<24
		sid += fmt.Sprintf("-%d", subAuth)
	}
	return sid, nil
}

// attributeStringValues renders every raw value of an LDAP attribute as a
// Ballerina-facing string, applying the objectGUID/objectSid special cases
// before falling back to the generic base64-or-plain heuristic.
func attributeStringValues(attr *goldap.EntryAttribute) []string {
	name := strings.ToLower(attr.Name)
	vals := make([]string, len(attr.ByteValues))
	for i, raw := range attr.ByteValues {
		switch name {
		case "objectguid":
			if s, err := objectGUIDToString(raw); err == nil {
				vals[i] = s
				continue
			}
		case "objectsid":
			if s, err := objectSidToString(raw); err == nil {
				vals[i] = s
				continue
			}
		}
		if needsBase64Encoding(raw) {
			vals[i] = base64.StdEncoding.EncodeToString(raw)
		} else {
			vals[i] = string(raw)
		}
	}
	return vals
}

func stringsToBalList(types ldapTypes, ctx *extern.Context, vals []string) *values.List {
	balVals := make([]values.BalValue, len(vals))
	for i, s := range vals {
		balVals[i] = s
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.strArrTy)
	return values.NewList(types.strArrTy, atomic, true, nil, len(balVals), balVals)
}

func emptyMapArrList(types ldapTypes, ctx *extern.Context) *values.List {
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.mapArrTy)
	return values.NewList(types.mapArrTy, atomic, true, nil, 0, nil)
}

// buildEntryRecord builds a generic ldap:Entry record (all attribute values
// as string/string[]) from a search/getEntry result.
func buildEntryRecord(types ldapTypes, ctx *extern.Context, entry *goldap.Entry) *values.Map {
	entries := make([]values.MapEntry, 0, len(entry.Attributes))
	for _, attr := range entry.Attributes {
		if v := rawAttributeValue(types, ctx, attr); v != nil {
			entries = append(entries, values.MapEntry{Key: attr.Name, Value: v})
		}
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), types.entryTy)
	return values.NewMap(types.entryTy, atomic, false, entries)
}

func findAttribute(entry *goldap.Entry, name string) *goldap.EntryAttribute {
	for _, attr := range entry.Attributes {
		if strings.EqualFold(attr.Name, name) {
			return attr
		}
	}
	return nil
}

// attributesForRecordType returns the attribute names to request for a
// closed target record type (field names double as LDAP attribute names),
// or nil (meaning "fetch all") when the type is not a closed record — e.g.
// the generic ldap:Entry type, whose AttributeType... rest field means the
// set of attributes can't be inferred.
func attributesForRecordType(ctx *extern.Context, ty semtypes.SemType) []string {
	cx := ctx.TypeCtx()
	atomic := semtypes.ToMappingAtomicType(cx, ty)
	if atomic == nil {
		return nil
	}
	names := atomic.FieldNames()
	undeclared := semtypes.String
	for _, name := range names {
		undeclared = semtypes.Diff(undeclared, semtypes.StringConst(name))
	}
	if !semtypes.IsNever(semtypes.MappingMemberTypeInnerVal(cx, ty, undeclared)) {
		return nil
	}
	return names
}

// coerceAttributeValue converts an LDAP attribute's raw values into the Go
// representation matching the target field's declared type, mirroring
// Ballerina's implicit string-to-scalar conversion rules. A field that accepts
// string keeps the raw string; a value that can't be converted is also left as
// a string so the caller reports the mismatch.
func coerceAttributeValue(ctx *extern.Context, types ldapTypes, vals []string, fieldTy semtypes.SemType) values.BalValue {
	cx := ctx.TypeCtx()
	acceptsList := semtypes.IsSubtype(cx, types.strArrTy, fieldTy)
	if len(vals) > 1 || (acceptsList && !semtypes.IsSubtype(cx, semtypes.String, fieldTy)) {
		return stringsToBalList(types, ctx, vals)
	}
	s := vals[0]
	if semtypes.IsSubtype(cx, semtypes.String, fieldTy) {
		return s
	}
	switch {
	case semtypes.IsSubtype(cx, semtypes.Boolean, fieldTy):
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	case semtypes.IsSubtype(cx, semtypes.Int, fieldTy):
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	case semtypes.IsSubtype(cx, semtypes.Float, fieldTy):
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	case semtypes.IsSubtype(cx, semtypes.Decimal, fieldTy):
		if d, derr := decimal.FromString(s); derr == nil {
			return d
		}
	}
	return s
}

// rawAttributeValue surfaces an attribute as string, or string[] when it is
// multi-valued, matching jBallerina's untyped entry shape.
func rawAttributeValue(types ldapTypes, ctx *extern.Context, attr *goldap.EntryAttribute) values.BalValue {
	vals := attributeStringValues(attr)
	if len(vals) == 0 {
		return nil
	}
	if len(vals) == 1 {
		return vals[0]
	}
	return stringsToBalList(types, ctx, vals)
}

// coerceEntryToType builds a value of targetTy from an LDAP entry. Declared
// record fields are matched case-insensitively and coerced to their field type;
// remaining attributes are kept as-is when the target admits rest fields.
// CloneWithType then enforces required fields and field types.
func coerceEntryToType(ctx *extern.Context, types ldapTypes, entry *goldap.Entry, targetTy semtypes.SemType) (values.BalValue, *values.Error) {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), targetTy)
	if atomic == nil {
		return cloneEntryWithType(ctx, buildEntryRecord(types, ctx, entry), targetTy)
	}
	names := atomic.FieldNames()
	entries := make([]values.MapEntry, 0, len(entry.Attributes))
	matched := make(map[*goldap.EntryAttribute]bool, len(names))
	for _, name := range names {
		fieldTy := atomic.FieldInnerVal(name)
		attr := findAttribute(entry, name)
		var vals []string
		if attr != nil {
			matched[attr] = true
			vals = attributeStringValues(attr)
		}
		if len(vals) == 0 {
			if !atomic.IsOptional(ctx.TypeCtx(), name) {
				return nil, ldapError(fmt.Sprintf("missing required field '%s' of type '%s'",
					name, semtypes.ToString(ctx.TypeCtx(), fieldTy)))
			}
			continue
		}
		v := coerceAttributeValue(ctx, types, vals, fieldTy)
		if !semtypes.IsSubtype(ctx.TypeCtx(), values.SemTypeForValue(v), fieldTy) {
			return nil, ldapError(fmt.Sprintf("field '%s' should be of type '%s', found '%s'",
				name, semtypes.ToString(ctx.TypeCtx(), fieldTy), values.String(v, nil)))
		}
		entries = append(entries, values.MapEntry{Key: name, Value: v})
	}
	for _, attr := range entry.Attributes {
		if matched[attr] || semtypes.IsNever(atomic.FieldInnerVal(attr.Name)) {
			continue
		}
		if v := rawAttributeValue(types, ctx, attr); v != nil {
			entries = append(entries, values.MapEntry{Key: attr.Name, Value: v})
		}
	}
	entryAtomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), types.entryTy)
	return cloneEntryWithType(ctx, values.NewMap(types.entryTy, entryAtomic, false, entries), targetTy)
}

func cloneEntryWithType(ctx *extern.Context, entry *values.Map, targetTy semtypes.SemType) (values.BalValue, *values.Error) {
	result, convErr := values.CloneWithType(ctx.TypeCtx(), entry, targetTy)
	if convErr != nil {
		return nil, ldapError(convErr.Message)
	}
	return result, nil
}

func buildSearchReferences(types ldapTypes, ctx *extern.Context, referrals []string) *values.List {
	mapAtomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), types.mapTy)
	refs := make([]values.BalValue, len(referrals))
	for i, uri := range referrals {
		refs[i] = values.NewMap(types.mapTy, mapAtomic, false, []values.MapEntry{
			{Key: "messageId", Value: int64(0)},
			{Key: "uris", Value: stringsToBalList(types, ctx, []string{uri})},
			{Key: "controls", Value: emptyMapArrList(types, ctx)},
		})
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.mapArrTy)
	return values.NewList(types.mapArrTy, atomic, true, nil, len(refs), refs)
}

func buildSearchResultRecord(types ldapTypes, ctx *extern.Context, entries []*goldap.Entry, referrals []string) *values.Map {
	mapEntries := []values.MapEntry{{Key: "resultCode", Value: "SUCCESS"}}
	if len(referrals) > 0 {
		mapEntries = append(mapEntries, values.MapEntry{Key: "searchReferences", Value: buildSearchReferences(types, ctx, referrals)})
	}
	if len(entries) > 0 {
		entryVals := make([]values.BalValue, len(entries))
		for i, e := range entries {
			entryVals[i] = buildEntryRecord(types, ctx, e)
		}
		atomic := semtypes.ToListAtomicType(ctx.TypeEnv(), types.entryArrTy)
		mapEntries = append(mapEntries, values.MapEntry{
			Key: "entries", Value: values.NewList(types.entryArrTy, atomic, true, nil, len(entryVals), entryVals),
		})
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx(), types.mapTy)
	return values.NewMap(types.mapTy, atomic, false, mapEntries)
}

func stringSliceArg(v values.BalValue) []string {
	list, ok := v.(*values.List)
	if !ok {
		return nil
	}
	out := make([]string, list.Len())
	for i := 0; i < list.Len(); i++ {
		out[i], _ = list.Get(i).(string)
	}
	return out
}

func registerSearchFunctions(rt *runtime.Runtime, types ldapTypes) {
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$getEntry", getEntryExtern(types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$search", searchExtern(types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$searchWithType", searchWithTypeExtern(types))
}

func entryNotFoundError(dn string) *values.Error {
	return ldapError(fmt.Sprintf("Entry is not found for DN: '%s'", dn))
}

func getEntryExtern(types ldapTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		dn := args[1].(string)
		targetTD := args[len(args)-1].(*values.TypeDesc)
		var attrs []string
		if len(args) == 4 {
			attrs = stringSliceArg(args[2])
		}
		if len(attrs) == 0 {
			attrs = attributesForRecordType(ctx, targetTD.Constraint())
		}
		conn, connErr := requireConn(self)
		if connErr != nil {
			return connErr, nil
		}
		req := goldap.NewSearchRequest(dn, goldap.ScopeBaseObject, goldap.NeverDerefAliases,
			0, 0, false, "(objectClass=*)", attrs, nil)
		result, err := conn.Search(req)
		if err != nil {
			return ldapErrorFromErr(err), nil
		}
		if len(result.Entries) == 0 {
			return entryNotFoundError(dn), nil
		}
		value, convErr := coerceEntryToType(ctx, types, result.Entries[0], targetTD.Constraint())
		if convErr != nil {
			return convErr, nil
		}
		return value, nil
	}
}

func searchExtern(types ldapTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		baseDn := args[1].(string)
		filter := args[2].(string)
		scope := args[3].(string)
		var attrs []string
		if len(args) == 5 {
			attrs = stringSliceArg(args[4])
		}
		conn, connErr := requireConn(self)
		if connErr != nil {
			return connErr, nil
		}
		req := goldap.NewSearchRequest(baseDn, scopeFromString(scope), goldap.NeverDerefAliases,
			0, 0, false, filter, attrs, nil)
		result, err := conn.Search(req)
		if err != nil {
			return ldapErrorFromErr(err), nil
		}
		if len(result.Entries) == 0 {
			return entryNotFoundError(baseDn), nil
		}
		return buildSearchResultRecord(types, ctx, result.Entries, result.Referrals), nil
	}
}

func searchWithTypeExtern(types ldapTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self := args[0].(*values.Object)
		baseDn := args[1].(string)
		filter := args[2].(string)
		scope := args[3].(string)
		targetTD := args[len(args)-1].(*values.TypeDesc)
		var attrs []string
		if len(args) == 6 {
			attrs = stringSliceArg(args[4])
		}
		// ToListAtomicType can't resolve the Bdd for a typedesc-sourced
		// array-of-record SemType (unlike the analogous mapping case
		// used by getEntry, which works fine) — use the member-type
		// projection instead, which is the same primitive
		// stream_member.go uses for index-based access.
		elemTy := semtypes.ListMemberTypeInnerVal(ctx.TypeCtx(), targetTD.Constraint(), semtypes.IntConst(0))
		if len(attrs) == 0 {
			attrs = attributesForRecordType(ctx, elemTy)
		}
		conn, connErr := requireConn(self)
		if connErr != nil {
			return connErr, nil
		}
		req := goldap.NewSearchRequest(baseDn, scopeFromString(scope), goldap.NeverDerefAliases,
			0, 0, false, filter, attrs, nil)
		result, err := conn.Search(req)
		if err != nil {
			return ldapErrorFromErr(err), nil
		}
		if len(result.Entries) == 0 {
			return entryNotFoundError(baseDn), nil
		}
		entryVals := make([]values.BalValue, len(result.Entries))
		for i, e := range result.Entries {
			value, convErr := coerceEntryToType(ctx, types, e, elemTy)
			if convErr != nil {
				return convErr, nil
			}
			entryVals[i] = value
		}
		// Tag the result with a freshly built elemTy[] rather than the
		// original targetTD.Constraint(), for the same reason as above.
		resultListBld := semtypes.NewListDefinition()
		resultListTy := resultListBld.Define(ctx.TypeEnv(), nil, semtypes.ListRest(elemTy))
		resultAtomic := semtypes.ToListAtomicType(ctx.TypeEnv(), resultListTy)
		return values.NewList(resultListTy, resultAtomic, false, nil, len(entryVals), entryVals), nil
	}
}
