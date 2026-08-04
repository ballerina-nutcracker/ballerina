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

package semtypes

type ListDefinition struct {
	rec     *recAtom
	semType SemType
}

type listDefinitionOptions struct {
	fixedLength int
	rest        SemType
	mutability  CellMutability
}

type ListDefinitionOption func(*listDefinitionOptions)

func ListFixedLength(length int) ListDefinitionOption {
	return func(options *listDefinitionOptions) {
		options.fixedLength = length
	}
}

func ListRest(rest SemType) ListDefinitionOption {
	return func(options *listDefinitionOptions) {
		options.rest = rest
	}
}

func ListMutability(mutability CellMutability) ListDefinitionOption {
	return func(options *listDefinitionOptions) {
		options.mutability = mutability
	}
}

var _ Definition = &ListDefinition{}

func NewListDefinition() ListDefinition {
	this := ListDefinition{}
	this.rec = nil
	this.semType = SemType{}
	// Default field initializations

	return this
}

func (l *ListDefinition) GetSemType(env Env) SemType {
	s := l.semType
	if IsZero(s) {
		rec := env.recListAtom()
		l.rec = &rec
		return l.createSemType(env, &rec)
	} else {
		return s
	}
}

func (l *ListDefinition) Define(env Env, initial []SemType, options ...ListDefinitionOption) SemType {
	opts := listDefinitionOptions{
		fixedLength: len(initial),
		rest:        Never,
		mutability:  CellMutabilityLimited,
	}
	for _, option := range options {
		option(&opts)
	}

	initialCells := make([]SemType, 0, len(initial))
	for _, member := range initial {
		initialCells = append(initialCells, cellContainingWithEnvSemTypeCellMutability(env, member, opts.mutability))
	}
	restMut := opts.mutability
	if IsNever(opts.rest) {
		restMut = CellMutabilityNone
	}
	restCell := cellContainingWithEnvSemTypeCellMutability(env, Union(opts.rest, Undef), restMut)
	return l.defineFromCells(env, initialCells, opts.fixedLength, restCell)
}

func (l *ListDefinition) defineFromCells(env Env, initial []SemType, fixedLength int, rest SemType) SemType {
	members := l.fixedLengthNormalize(fixedLengthArrayFrom(initial, fixedLength))
	atomicType := listAtomicTypeFrom(members, rest)
	var atom atom
	rec := l.rec
	if rec != nil {
		atom = rec
		env.setRecListAtomType(*rec, &atomicType)
	} else {
		atom = env.listAtom(&atomicType)
	}
	return l.createSemType(env, atom)
}

func (l *ListDefinition) fixedLengthNormalize(array fixedLengthArray) fixedLengthArray {
	initial := array.initial
	i := (len(initial) - 1)
	if i <= 0 {
		return array
	}
	last := initial[i]
	i = (i - 1)
	for i >= 0 {
		if !sameComplexSemType(last, initial[i]) {
			break
		}
		i = (i - 1)
	}
	return fixedLengthArrayFrom(initial[:i+2], array.FixedLength)
}

func (l *ListDefinition) createSemType(env Env, atom atom) SemType {
	bdd := bddAtom(atom)
	semType := getBasicSubtype(btList, bdd)
	l.semType = semType
	return semType
}
