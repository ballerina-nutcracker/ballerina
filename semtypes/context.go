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

// Context is an opaque pointer to the thread local context for type system. As such it should never be used
// concurrently in multiple type check operations at the same time. That being said it is highly advised to
// reuse the Context given it acts as a cache for many type operations.
type Context = *context

type context struct {
	_memoStack    []*bddMemo
	_conjunctions []conjunction

	_cloneableMemo      SemType
	_orderedMemo        SemType
	_isolatedObjectMemo SemType
	_serviceObjectMemo  SemType
	_clientObjectMemo   SemType
	_isolatedFnMemo     SemType
	_isolatedMemo       SemType
	_iterableMemo       SemType

	_env                   Env
	_listMemo              map[bddKey]*bddMemo
	_mappingMemo           map[bddKey]*bddMemo
	_functionMemo          map[bddKey]*bddMemo
	_comparableMemo        map[comparableMemoKey]*comparableMemo
	_fillerMemo            map[atomicType]Filler
	_streamImplementorMemo map[streamImplementorMemoKey]SemType
	_listenerMemo          map[listenerMemoKey]SemType
	_semtypeInterner       *SemtypeInterner
}

type streamImplementorMemoKey struct {
	valueTy      InternHandle
	completionTy InternHandle
}

type listenerMemoKey struct {
	t InternHandle
	a InternHandle
}

type comparableMemo struct {
	comparable bool
}

type comparableMemoKey struct {
	key1 bddKey
	key2 bddKey
}

func (c *context) pushToMemoStack(m *bddMemo) {
	c._memoStack = append(c._memoStack, m)
}

func (c *context) getMemoStackDepth() int {
	return len(c._memoStack)
}

func (c *context) getMemoStack(i int) *bddMemo {
	return c._memoStack[i]
}

func (c *context) popFromMemoStack() *bddMemo {
	lastIndex := len(c._memoStack) - 1
	memo := c._memoStack[lastIndex]
	c._memoStack = c._memoStack[:lastIndex]
	return memo
}

func (c *context) Env() Env {
	return c._env
}

func (c *context) cloneableMemo() SemType {
	return c._cloneableMemo
}

func (c *context) setCloneableMemo(t SemType) {
	c._cloneableMemo = t
}

func (c *context) orderedMemo() SemType {
	return c._orderedMemo
}

func (c *context) setOrderedMemo(t SemType) {
	c._orderedMemo = t
}

func (c *context) isolatedObjectMemo() SemType {
	return c._isolatedObjectMemo
}

func (c *context) setIsolatedObjectMemo(t SemType) {
	c._isolatedObjectMemo = t
}

func (c *context) serviceObjectMemo() SemType {
	return c._serviceObjectMemo
}

func (c *context) setServiceObjectMemo(t SemType) {
	c._serviceObjectMemo = t
}

func (c *context) clientObjectMemo() SemType {
	return c._clientObjectMemo
}

func (c *context) setClientObjectMemo(t SemType) {
	c._clientObjectMemo = t
}

func (c *context) iterableMemo() SemType {
	return c._iterableMemo
}

func (c *context) setIterableMemo(t SemType) {
	c._iterableMemo = t
}

func (c *context) mappingMemo() map[bddKey]*bddMemo {
	return c._mappingMemo
}

func (c *context) functionMemo() map[bddKey]*bddMemo {
	return c._functionMemo
}

func (c *context) listMemo() map[bddKey]*bddMemo {
	return c._listMemo
}

func (c *context) FunctionAtomType(atom atom) *functionAtomicType {
	return c._env.functionAtomType(atom)
}

func (c *context) ListAtomType(atom atom) *ListAtomicType {
	return c._env.listAtomType(atom)
}

func (c *context) MappingAtomType(atom atom) *MappingAtomicType {
	return c._env.mappingAtomType(atom)
}

func (c *context) pushConjunction(atom atom, next conjunctionHandle) conjunctionHandle {
	idx := conjunctionHandle(len(c._conjunctions) + 1)
	c._conjunctions = append(c._conjunctions, conjunction{atom: atom, Next: next})
	return idx
}

func (c *context) conjunctionAtom(h conjunctionHandle) atom {
	return c._conjunctions[h-1].atom
}

func (c *context) conjunctionNext(h conjunctionHandle) conjunctionHandle {
	return c._conjunctions[h-1].Next
}

func (c *context) conjunctionStackDepth() int32 {
	return int32(len(c._conjunctions))
}

func (c *context) resetConjunctionStack(depth int32) {
	c._conjunctions = c._conjunctions[:depth]
}

func ContextFrom(env Env) Context {
	return &context{
		_env:                   env,
		_listMemo:              make(map[bddKey]*bddMemo),
		_mappingMemo:           make(map[bddKey]*bddMemo),
		_functionMemo:          make(map[bddKey]*bddMemo),
		_comparableMemo:        make(map[comparableMemoKey]*comparableMemo),
		_fillerMemo:            make(map[atomicType]Filler),
		_streamImplementorMemo: make(map[streamImplementorMemoKey]SemType),
		_listenerMemo:          make(map[listenerMemoKey]SemType),
		_semtypeInterner:       NewSemtypeInterner(),
		_conjunctions:          make([]conjunction, 0, 64),
	}
}

func (c *context) streamImplementorMemo(valueTy, completionTy SemType) (SemType, bool) {
	key := streamImplementorMemoKey{valueTy: c._semtypeInterner.Intern(valueTy), completionTy: c._semtypeInterner.Intern(completionTy)}
	t, ok := c._streamImplementorMemo[key]
	return t, ok
}

func (c *context) setStreamImplementorMemo(valueTy, completionTy, t SemType) {
	key := streamImplementorMemoKey{valueTy: c._semtypeInterner.Intern(valueTy), completionTy: c._semtypeInterner.Intern(completionTy)}
	c._streamImplementorMemo[key] = t
}

func (c *context) listenerMemo(t, a SemType) (SemType, bool) {
	key := listenerMemoKey{t: c._semtypeInterner.Intern(t), a: c._semtypeInterner.Intern(a)}
	ty, ok := c._listenerMemo[key]
	return ty, ok
}

func (c *context) setListenerMemo(t, a, listenerTy SemType) {
	key := listenerMemoKey{t: c._semtypeInterner.Intern(t), a: c._semtypeInterner.Intern(a)}
	c._listenerMemo[key] = listenerTy
}

func (c *context) comparableMemo(b1, b2 Bdd) *comparableMemo {
	return c._comparableMemo[comparableMemoKeyOf(b1, b2)]
}

func (c *context) setComparableMemo(b1, b2 Bdd, memo *comparableMemo) {
	c._comparableMemo[comparableMemoKeyOf(b1, b2)] = memo
}

func comparableMemoKeyOf(b1, b2 Bdd) comparableMemoKey {
	return comparableMemoKey{key1: b1.canonicalKey(), key2: b2.canonicalKey()}
}
