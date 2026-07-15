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
	"regexp"
	"strconv"
	"strings"
	"time"

	"ballerina-lang-go/decimal"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/semtypes"
	"ballerina-lang-go/values"
)

const (
	orgName    = "ballerina"
	moduleName = "time"
)

// secondsHavePattern detects presence of seconds in an RFC 3339 datetime string.
var secondsHavePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

// utcOnlyPattern matches RFC 3339 strings ending in bare Z (no fixed offset in Civil record).
var utcOnlyPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z$)`)

// ianaZoneSuffixPattern matches the RFC 9557 IANA zone annotation "[Zone/Name]" at end of string.
var ianaZoneSuffixPattern = regexp.MustCompile(`\[([^\]]+)\]$`)

// emailCommentPattern extracts the optional comment like "(PST)" from RFC 5322 date strings.
var emailCommentPattern = regexp.MustCompile(`\(([^)]*)\)`)

// nanosPerSec is 1,000,000,000 as a decimal constant used for ns ↔ seconds conversion.
var nanosPerSec = decimal.FromInt64(1_000_000_000)

// utcToGoTime converts a Ballerina Utc tuple [int, decimal] to a Go time.Time in UTC.
func utcToGoTime(utc *values.List) time.Time {
	epochSec, _ := utc.Get(0).(int64)
	frac, _ := utc.Get(1).(*decimal.Decimal)
	nanos := int64(0)
	if frac != nil {
		nanos = decimalToNanos(frac)
	}
	return time.Unix(epochSec, nanos).UTC()
}

// decimalToNanos converts a decimal number of seconds to nanoseconds using
// pure decimal arithmetic to avoid float64 rounding drift.
func decimalToNanos(d *decimal.Decimal) int64 {
	if d == nil {
		return 0
	}
	product, _ := d.Mul(nanosPerSec)
	n, _, _ := product.Int64()
	return n
}

// nanosToFrac converts nanoseconds to a Ballerina Seconds fraction [0,1).
func nanosToFrac(nanos int64) *decimal.Decimal {
	n := decimal.FromInt64(nanos)
	frac, _ := n.Quo(nanosPerSec)
	return frac
}

// goTimeToUtc converts a Go time.Time to a Ballerina Utc tuple.
func goTimeToUtc(utcTy semtypes.SemType, tc semtypes.Context, t time.Time) *values.List {
	t = t.UTC()
	return buildUtcTuple(utcTy, tc, t.Unix(), nanosToFrac(int64(t.Nanosecond())))
}

// goTimeToUtcWithPrecision converts a Go time.Time to a Utc tuple, rounding fractional seconds
// to the given decimal precision using Go's Round (ties away from zero = HALF_UP for positive values).
func goTimeToUtcWithPrecision(utcTy semtypes.SemType, tc semtypes.Context, t time.Time, precision int) *values.List {
	t = t.UTC()
	unit := time.Duration(roundingUnit(precision))
	t = t.Round(unit)
	return buildUtcTuple(utcTy, tc, t.Unix(), nanosToFrac(int64(t.Nanosecond())))
}

// roundingUnit returns the nanosecond unit for the given decimal precision (0..9).
func roundingUnit(precision int) int64 {
	unit := int64(1)
	for i := 0; i < 9-precision; i++ {
		unit *= 10
	}
	return unit
}

// buildUtcTuple wraps epochSec and frac into a readonly [int, decimal] Ballerina tuple.
func buildUtcTuple(utcTy semtypes.SemType, tc semtypes.Context, epochSec int64, frac *decimal.Decimal) *values.List {
	atomic := semtypes.ToListAtomicType(tc, utcTy)
	items := []values.BalValue{epochSec, frac}
	return values.NewList(utcTy, atomic, true, nil, 2, items)
}

// formatInstantNanos returns the sub-second suffix matching Java's Instant.toString() grouping:
// 0 ns → ""; otherwise the smallest of ".NNN", ".NNNNNN", ".NNNNNNNNN" that captures all info.
func formatInstantNanos(nanos int) string {
	if nanos == 0 {
		return ""
	}
	s := fmt.Sprintf("%09d", nanos)
	switch {
	case nanos%1_000_000 == 0:
		return "." + s[:3]
	case nanos%1_000 == 0:
		return "." + s[:6]
	default:
		return "." + s
	}
}

// formatRFC3339Instant formats t as an RFC 3339 string matching Java's Instant.toString().
func formatRFC3339Instant(t time.Time) string {
	t = t.UTC()
	return t.Format("2006-01-02T15:04:05") + formatInstantNanos(t.Nanosecond()) + "Z"
}

// formatRFC3339WithOffset formats t with a fixed zone offset (seconds east of UTC),
// matching Java's ZonedDateTime.toString() for fixed-offset zones.
func formatRFC3339WithOffset(t time.Time, offsetSecs int) string {
	if offsetSecs == 0 {
		return formatRFC3339Instant(t.UTC())
	}
	sign := "+"
	if offsetSecs < 0 {
		sign = "-"
		offsetSecs = -offsetSecs
	}
	return t.Format("2006-01-02T15:04:05") + formatInstantNanos(t.Nanosecond()) +
		fmt.Sprintf("%s%02d:%02d", sign, offsetSecs/3600, (offsetSecs%3600)/60)
}

// decimalToSecNano splits a Ballerina Seconds decimal into whole seconds and nanoseconds.
// Uses decimal arithmetic for the fractional part to avoid float64 precision errors
// (e.g., float64(0.52)*1e9 = 519999999, not 520000000).
func decimalToSecNano(sec *decimal.Decimal) (int, int) {
	if sec == nil {
		return 0, 0
	}
	f := sec.Float64()
	intSec := int(f)
	// Subtract the whole part using exact decimal arithmetic, then multiply by 1e9.
	intSecDec := decimal.FromInt64(int64(intSec))
	fracDec, _ := sec.Sub(intSecDec)
	nanosDec, _ := fracDec.Mul(nanosPerSec)
	nanosInt, _, _ := nanosDec.Int64()
	return intSec, int(nanosInt)
}

// civilFixedOffsetTime builds a time.Time from Civil map fields using a fixed zone offset.
// The zone is named using the offset string (e.g., "+05:30") so it can be round-tripped
// through zoneAbbrevFor → civilNamedZoneTime, matching Java's ZoneId.of("+05:30") behavior.
func civilFixedOffsetTime(m *values.Map) (time.Time, int, error) {
	year := int(mapInt(m, "year"))
	month := int(mapInt(m, "month"))
	day := int(mapInt(m, "day"))
	hour := int(mapInt(m, "hour"))
	minute := int(mapInt(m, "minute"))
	second := mapDecimal(m, "second")
	intSec, nanos := decimalToSecNano(second)

	offsetHours, offsetMinutes, offsetSeconds := zoneOffsetFields(m)
	totalOffset := offsetHours*3600 + offsetMinutes*60 + offsetSeconds

	loc := time.FixedZone(offsetToZoneName(totalOffset), totalOffset)
	t := time.Date(year, time.Month(month), day, hour, minute, intSec, nanos, loc)
	if t.Day() != day || int(t.Month()) != month || t.Year() != year {
		return time.Time{}, 0, fmt.Errorf("invalid date: %04d-%02d-%02d", year, month, day)
	}
	return t, totalOffset, nil
}

// offsetToZoneName formats totalOffsetSecs as "+HH:MM" or "-HH:MM", matching Java's
// ZoneOffset.ofTotalSeconds(n).toString() which produces the same format.
func offsetToZoneName(totalOffsetSecs int) string {
	if totalOffsetSecs == 0 {
		return "Z"
	}
	sign := "+"
	if totalOffsetSecs < 0 {
		sign = "-"
		totalOffsetSecs = -totalOffsetSecs
	}
	return fmt.Sprintf("%s%02d:%02d", sign, totalOffsetSecs/3600, (totalOffsetSecs%3600)/60)
}

// civilNamedZoneTime builds a time.Time from Civil map fields using a named timezone.
func civilNamedZoneTime(m *values.Map) (time.Time, int, error) {
	year := int(mapInt(m, "year"))
	month := int(mapInt(m, "month"))
	day := int(mapInt(m, "day"))
	hour := int(mapInt(m, "hour"))
	minute := int(mapInt(m, "minute"))
	second := mapDecimal(m, "second")
	intSec, nanos := decimalToSecNano(second)
	abbrev := mapString(m, "timeAbbrev")

	// "Z" is UTC; "+HH:MM"/"-HH:MM" are fixed-offset zones (Java ZoneId.of behavior);
	// everything else is an IANA name.
	var loc *time.Location
	switch {
	case abbrev == "Z" || strings.ToLower(abbrev) == "z":
		loc = time.UTC
	case isNumericOffset(abbrev):
		offsetSecs, err := parseNumericOffset(abbrev)
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("invalid zone offset: %s", abbrev)
		}
		loc = time.FixedZone(abbrev, offsetSecs)
	default:
		var err error
		loc, err = time.LoadLocation(abbrev)
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("unknown time zone: %s", abbrev)
		}
	}
	t := time.Date(year, time.Month(month), day, hour, minute, intSec, nanos, loc)
	_, offsetSecs := t.Zone()
	return t, offsetSecs, nil
}

// zoneHandlingFor returns the zone handling string based on the Civil record fields.
func zoneHandlingFor(m *values.Map) (string, error) {
	_, hasOffset := m.Get("utcOffset")
	_, hasAbbrev := m.Get("timeAbbrev")
	if !hasOffset && !hasAbbrev {
		return "", fmt.Errorf("the civil value should have either `utcOffset` or `timeAbbrev`")
	}
	if !hasOffset {
		return "PREFER_TIME_ABBREV", nil
	}
	return "PREFER_ZONE_OFFSET", nil
}

// civilToGoTime dispatches on whether the Civil record has a utcOffset or timeAbbrev field.
func civilToGoTime(m *values.Map) (time.Time, int, error) {
	handling, err := zoneHandlingFor(m)
	if err != nil {
		return time.Time{}, 0, err
	}
	if handling == "PREFER_TIME_ABBREV" {
		return civilNamedZoneTime(m)
	}
	return civilFixedOffsetTime(m)
}

// buildCivil constructs a Ballerina Civil map from a Go time.Time (always includes second field).
// Matches Java's Civil.build().
func buildCivil(tc semtypes.Context, t time.Time) *values.Map {
	m := values.NewMap(semtypes.MAPPING, semtypes.ToMappingAtomicType(tc, semtypes.MAPPING), false, nil)
	putCivilCommonFields(tc, m, t)
	m.Put(tc, "second", timeToSecondDecimal(t))
	return m
}

// buildCivilWithZone constructs a Civil map, optionally including second and utcOffset.
// Matches Java's Civil.buildWithZone().
func buildCivilWithZone(tc semtypes.Context, t time.Time, includeSecond, includeOffset bool) *values.Map {
	m := values.NewMap(semtypes.MAPPING, semtypes.ToMappingAtomicType(tc, semtypes.MAPPING), false, nil)
	putCivilCommonFields(tc, m, t)
	if includeSecond {
		m.Put(tc, "second", timeToSecondDecimal(t))
	}
	if includeOffset {
		_, offsetSecs := t.Zone()
		m.Put(tc, "utcOffset", buildZoneOffset(tc, offsetSecs))
	}
	return m
}

func putCivilCommonFields(tc semtypes.Context, m *values.Map, t time.Time) {
	m.Put(tc, "year", int64(t.Year()))
	m.Put(tc, "month", int64(t.Month()))
	m.Put(tc, "day", int64(t.Day()))
	m.Put(tc, "hour", int64(t.Hour()))
	m.Put(tc, "minute", int64(t.Minute()))
	m.Put(tc, "timeAbbrev", zoneAbbrevFor(t))
	m.Put(tc, "dayOfWeek", int64(t.Weekday())) // Go Weekday: Sunday=0..Saturday=6 matches Ballerina
}

// zoneAbbrevFor returns the zone abbreviation string for a time, mapping Go's "UTC" to "Z"
// to match Java's ZoneId.of("Z").toString() = "Z" behavior.
func zoneAbbrevFor(t time.Time) string {
	name := t.Location().String()
	if name == "UTC" {
		return "Z"
	}
	return name
}

// timeToSecondDecimal converts a time.Time's second+nanosecond to a Ballerina Seconds decimal.
func timeToSecondDecimal(t time.Time) *decimal.Decimal {
	intSec := decimal.FromInt64(int64(t.Second()))
	frac := nanosToFrac(int64(t.Nanosecond()))
	sec, _ := intSec.Add(frac)
	return sec
}

// buildZoneOffset creates a Ballerina ZoneOffset map from total offset seconds east of UTC.
func buildZoneOffset(tc semtypes.Context, totalOffsetSecs int) *values.Map {
	sign := 1
	if totalOffsetSecs < 0 {
		sign = -1
		totalOffsetSecs = -totalOffsetSecs
	}
	zm := values.NewMap(semtypes.MAPPING, semtypes.ToMappingAtomicType(tc, semtypes.MAPPING), false, nil)
	zm.Put(tc, "hours", int64(sign*(totalOffsetSecs/3600)))
	zm.Put(tc, "minutes", int64(sign*((totalOffsetSecs%3600)/60)))
	return zm
}

// zoneOffsetFields extracts hours, minutes, seconds from a Civil's utcOffset sub-record.
func zoneOffsetFields(civil *values.Map) (hours, minutes, seconds int) {
	v, ok := civil.Get("utcOffset")
	if !ok {
		return 0, 0, 0
	}
	zm, ok := v.(*values.Map)
	if !ok {
		return 0, 0, 0
	}
	hours = int(mapInt(zm, "hours"))
	minutes = int(mapInt(zm, "minutes"))
	if sv, ok := zm.Get("seconds"); ok {
		if d, ok := sv.(*decimal.Decimal); ok {
			seconds = int(d.Float64())
		}
	}
	return
}

// formatEmailDate formats t as an RFC 5322 date string with non-zero-padded day,
// matching Java's DateTimeFormatter.RFC_1123_DATE_TIME output.
func formatEmailDate(t time.Time, offsetStr string) string {
	return fmt.Sprintf("%s, %d %s %04d %02d:%02d:%02d %s",
		t.Format("Mon"), t.Day(), t.Format("Jan"),
		t.Year(), t.Hour(), t.Minute(), t.Second(),
		offsetStr,
	)
}

// parseEmailDate parses RFC 5322/RFC 1123 date strings (with optional non-padded day).
func parseEmailDate(s string) (time.Time, error) {
	return time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", normaliseEmailDay(s))
}

// normaliseEmailDay pads a single-digit day to two digits for Go's time parser.
// Input: "Mon, 3 Dec …" → "Mon, 03 Dec …"
func normaliseEmailDay(s string) string {
	if len(s) > 6 && s[5] >= '0' && s[5] <= '9' && s[6] == ' ' {
		return s[:5] + "0" + s[5:]
	}
	return s
}

// emailOffsetStr maps a Ballerina UtcZoneHandling value to the offset string used in RFC 5322 output,
// matching Java: zh="0" → "+0000"; others used verbatim.
func emailOffsetStr(zh string) string {
	if zh == "0" {
		return "+0000"
	}
	return zh
}

func newFormatError(msg string) *values.Error {
	return values.NewError(semtypes.ERROR, msg, nil, "FormatError", nil)
}

func mapInt(m *values.Map, key string) int64 {
	v, _ := m.Get(key)
	if n, ok := v.(int64); ok {
		return n
	}
	return 0
}

func mapDecimal(m *values.Map, key string) *decimal.Decimal {
	v, _ := m.Get(key)
	if d, ok := v.(*decimal.Decimal); ok {
		return d
	}
	return decimal.FromInt64(0)
}

func mapString(m *values.Map, key string) string {
	v, _ := m.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getInt64Arg(args []values.BalValue, idx int) int64 {
	if idx < len(args) {
		if n, ok := args[idx].(int64); ok {
			return n
		}
	}
	return 0
}

func getStringArg(args []values.BalValue, idx int) string {
	if idx < len(args) {
		if s, ok := args[idx].(string); ok {
			return s
		}
	}
	return ""
}

// getStoredLocation retrieves the *time.Location stored in a TimeZone object.
func getStoredLocation(obj *values.Object) *time.Location {
	v, _ := obj.Get("$location")
	if loc, ok := v.(*time.Location); ok {
		return loc
	}
	return time.UTC
}

// civilMapToGoTimeInLocation converts a Civil map to a time.Time using a specific location.
func civilMapToGoTimeInLocation(m *values.Map, loc *time.Location) (time.Time, error) {
	year := int(mapInt(m, "year"))
	month := int(mapInt(m, "month"))
	day := int(mapInt(m, "day"))
	hour := int(mapInt(m, "hour"))
	minute := int(mapInt(m, "minute"))
	second := mapDecimal(m, "second")
	intSec, nanos := decimalToSecNano(second)
	t := time.Date(year, time.Month(month), day, hour, minute, intSec, nanos, loc)
	if t.Day() != day || int(t.Month()) != month || t.Year() != year {
		return time.Time{}, fmt.Errorf("invalid date: %04d-%02d-%02d", year, month, day)
	}
	return t, nil
}

func initTimeModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	utcBld := semtypes.NewListDefinition()
	utcTy := utcBld.TupleTypeWrappedRo(env, semtypes.INT, semtypes.DECIMAL)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcNow",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			precision := int(getInt64Arg(args, 0))
			now := rt.Platform().Time.Now()
			if precision < 0 {
				return goTimeToUtc(utcTy, ctx.TypeCtx, now), nil
			}
			return goTimeToUtcWithPrecision(utcTy, ctx.TypeCtx, now, precision), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externMonotonicNow",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			d := rt.Platform().Time.MonotonicNow()
			nanos := decimal.FromInt64(d.Nanoseconds())
			result, _ := nanos.Quo(nanosPerSec)
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcFromString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			str := getStringArg(args, 0)
			t, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				return newFormatError(fmt.Sprintf(
					"The provided string '%s' does not adhere to the expected RFC 3339 format 'YYYY-MM-DDTHH:MM:SS.SSZ'. ", str)), nil
			}
			return goTimeToUtc(utcTy, ctx.TypeCtx, t), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcToString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			utc, _ := args[0].(*values.List)
			return formatRFC3339Instant(utcToGoTime(utc)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcAddSeconds",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			utc, _ := args[0].(*values.List)
			seconds, _ := args[1].(*decimal.Decimal)
			t := utcToGoTime(utc)
			if seconds != nil {
				dur := time.Duration(decimalToNanos(seconds))
				t = t.Add(dur)
			}
			return goTimeToUtc(utcTy, ctx.TypeCtx, t), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcDiffSeconds",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			utc1, _ := args[0].(*values.List)
			utc2, _ := args[1].(*values.List)
			diff := utcToGoTime(utc1).Sub(utcToGoTime(utc2))
			nanos := decimal.FromInt64(diff.Nanoseconds())
			result, _ := nanos.Quo(nanosPerSec)
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externDateValidate",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			m, _ := args[0].(*values.Map)
			year := int(mapInt(m, "year"))
			month := int(mapInt(m, "month"))
			day := int(mapInt(m, "day"))
			if month < 1 || month > 12 {
				return newFormatError(fmt.Sprintf("invalid month: %d", month)), nil
			}
			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			if t.Day() != day || int(t.Month()) != month || t.Year() != year {
				return newFormatError(fmt.Sprintf("invalid date: %04d-%02d-%02d", year, month, day)), nil
			}
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externDayOfWeek",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			m, _ := args[0].(*values.Map)
			year := int(mapInt(m, "year"))
			month := int(mapInt(m, "month"))
			day := int(mapInt(m, "day"))
			if month < 1 || month > 12 {
				return newFormatError(fmt.Sprintf("invalid month: %d", month)), nil
			}
			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			if t.Day() != day || int(t.Month()) != month {
				return newFormatError(fmt.Sprintf("invalid date: %04d-%02d-%02d", year, month, day)), nil
			}
			return int64(t.Weekday()), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcToCivil",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			utc, _ := args[0].(*values.List)
			return buildCivil(ctx.TypeCtx, utcToGoTime(utc)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcFromCivil",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			m, _ := args[0].(*values.Map)
			_, hasOffset := m.Get("utcOffset")
			abbrevVal, hasAbbrev := m.Get("timeAbbrev")

			var t time.Time
			var err error
			if !hasOffset {
				if hasAbbrev {
					if abbrev, ok := abbrevVal.(string); ok && strings.ToLower(abbrev) == "z" {
						t, _, err = civilFixedOffsetTime(m)
					} else {
						return newFormatError("civilTime.utcOffset must not be null"), nil
					}
				} else {
					return newFormatError("civilTime.utcOffset must not be null"), nil
				}
			} else {
				t, _, err = civilFixedOffsetTime(m)
			}
			if err != nil {
				return newFormatError(err.Error()), nil
			}
			return goTimeToUtc(utcTy, ctx.TypeCtx, t), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCivilFromString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			str := getStringArg(args, 0)
			parseStr := str
			ianaZone := ""
			if mm := ianaZoneSuffixPattern.FindStringSubmatchIndex(parseStr); mm != nil {
				ianaZone = parseStr[mm[2]:mm[3]]
				parseStr = parseStr[:mm[0]]
			}
			t, err := time.Parse(time.RFC3339Nano, parseStr)
			if err != nil {
				return newFormatError(fmt.Sprintf("invalid date-time string: %s", str)), nil
			}
			hasSeconds := secondsHavePattern.MatchString(parseStr)
			isUTCOnly := utcOnlyPattern.MatchString(parseStr)
			result := buildCivilWithZone(ctx.TypeCtx, t, hasSeconds, !isUTCOnly)
			if ianaZone != "" {
				result.Put(ctx.TypeCtx, "timeAbbrev", ianaZone)
			}
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCivilToString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			m, _ := args[0].(*values.Map)
			t, offsetSecs, err := civilToGoTime(m)
			if err != nil {
				return newFormatError(err.Error()), nil
			}
			return formatRFC3339WithOffset(t, offsetSecs), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externUtcToEmailString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			utc, _ := args[0].(*values.List)
			zh := getStringArg(args, 1)
			return formatEmailDate(utcToGoTime(utc).UTC(), emailOffsetStr(zh)), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCivilFromEmailString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			str := getStringArg(args, 0)
			comment := ""
			if mm := emailCommentPattern.FindStringSubmatch(str); len(mm) > 1 {
				comment = mm[1]
			}
			stripped := strings.TrimSpace(emailCommentPattern.ReplaceAllString(str, ""))
			t, err := parseEmailDate(stripped)
			if err != nil {
				return newFormatError(fmt.Sprintf("invalid email date-time string: %s", str)), nil
			}
			result := buildCivilWithZone(ctx.TypeCtx, t, true, true)
			if comment != "" {
				result.Put(ctx.TypeCtx, "timeAbbrev", comment)
			}
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCivilToEmailString",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			m, _ := args[0].(*values.Map)
			zoneHandling := getStringArg(args, 1)

			t, offsetSecs, err := civilToGoTime(m)
			if err != nil {
				return newFormatError(err.Error()), nil
			}

			sign := "+"
			absSecs := offsetSecs
			if absSecs < 0 {
				sign = "-"
				absSecs = -absSecs
			}
			offsetStr := fmt.Sprintf("%s%02d%02d", sign, absSecs/3600, (absSecs%3600)/60)
			result := formatEmailDate(t, offsetStr)
			if zoneHandling == "ZONE_OFFSET_WITH_TIME_ABBREV_COMMENT" {
				if abbrev := mapString(m, "timeAbbrev"); abbrev != "" {
					result += " (" + abbrev + ")"
				}
			}
			return result, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "externCivilAddDuration",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			civil, _ := args[0].(*values.Map)
			duration, _ := args[1].(*values.Map)

			t, _, err := civilToGoTime(civil)
			if err != nil {
				return newFormatError(err.Error()), nil
			}

			duYear := int(mapInt(duration, "years"))
			duMonth := int(mapInt(duration, "months"))
			duWeek := int(mapInt(duration, "weeks"))
			duDay := int(mapInt(duration, "days")) + duWeek*7
			duHour := int(mapInt(duration, "hours"))
			duMinute := int(mapInt(duration, "minutes"))
			duSecond := mapDecimal(duration, "seconds")

			t = t.AddDate(duYear, duMonth, duDay)
			t = t.Add(time.Duration(duHour)*time.Hour +
				time.Duration(duMinute)*time.Minute +
				time.Duration(decimalToNanos(duSecond)))

			return buildCivil(ctx.TypeCtx, t), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "TimeZone.initNative",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			var zoneIdStr string
			isString := false
			if len(args) > 1 && args[1] != nil {
				if s, ok := args[1].(string); ok {
					zoneIdStr = s
					isString = true
				}
			}

			var loc *time.Location
			if !isString {
				loc = time.Local
				zoneIdStr = time.Local.String()
			} else {
				switch {
				case zoneIdStr == "Z" || zoneIdStr == "UTC":
					loc = time.UTC
				case isNumericOffset(zoneIdStr):
					offsetSecs, err := parseNumericOffset(zoneIdStr)
					if err != nil {
						return newFormatError(fmt.Sprintf("invalid time zone offset: %s", zoneIdStr)), nil
					}
					loc = time.FixedZone(zoneIdStr, offsetSecs)
				default:
					var err error
					loc, err = time.LoadLocation(zoneIdStr)
					if err != nil {
						return newFormatError(fmt.Sprintf("invalid time zone ID: %s", zoneIdStr)), nil
					}
				}
			}

			self.Put("$location", loc)
			self.Put("$zoneId", zoneIdStr)
			return nil, nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "TimeZone.fixedOffset",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			zoneIdVal, _ := self.Get("$zoneId")
			zoneId, _ := zoneIdVal.(string)

			switch {
			case isNumericOffset(zoneId):
				offsetSecs, err := parseNumericOffset(zoneId)
				if err != nil {
					return nil, nil
				}
				return buildZoneOffset(ctx.TypeCtx, offsetSecs), nil
			case zoneId == "Z" || zoneId == "UTC":
				return buildZoneOffset(ctx.TypeCtx, 0), nil
			default:
				return nil, nil
			}
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "TimeZone.utcFromCivil",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			civil, _ := args[1].(*values.Map)

			if civil == nil {
				return newFormatError("civil value is nil"), nil
			}
			if _, hasAbbrev := civil.Get("timeAbbrev"); !hasAbbrev {
				return newFormatError("Abbreviation for the local time is required for the conversion"), nil
			}
			loc := getStoredLocation(self)
			t, err := civilMapToGoTimeInLocation(civil, loc)
			if err != nil {
				return newFormatError(err.Error()), nil
			}
			return goTimeToUtc(utcTy, ctx.TypeCtx, t), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "TimeZone.utcToCivil",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			utc, _ := args[1].(*values.List)

			loc := getStoredLocation(self)
			t := utcToGoTime(utc).In(loc)
			return buildCivilWithZone(ctx.TypeCtx, t, true, true), nil
		})

	runtime.RegisterExternFunction(rt, orgName, moduleName, "TimeZone.civilAddDuration",
		func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
			self, _ := args[0].(*values.Object)
			civil, _ := args[1].(*values.Map)
			duration, _ := args[2].(*values.Map)

			loc := getStoredLocation(self)
			t, err := civilMapToGoTimeInLocation(civil, loc)
			if err != nil {
				return newFormatError(err.Error()), nil
			}

			duYear := int(mapInt(duration, "years"))
			duMonth := int(mapInt(duration, "months"))
			duWeek := int(mapInt(duration, "weeks"))
			duDay := int(mapInt(duration, "days")) + duWeek*7
			duHour := int(mapInt(duration, "hours"))
			duMinute := int(mapInt(duration, "minutes"))
			duSecond := mapDecimal(duration, "seconds")

			t = t.AddDate(duYear, duMonth, duDay)
			t = t.Add(time.Duration(duHour)*time.Hour +
				time.Duration(duMinute)*time.Minute +
				time.Duration(decimalToNanos(duSecond)))

			return buildCivilWithZone(ctx.TypeCtx, t.In(loc), true, true), nil
		})
}

// isNumericOffset reports whether s is exactly "+HHMM" or "+HH:MM" (and the '-' variants).
func isNumericOffset(s string) bool {
	n := len(s)
	if n != 5 && n != 6 {
		return false
	}
	if s[0] != '+' && s[0] != '-' {
		return false
	}
	if s[1] < '0' || s[1] > '9' || s[2] < '0' || s[2] > '9' {
		return false
	}
	if n == 6 {
		return s[3] == ':' && s[4] >= '0' && s[4] <= '9' && s[5] >= '0' && s[5] <= '9'
	}
	return s[3] >= '0' && s[3] <= '9' && s[4] >= '0' && s[4] <= '9'
}

// parseNumericOffset parses "+HHMM" or "+HH:MM" (and '-' variants) to seconds east of UTC.
// Returns an error for out-of-range or malformed inputs.
func parseNumericOffset(s string) (int, error) {
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	rest := s[1:]
	var hStr, mStr string
	if len(rest) == 5 { // HH:MM
		hStr, mStr = rest[:2], rest[3:]
	} else { // HHMM
		hStr, mStr = rest[:2], rest[2:]
	}
	h, err := strconv.Atoi(hStr)
	if err != nil {
		return 0, fmt.Errorf("invalid offset %q", s)
	}
	m, err := strconv.Atoi(mStr)
	if err != nil {
		return 0, fmt.Errorf("invalid offset %q", s)
	}
	if h > 23 || m > 59 {
		return 0, fmt.Errorf("offset out of range in %q", s)
	}
	return sign * (h*3600 + m*60), nil
}

func init() {
	runtime.RegisterModuleInitializer(initTimeModule)
}
