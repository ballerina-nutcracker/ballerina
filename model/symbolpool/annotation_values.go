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

package symbolpool

import (
	"bytes"
	"fmt"
	"sort"

	"ballerina/decimal"
	"ballerina/semtypes"
	"ballerina/values"
)

type annotationValueTag int8

const (
	annotationValueTagInt annotationValueTag = iota + 1
	annotationValueTagByte
	annotationValueTagFloat
	annotationValueTagDecimal
	annotationValueTagString
	annotationValueTagBoolean
	annotationValueTagNil
	annotationValueTagMap
	annotationValueTagList
	annotationValueTagTypedesc
	annotationValueTagRuntimeRef
)

func (sw *symbolWriter) writeAnnotationValues(buf *bytes.Buffer, annotations values.AnnotationValues) error {
	keys := make([]string, 0, len(annotations))
	for key := range annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if err := write(buf, int64(len(keys))); err != nil {
		return err
	}
	for _, key := range keys {
		if err := sw.writeStringCP(buf, key); err != nil {
			return err
		}
		if err := sw.writeAnnotationValue(buf, annotations[key]); err != nil {
			return err
		}
	}
	return nil
}

func (sw *symbolWriter) writeAnnotationValue(buf *bytes.Buffer, value values.AnnotationValue) error {
	tag, err := inferAnnotationValueTag(value)
	if err != nil {
		return err
	}
	if err := write(buf, int8(tag)); err != nil {
		return err
	}

	switch tag {
	case annotationValueTagInt:
		var val int64
		switch v := value.(type) {
		case int:
			val = int64(v)
		case int64:
			val = v
		case int32:
			val = int64(v)
		case int16:
			val = int64(v)
		case int8:
			val = int64(v)
		}
		return write(buf, val)
	case annotationValueTagByte:
		return write(buf, value.(byte))
	case annotationValueTagFloat:
		var val float64
		switch v := value.(type) {
		case float64:
			val = v
		case float32:
			val = float64(v)
		}
		return write(buf, val)
	case annotationValueTagDecimal:
		return sw.writeStringCP(buf, value.(*decimal.Decimal).String())
	case annotationValueTagString:
		switch v := value.(type) {
		case string:
			return sw.writeStringCP(buf, v)
		case *string:
			if v == nil {
				return sw.writeStringCP(buf, "")
			}
			return sw.writeStringCP(buf, *v)
		}
	case annotationValueTagBoolean:
		return write(buf, value.(bool))
	case annotationValueTagNil:
		return nil
	case annotationValueTagMap:
		m := value.(*values.Map)
		if err := sw.writeType(buf, m.Type); err != nil {
			return err
		}
		if err := write(buf, m.IsReadonly()); err != nil {
			return err
		}
		keys := m.Keys()
		if err := write(buf, int64(len(keys))); err != nil {
			return err
		}
		for _, key := range keys {
			if err := sw.writeStringCP(buf, key); err != nil {
				return err
			}
			entry, _ := m.Get(key)
			if err := sw.writeAnnotationValue(buf, entry); err != nil {
				return err
			}
		}
		return nil
	case annotationValueTagList:
		list := value.(*values.List)
		if err := sw.writeType(buf, list.Type); err != nil {
			return err
		}
		if err := write(buf, list.IsReadonly()); err != nil {
			return err
		}
		if err := write(buf, int64(list.Len())); err != nil {
			return err
		}
		for i := 0; i < list.Len(); i++ {
			if err := sw.writeAnnotationValue(buf, list.Get(i)); err != nil {
				return err
			}
		}
		return nil
	case annotationValueTagTypedesc:
		td := value.(*values.TypeDesc)
		if err := sw.writeType(buf, td.Type); err != nil {
			return err
		}
		return sw.writeAnnotationValues(buf, td.Annotations)
	case annotationValueTagRuntimeRef:
		ref := value.(*values.RuntimeAnnotationValueRef)
		if err := sw.writeStringCP(buf, ref.Organization); err != nil {
			return err
		}
		if err := sw.writeStringCP(buf, ref.Module); err != nil {
			return err
		}
		return sw.writeStringCP(buf, ref.GlobalName)
	}
	return fmt.Errorf("unsupported annotation value tag: %d", tag)
}

func inferAnnotationValueTag(value values.AnnotationValue) (annotationValueTag, error) {
	switch value.(type) {
	case int, int64, int32, int16, int8:
		return annotationValueTagInt, nil
	case byte:
		return annotationValueTagByte, nil
	case float64, float32:
		return annotationValueTagFloat, nil
	case *decimal.Decimal:
		return annotationValueTagDecimal, nil
	case string, *string:
		return annotationValueTagString, nil
	case bool:
		return annotationValueTagBoolean, nil
	case nil:
		return annotationValueTagNil, nil
	case *values.Map:
		return annotationValueTagMap, nil
	case *values.List:
		return annotationValueTagList, nil
	case *values.TypeDesc:
		return annotationValueTagTypedesc, nil
	case *values.RuntimeAnnotationValueRef:
		return annotationValueTagRuntimeRef, nil
	default:
		return 0, fmt.Errorf("unsupported annotation value type: %T", value)
	}
}

func (sr *symbolReader) readAnnotationValues() values.AnnotationValues {
	var count int64
	read(sr.r, &count)
	annotations := values.NewAnnotationValues()
	for i := int64(0); i < count; i++ {
		key := sr.readStringCP()
		annotations[key] = sr.readAnnotationValue()
	}
	return annotations
}

func (sr *symbolReader) readAnnotationValue() values.AnnotationValue {
	var rawTag int8
	read(sr.r, &rawTag)
	tag := annotationValueTag(rawTag)

	switch tag {
	case annotationValueTagInt:
		var val int64
		read(sr.r, &val)
		return val
	case annotationValueTagByte:
		var val byte
		read(sr.r, &val)
		return val
	case annotationValueTagFloat:
		var val float64
		read(sr.r, &val)
		return val
	case annotationValueTagDecimal:
		str := sr.readStringCP()
		val, err := decimal.FromString(str)
		if err != nil {
			panic(fmt.Sprintf("invalid decimal annotation value %q: %v", str, err))
		}
		return val
	case annotationValueTagString:
		return sr.readStringCP()
	case annotationValueTagBoolean:
		var val bool
		read(sr.r, &val)
		return val
	case annotationValueTagNil:
		return nil
	case annotationValueTagMap:
		ty := sr.readType()
		var isReadonly bool
		read(sr.r, &isReadonly)
		var count int64
		read(sr.r, &count)
		entries := make([]values.MapEntry, 0, count)
		for i := int64(0); i < count; i++ {
			key := sr.readStringCP()
			entries = append(entries, values.MapEntry{Key: key, Value: sr.readAnnotationValue()})
		}
		tyCtx := semtypes.TypeCheckContext(sr.env.GetTypeEnv())
		atomic := semtypes.ToMappingAtomicType(tyCtx, ty)
		if atomic == nil {
			panic("annotation map type is not atomic")
		}
		return values.NewMap(ty, atomic, isReadonly, entries)
	case annotationValueTagList:
		ty := sr.readType()
		var isReadonly bool
		read(sr.r, &isReadonly)
		var count int64
		read(sr.r, &count)
		initial := make([]values.BalValue, count)
		for i := int64(0); i < count; i++ {
			initial[i] = sr.readAnnotationValue()
		}
		tyCtx := semtypes.TypeCheckContext(sr.env.GetTypeEnv())
		atomic := semtypes.ToListAtomicType(tyCtx, ty)
		if atomic == nil {
			panic("annotation list type is not atomic")
		}
		restFiller, _ := values.FillerFactoryFor(tyCtx, atomic.Rest())
		return values.NewList(ty, atomic, isReadonly, restFiller, int(count), initial)
	case annotationValueTagTypedesc:
		return values.NewTypeDesc(sr.readType(), sr.readAnnotationValues())
	case annotationValueTagRuntimeRef:
		return &values.RuntimeAnnotationValueRef{
			Organization: sr.readStringCP(),
			Module:       sr.readStringCP(),
			GlobalName:   sr.readStringCP(),
		}
	default:
		panic(fmt.Sprintf("unknown annotation value tag: %d", tag))
	}
}
