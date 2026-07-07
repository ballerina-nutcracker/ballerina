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

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
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

// needsBase64Encoding implements the RFC 2849 "safe string" check that
// decides whether an LDAP attribute value must be base64-encoded when
// surfaced as text (mirrors UnboundID's Attribute.needsBase64Encoding()).
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
		if b == 0 || b == '\n' || b == '\r' || b >= 0x80 {
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
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.strArrTy)
	return values.NewList(types.strArrTy, atomic, true, nil, len(balVals), balVals)
}

func emptyMapArrList(types ldapTypes, ctx *extern.Context) *values.List {
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.mapArrTy)
	return values.NewList(types.mapArrTy, atomic, true, nil, 0, nil)
}

// buildEntryRecord builds a generic ldap:Entry record (all attribute values
// as string/string[]) from a search/getEntry result.
func buildEntryRecord(types ldapTypes, ctx *extern.Context, entry *goldap.Entry) *values.Map {
	entries := make([]values.MapEntry, 0, len(entry.Attributes))
	for _, attr := range entry.Attributes {
		vals := attributeStringValues(attr)
		if len(vals) == 0 {
			continue
		}
		var v values.BalValue = vals[0]
		if len(vals) > 1 {
			v = stringsToBalList(types, ctx, vals)
		}
		entries = append(entries, values.MapEntry{Key: attr.Name, Value: v})
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, types.mapTy)
	return values.NewMap(types.mapTy, atomic, false, entries)
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
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, ty)
	if atomic == nil || !semtypes.IsNever(atomic.Rest) {
		return nil
	}
	return atomic.Names
}

// coerceAttributeValue converts an LDAP attribute's raw values into the Go
// representation matching the target field's declared type, mirroring
// Ballerina's implicit string-to-scalar conversion rules.
func coerceAttributeValue(ctx *extern.Context, types ldapTypes, vals []string, fieldTy semtypes.SemType) values.BalValue {
	if len(vals) == 0 {
		return nil
	}
	if len(vals) > 1 && semtypes.IsSubtype(ctx.TypeCtx, types.strArrTy, fieldTy) {
		return stringsToBalList(types, ctx, vals)
	}
	s := vals[0]
	switch {
	case semtypes.IsSubtype(ctx.TypeCtx, semtypes.BOOLEAN, fieldTy):
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	case semtypes.IsSubtype(ctx.TypeCtx, semtypes.INT, fieldTy):
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	case semtypes.IsSubtype(ctx.TypeCtx, semtypes.FLOAT, fieldTy):
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	case semtypes.IsSubtype(ctx.TypeCtx, semtypes.DECIMAL, fieldTy):
		if d, derr := decimal.FromString(s); derr == nil {
			return d
		}
	}
	return s
}

// coerceEntryToType builds a record value of targetTy from an LDAP entry,
// walking the target's declared fields and coercing each matching attribute.
// Falls back to a generic Entry-shaped map if targetTy isn't a closed record.
func coerceEntryToType(ctx *extern.Context, types ldapTypes, entry *goldap.Entry, targetTy semtypes.SemType) values.BalValue {
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, targetTy)
	if atomic == nil {
		return buildEntryRecord(types, ctx, entry)
	}
	entries := make([]values.MapEntry, 0, len(atomic.Names))
	for i, name := range atomic.Names {
		attr := findAttribute(entry, name)
		if attr == nil {
			continue
		}
		vals := attributeStringValues(attr)
		entries = append(entries, values.MapEntry{Key: name, Value: coerceAttributeValue(ctx, types, vals, atomic.Types[i])})
	}
	return values.NewMap(targetTy, atomic, false, entries)
}

func buildSearchReferences(types ldapTypes, ctx *extern.Context, referrals []string) *values.List {
	mapAtomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, types.mapTy)
	refs := make([]values.BalValue, len(referrals))
	for i, uri := range referrals {
		refs[i] = values.NewMap(types.mapTy, mapAtomic, false, []values.MapEntry{
			{Key: "messageId", Value: int64(0)},
			{Key: "uris", Value: stringsToBalList(types, ctx, []string{uri})},
			{Key: "controls", Value: emptyMapArrList(types, ctx)},
		})
	}
	atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.mapArrTy)
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
		atomic := semtypes.ToListAtomicType(ctx.TypeCtx, types.mapArrTy)
		mapEntries = append(mapEntries, values.MapEntry{
			Key: "entries", Value: values.NewList(types.mapArrTy, atomic, true, nil, len(entryVals), entryVals),
		})
	}
	atomic := semtypes.ToMappingAtomicType(ctx.TypeCtx, types.mapTy)
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
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$getEntry",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self := args[0].(*values.Object)
			dn := args[1].(string)
			targetTD := args[len(args)-1].(*values.TypeDesc)
			var attrs []string
			if len(args) == 4 {
				attrs = stringSliceArg(args[2])
			}
			if len(attrs) == 0 {
				attrs = attributesForRecordType(ctx, targetTD.Type)
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
				return ldapError(fmt.Sprintf("entry not found: %s (NO_SUCH_OBJECT)", dn)), nil
			}
			return coerceEntryToType(ctx, types, result.Entries[0], targetTD.Type), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$search",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
				return ldapError(fmt.Sprintf("no entries found for baseDN %q, filter %q (OTHER)", baseDn, filter)), nil
			}
			return buildSearchResultRecord(types, ctx, result.Entries, result.Referrals), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Client.$remote$searchWithType",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
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
			elemTy := semtypes.ListMemberTypeInnerVal(ctx.TypeCtx, targetTD.Type, semtypes.IntConst(0))
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
				return ldapError(fmt.Sprintf("no entries found for baseDN %q, filter %q (OTHER)", baseDn, filter)), nil
			}
			entryVals := make([]values.BalValue, len(result.Entries))
			for i, e := range result.Entries {
				entryVals[i] = coerceEntryToType(ctx, types, e, elemTy)
			}
			// Tag the result with a freshly built elemTy[] rather than the
			// original targetTD.Type, for the same reason as above.
			resultListBld := semtypes.NewListDefinition()
			resultListTy := resultListBld.DefineListTypeWrappedWithEnvSemType(ctx.Env.TypeEnv, elemTy)
			resultAtomic := semtypes.ToListAtomicType(ctx.TypeCtx, resultListTy)
			return values.NewList(resultListTy, resultAtomic, false, nil, len(entryVals), entryVals), nil
		})
}
