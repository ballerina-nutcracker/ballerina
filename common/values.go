// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package common

import (
	"iter"
	"slices"
)

type Optional[T any] struct {
	value    T
	hasValue bool
}

func (o Optional[T]) IsPresent() bool {
	return o.hasValue
}

func (o Optional[T]) IsEmpty() bool {
	return !o.hasValue
}

func (o Optional[T]) Get() T {
	if !o.hasValue {
		panic("No value present")
	}
	return o.value
}

func OptionalOf[T any](value T) Optional[T] {
	return Optional[T]{
		value:    value,
		hasValue: true,
	}
}

func OptionalEmpty[T any]() Optional[T] {
	return Optional[T]{
		hasValue: false,
	}
}

func PointerEqualToValue[T comparable](ptr any, value T) bool {
	if val, ok := ptr.(T); ok {
		return val == value
	}
	return false
}

func ValueEqual[T comparable](v1, v2 T) bool {
	return v1 == v2
}

// TODO: think of a way to implement this without depending on comparable
type Set[T comparable] interface {
	Add(value T)
	Remove(value T)
	Contains(value T) bool
	Values() iter.Seq[T]
}

type (
	UnorderedSet[T comparable] struct {
		values map[T]bool
	}
	OrderedSet[T comparable] struct {
		// TODO: we need a more efficient data structure for this
		values []T
	}

	OrderedMap[K comparable, V any] struct {
		keys   []K
		values []V
	}
)

var (
	_ Set[any] = &UnorderedSet[any]{}
	_ Set[any] = &OrderedSet[any]{}
)

func (s *UnorderedSet[T]) Add(value T) {
	if s.values == nil {
		s.values = make(map[T]bool)
	}
	s.values[value] = true
}

func (s *UnorderedSet[T]) Remove(value T) {
	delete(s.values, value)
}

func (s *UnorderedSet[T]) Contains(value T) bool {
	return s.values[value]
}

func (s *OrderedSet[T]) Add(value T) {
	s.values = append(s.values, value)
}

func (s *OrderedSet[T]) Remove(value T) {
	for i, v := range s.values {
		if v == value {
			s.values = append(s.values[:i], s.values[i+1:]...)
			break
		}
	}
}

func (s *OrderedSet[T]) Contains(value T) bool {
	return slices.Contains(s.values, value)
}

func (s *UnorderedSet[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s.values == nil {
			return
		}
		for k := range s.values {
			if !yield(k) {
				break
			}
		}
	}
}

func (s *OrderedSet[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.values {
			if !yield(v) {
				break
			}
		}
	}
}

func (m *OrderedMap[K, V]) Add(key K, value V) {
	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
}

func (m *OrderedMap[K, V]) Remove(key K) {
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			m.values = append(m.values[:i], m.values[i+1:]...)
			break
		}
	}
}
