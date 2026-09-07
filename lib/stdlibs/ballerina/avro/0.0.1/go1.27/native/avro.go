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
	"github.com/linkedin/goavro/v2"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
	"github.com/ballerina-nutcracker/ballerina/values"
)

const (
	orgName    = "ballerina"
	moduleName = "avro"

	// schemaField holds the parsed schema on an avro:Schema object, as a
	// *parsedSchema. The name is unaddressable from Ballerina, matching
	// os:Process's $handle.
	schemaField = "$schema"

	// The two outer error messages jBallerina's Utils.createError uses. Schema
	// parsing has no jBallerina counterpart — it throws out of `new` there.
	schemaParsingError   = "Avro schema parsing error"
	serializationError   = "Avro serialization error"
	deserializationError = "Avro deserialization error"
)

// parsedSchema pairs goavro's own codec, which owns the wire format, with the
// shape tree encodeValue/decodeValue walk for leaf-type discrimination and
// union-branch naming — see schema.go's package doc for why both are needed.
type parsedSchema struct {
	codec *goavro.Codec
	shape *shape
}

// avroTypes holds the semtypes used when building decoded values. They are
// built once in initAvroModule (single-threaded) and captured by the handler
// closures.
type avroTypes struct {
	env           semtypes.Env
	byteArrTy     semtypes.SemType
	anydataMapTy  semtypes.SemType
	anydataListTy semtypes.SemType
	anydataMapAt  *semtypes.MappingAtomicType
	anydataListAt *semtypes.ListAtomicType
}

func initAvroModule(rt *runtime.Runtime) {
	env := rt.GetTypeEnv()
	anydataTy := semtypes.CreateAnydata(semtypes.ContextFrom(env))
	byteArrLd := semtypes.NewListDefinition()
	anydataListLd := semtypes.NewListDefinition()
	anydataMapMd := semtypes.NewMappingDefinition()
	types := &avroTypes{
		env:           env,
		byteArrTy:     byteArrLd.Define(env, nil, semtypes.ListRest(semtypes.Byte)),
		anydataListTy: anydataListLd.Define(env, nil, semtypes.ListRest(anydataTy)),
		anydataMapTy:  anydataMapMd.Define(env, nil, anydataTy),
	}
	types.anydataListAt = semtypes.ToListAtomicType(env, types.anydataListTy)
	types.anydataMapAt = semtypes.ToMappingAtomicType(semtypes.ContextFrom(env), types.anydataMapTy)

	runtime.RegisterExternFunction(rt, orgName, moduleName, "Schema.generateSchema", generateSchemaExtern())
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Schema.toAvro", toAvroExtern(types))
	runtime.RegisterExternFunction(rt, orgName, moduleName, "Schema.fromAvro", fromAvroExtern(types))
}

func init() {
	runtime.RegisterModuleInitializer(initAvroModule)
}

// newAvroError builds an avro:Error. The semtype is the generic error type —
// distinct error types cannot be constructed from Go, so avro:Error is
// declared as a plain alias in avro.bal.
func newAvroError(message string, cause values.BalValue) *values.Error {
	return values.NewError(semtypes.Error, message, cause, "Error", nil)
}

// avroErrorFrom mirrors jBallerina's Utils.createError: a fixed outer message
// with the underlying failure attached as the cause.
func avroErrorFrom(message string, err error) *values.Error {
	return newAvroError(message, values.NewErrorWithMessage(err.Error()))
}

// schemaOf recovers the schema stored by generateSchema. A miss means the
// schema never parsed but the caller kept the object alive, so it is reported
// as an error rather than panicking.
func schemaOf(self *values.Object) (*parsedSchema, *values.Error) {
	stored, ok := self.Get(schemaField)
	if !ok {
		return nil, newAvroError("Uninitialized Avro schema", nil)
	}
	schema, ok := stored.(*parsedSchema)
	if !ok {
		return nil, newAvroError("Uninitialized Avro schema", nil)
	}
	return schema, nil
}

// generateSchemaExtern parses definition twice, once into our own shape tree
// and once into goavro's codec, and only accepts the schema if both succeed
// — see schema.go's package doc for why the two must agree.
func generateSchemaExtern() extern.NativeFunc {
	return func(_ *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		definition, _ := args[1].(string)
		rootShape, strippedDefinition, err := parseSchemaShape(definition)
		if err != nil {
			return avroErrorFrom(schemaParsingError, err), nil
		}
		codec, err := goavro.NewCodec(strippedDefinition)
		if err != nil {
			return avroErrorFrom(schemaParsingError, err), nil
		}
		self.Put(schemaField, &parsedSchema{codec: codec, shape: rootShape})
		return nil, nil
	}
}

func toAvroExtern(types *avroTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		schema, schemaErr := schemaOf(self)
		if schemaErr != nil {
			return schemaErr, nil
		}
		native, err := encodeValue(ctx.TypeCtx(), schema.shape, args[1])
		if err != nil {
			return avroErrorFrom(serializationError, err), nil
		}
		encoded, err := schema.codec.BinaryFromNative(nil, native)
		if err != nil {
			return avroErrorFrom(serializationError, err), nil
		}
		return values.ByteSliceToList(types.byteArrTy, types.env, encoded), nil
	}
}

func fromAvroExtern(types *avroTypes) extern.NativeFunc {
	return func(ctx *extern.Context, args []values.BalValue) (values.BalValue, error) {
		self, _ := args[0].(*values.Object)
		schema, schemaErr := schemaOf(self)
		if schemaErr != nil {
			return schemaErr, nil
		}
		payload, _ := args[1].(*values.List)
		target, _ := args[2].(*values.TypeDesc)
		tc := ctx.TypeCtx()

		native, _, err := schema.codec.NativeFromBinary(payload.ToByteSlice())
		if err != nil {
			return avroErrorFrom(deserializationError, err), nil
		}
		decoded, err := decodeValue(tc, types, schema.shape, native)
		if err != nil {
			return avroErrorFrom(deserializationError, err), nil
		}
		return bindToTarget(tc, decoded, target.Type), nil
	}
}

// bindToTarget turns the generic value graph decodeValue produced into the type
// the call site asked for, the way http's client data binding does: skip the
// clone when the decoded value already fits, otherwise convert with the same
// routine as lang.value:fromJsonWithType so records, enums and tuples narrow.
func bindToTarget(tc semtypes.Context, decoded values.BalValue, target semtypes.SemType) values.BalValue {
	if semtypes.IsSubtype(tc, values.SemTypeForValue(decoded), target) {
		return decoded
	}
	bound, convErr := values.CloneWithType(tc, decoded, target)
	if convErr != nil {
		return newAvroError(deserializationError, convErr)
	}
	return bound
}
