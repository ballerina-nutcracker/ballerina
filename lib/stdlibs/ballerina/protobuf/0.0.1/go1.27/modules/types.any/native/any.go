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
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "protobuf.types.any"

	typeURLPrefix = "type.googleapis.com/google.protobuf."
)

// nanosPerSec is 1,000,000,000 as a decimal constant used for ns <-> seconds conversion.
var nanosPerSec = decimal.FromInt64(1_000_000_000)

// decimalZero is the zero decimal, used for sign tests.
var decimalZero = decimal.FromInt64(0)

// anyTypes holds the semtypes built once at module init, reused across calls.
type anyTypes struct {
	byteArrTy         semtypes.SemType
	utcTy             semtypes.SemType
	utcAtomic         *semtypes.ListAtomicType
	anydataTy         semtypes.SemType
	anydataMapTy      semtypes.SemType
	anydataMapAtomic  *semtypes.MappingAtomicType
	anydataListTy     semtypes.SemType
	anydataListAtomic *semtypes.ListAtomicType
}

var types anyTypes

func initAnyModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	tc := semtypes.TypeCheckContext(env)
	anydataTy := semtypes.CreateAnydata(tc)

	byteArrBld := semtypes.NewListDefinition()
	byteArrTy := byteArrBld.Define(env, nil, semtypes.ListRest(semtypes.Byte))

	utcBld := semtypes.NewListDefinition()
	utcTy := utcBld.Define(env, []semtypes.SemType{semtypes.Int, semtypes.Decimal},
		semtypes.ListMutability(semtypes.CellMutabilityNone))

	mapBld := semtypes.NewMappingDefinition()
	anydataMapTy := mapBld.Define(env, nil, anydataTy)

	listBld := semtypes.NewListDefinition()
	anydataListTy := listBld.Define(env, nil, semtypes.ListRest(anydataTy))

	types = anyTypes{
		byteArrTy:         byteArrTy,
		utcTy:             utcTy,
		utcAtomic:         semtypes.ToListAtomicType(env, utcTy),
		anydataTy:         anydataTy,
		anydataMapTy:      anydataMapTy,
		anydataMapAtomic:  semtypes.ToMappingAtomicType(tc, anydataMapTy),
		anydataListTy:     anydataListTy,
		anydataListAtomic: semtypes.ToListAtomicType(env, anydataListTy),
	}

	runtime.RegisterExternFunction(rt, orgName, moduleName, "serializeToHex", serializeToHexExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "unpack", unpackExtern)
	runtime.RegisterExternFunction(rt, orgName, moduleName, "registerProtoTypesAnyModule", registerModuleExtern)
}

func init() {
	runtime.RegisterModuleInitializer(initAnyModule)
}

func registerModuleExtern(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
	return nil, nil
}

// callModuleErrorConstructor invokes a private any.bal helper (e.g. newAnyError,
// newTypeMismatchError) that constructs a value of a real distinct error type declared
// in this module, so the error satisfies `is`-checks against that type. Native code
// cannot construct a distinct-typed value directly, since the distinct identity lives
// in the module's own compiled semtype environment.
//
// TODO: remove this indirection (and the newAnyError/newTypeMismatchError .bal helpers
// it calls) once the runtime exposes an API for native code to construct a value of a
// specific distinct error type directly, without calling back into Ballerina.
func callModuleErrorConstructor(ctx *extern.Context, funcName, message string) (values.BalValue, error) {
	handle, ok := ctx.LookupFunction(orgName, moduleName, funcName)
	if !ok {
		return nil, fmt.Errorf("protobuf.types.any: internal helper function %q not found", funcName)
	}
	return ctx.InvokeFunction(handle, []values.BalValue{message})
}

func newAnyError(ctx *extern.Context, message string) (values.BalValue, error) {
	return callModuleErrorConstructor(ctx, "newAnyError", message)
}

func newTypeMismatchError(ctx *extern.Context, message string) (values.BalValue, error) {
	return callModuleErrorConstructor(ctx, "newTypeMismatchError", message)
}

// serializeToHexExtern serializes a ValueType message (already dispatched to a
// google.protobuf.* well-known-type suffix by any.bal's typeUrlSuffix) into the
// hex-encoded protobuf wire bytes stored in Any.value.
func serializeToHexExtern(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	message := args[0]
	suffix, _ := args[1].(string)

	msg, err := valueToWireMessage(message, suffix)
	if err != nil {
		return newAnyError(ctx, "failed to pack value as google.protobuf."+suffix+": "+err.Error())
	}

	raw, err := proto.Marshal(msg)
	if err != nil {
		return newAnyError(ctx, "failed to serialize google.protobuf."+suffix+" value: "+err.Error())
	}
	// Upper-case to match jBallerina's Utils.bytesToHex, which always emits upper-case
	// digits; Any.value is public, so user code can compare it across implementations.
	// Decoding stays case-insensitive on both sides.
	return strings.ToUpper(hex.EncodeToString(raw)), nil
}

func valueToWireMessage(message values.BalValue, suffix string) (proto.Message, error) {
	switch suffix {
	case "Empty":
		return &emptypb.Empty{}, nil
	case "BoolValue":
		v, _ := message.(bool)
		return wrapperspb.Bool(v), nil
	case "Int64Value":
		v, _ := message.(int64)
		return wrapperspb.Int64(v), nil
	case "FloatValue":
		v, _ := message.(float64)
		return wrapperspb.Float(float32(v)), nil
	case "StringValue":
		v, _ := message.(string)
		return wrapperspb.String(v), nil
	case "BytesValue":
		list, _ := message.(*values.List)
		return wrapperspb.Bytes(list.ToByteSlice()), nil
	case "Timestamp":
		list, _ := message.(*values.List)
		return utcListToTimestamp(list)
	case "Duration":
		dec, _ := message.(*decimal.Decimal)
		return secondsToDuration(dec)
	case "Struct":
		m, _ := message.(*values.Map)
		goMap, err := balAnydataToGo(m)
		if err != nil {
			return nil, err
		}
		asMap, _ := goMap.(map[string]any)
		return structpb.NewStruct(asMap)
	default:
		return nil, fmt.Errorf("unsupported protobuf well-known type: %s", suffix)
	}
}

// unpackExtern decodes an Any value's hex-encoded wire bytes and, if the decoded
// well-known type matches the resolved target typedesc, returns the corresponding
// Ballerina value. Otherwise it returns a TypeMismatchError.
func unpackExtern(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
	anyMap, _ := args[0].(*values.Map)
	targetTypeDesc, _ := args[1].(*values.TypeDesc)

	typeURLVal, _ := anyMap.Get("typeUrl")
	typeURL, _ := typeURLVal.(string)
	hexVal, _ := anyMap.Get("value")
	hexStr, _ := hexVal.(string)

	raw, err := hex.DecodeString(hexStr)
	if err != nil {
		return newAnyError(ctx, "failed to decode Any value: "+err.Error())
	}

	suffix := strings.TrimPrefix(typeURL, typeURLPrefix)
	decoded, naturalTy, err := wireMessageToValue(ctx, suffix, raw)
	if err != nil {
		return newAnyError(ctx, "failed to unpack google.protobuf."+suffix+" value: "+err.Error())
	}
	if semtypes.IsZero(naturalTy) || !semtypes.IsSubtype(ctx.TypeCtx(), naturalTy, targetTypeDesc.Type) {
		return newTypeMismatchError(ctx, fmt.Sprintf("Type %s cannot unpack to %s",
			typeURL, semtypes.ToString(ctx.TypeCtx(), targetTypeDesc.Type)))
	}
	return decoded, nil
}

// wireMessageToValue decodes raw wire bytes for the given google.protobuf.* suffix into
// a Ballerina value and its natural (unpacked) semtype. A zero naturalTy (with a nil
// error) means the suffix is not a well-known type this module supports.
func wireMessageToValue(ctx *extern.Context, suffix string, raw []byte) (values.BalValue, semtypes.SemType, error) {
	env := ctx.TypeEnv()
	switch suffix {
	case "Empty":
		return nil, semtypes.Nil, nil
	case "BoolValue":
		msg := &wrapperspb.BoolValue{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return msg.GetValue(), semtypes.Boolean, nil
	case "Int64Value":
		msg := &wrapperspb.Int64Value{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return msg.GetValue(), semtypes.Int, nil
	case "FloatValue":
		msg := &wrapperspb.FloatValue{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return float64(msg.GetValue()), semtypes.Float, nil
	case "StringValue":
		msg := &wrapperspb.StringValue{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return msg.GetValue(), semtypes.String, nil
	case "BytesValue":
		msg := &wrapperspb.BytesValue{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return values.ByteSliceToList(types.byteArrTy, env, msg.GetValue()), types.byteArrTy, nil
	case "Timestamp":
		msg := &timestamppb.Timestamp{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return timestampToUtcList(msg), types.utcTy, nil
	case "Duration":
		msg := &durationpb.Duration{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return durationToSecondsDecimal(msg), semtypes.Decimal, nil
	case "Struct":
		msg := &structpb.Struct{}
		if err := proto.Unmarshal(raw, msg); err != nil {
			return nil, semtypes.SemType{}, err
		}
		return goToBalAnydata(msg.AsMap()), types.anydataMapTy, nil
	default:
		return nil, semtypes.SemType{}, nil
	}
}

// decimalToSecNano splits a Ballerina Seconds decimal into whole seconds and nanoseconds
// using exact decimal arithmetic throughout. The whole part truncates toward zero, matching
// jBallerina's BigDecimal.intValue(); routing it through float64 would round instead, and a
// Unix-timestamp-scale value carrying full nanosecond precision needs 19 significant digits
// where float64 offers ~15-17. Rounding there pushes the fraction past the second boundary
// and flips its sign, which google.protobuf.Duration forbids: for durations of a second or
// more, a non-zero nanos must carry the same sign as seconds. Returns an error if the whole
// seconds part does not fit in an int64, since google.protobuf.Duration/Timestamp both store
// seconds as int64.
func decimalToSecNano(sec *decimal.Decimal) (int64, int32, error) {
	intSecDec := truncateTowardZero(sec)
	intSec, ok, _ := intSecDec.Int64()
	if !ok {
		return 0, 0, fmt.Errorf("seconds value %s does not fit in an int64", sec.String())
	}
	fracDec, _ := sec.Sub(intSecDec)
	nanosDec, _ := fracDec.Mul(nanosPerSec)
	nanosInt, _, _ := nanosDec.Int64()
	return intSec, int32(nanosInt), nil
}

// truncateTowardZero discards sec's fractional part, so the result never exceeds sec
// in magnitude and keeps sec's sign. Floor/Ceiling cannot fail here: they only report an
// error on overflow, and a decimal128 value that is already integral (as any value at the
// extremes of its representable exponent range necessarily is) never needs to grow past
// that range to round to itself.
func truncateTowardZero(sec *decimal.Decimal) *decimal.Decimal {
	if sec.Cmp(decimalZero) < 0 {
		out, _ := sec.Ceiling()
		return out
	}
	out, _ := sec.Floor()
	return out
}

// secNanoToDecimal combines whole seconds and nanoseconds into a Ballerina Seconds decimal.
func secNanoToDecimal(sec int64, nanos int32) *decimal.Decimal {
	secDec := decimal.FromInt64(sec)
	nanosDec := decimal.FromInt64(int64(nanos))
	frac, _ := nanosDec.Quo(nanosPerSec)
	sum, _ := secDec.Add(frac)
	return sum
}

func secondsToDuration(sec *decimal.Decimal) (*durationpb.Duration, error) {
	s, n, err := decimalToSecNano(sec)
	if err != nil {
		return nil, err
	}
	return &durationpb.Duration{Seconds: s, Nanos: n}, nil
}

func durationToSecondsDecimal(d *durationpb.Duration) *decimal.Decimal {
	return secNanoToDecimal(d.GetSeconds(), d.GetNanos())
}

func utcListToTimestamp(list *values.List) (*timestamppb.Timestamp, error) {
	epochSec, _ := list.Get(0).(int64)
	frac, _ := list.Get(1).(*decimal.Decimal)
	_, nanos, err := decimalToSecNano(frac)
	if err != nil {
		return nil, err
	}
	return &timestamppb.Timestamp{Seconds: epochSec, Nanos: nanos}, nil
}

func timestampToUtcList(ts *timestamppb.Timestamp) *values.List {
	frac := secNanoToDecimal(0, ts.GetNanos())
	items := []values.BalValue{ts.GetSeconds(), frac}
	return values.NewList(types.utcTy, types.utcAtomic, true, nil, 2, items)
}

// balAnydataToGo converts a Ballerina anydata value into the plain Go value shape
// accepted by structpb.NewStruct: nil, bool, float64, string, map[string]any, []any.
// Integers and decimals widen to float64, since google.protobuf.Struct only supports
// double-precision numbers.
func balAnydataToGo(v values.BalValue) (any, error) {
	switch val := v.(type) {
	case nil:
		return nil, nil
	case bool:
		return val, nil
	case int64:
		return float64(val), nil
	case float64:
		return val, nil
	case *decimal.Decimal:
		return val.Float64(), nil
	case string:
		return val, nil
	case *values.Map:
		m := make(map[string]any, val.Len())
		for _, key := range val.Keys() {
			fieldVal, _ := val.Get(key)
			goVal, err := balAnydataToGo(fieldVal)
			if err != nil {
				return nil, err
			}
			m[key] = goVal
		}
		return m, nil
	case *values.List:
		items := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			goVal, err := balAnydataToGo(val.Get(i))
			if err != nil {
				return nil, err
			}
			items[i] = goVal
		}
		return items, nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be represented in a google.protobuf.Struct", v)
	}
}

// goToBalAnydata converts a decoded structpb value (as produced by Struct.AsMap()) into
// a Ballerina anydata value. Map keys are sorted for deterministic output, since Go's
// map iteration order is randomized and the wire format does not preserve field order.
func goToBalAnydata(v any) values.BalValue {
	switch val := v.(type) {
	case nil:
		return nil
	case bool:
		return val
	case float64:
		return val
	case string:
		return val
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		entries := make([]values.MapEntry, 0, len(keys))
		for _, k := range keys {
			entries = append(entries, values.MapEntry{Key: k, Value: goToBalAnydata(val[k])})
		}
		return values.NewMap(types.anydataMapTy, types.anydataMapAtomic, false, entries)
	case []any:
		items := make([]values.BalValue, len(val))
		for i, iv := range val {
			items[i] = goToBalAnydata(iv)
		}
		return values.NewList(types.anydataListTy, types.anydataListAtomic, false, nil, len(items), items)
	default:
		return nil
	}
}
