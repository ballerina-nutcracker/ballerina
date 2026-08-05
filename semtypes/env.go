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

import (
	"math"
	"sync"
	"sync/atomic"
	"weak"
)

// Env is an opaque pointer to the type environment. All (potentially recursive) types are defined in a type environment.
// Performing type operations involving types defined in different type environments is an undefined behaviour. Therefore it
// is advisable to use the same type environment during the full execution of an interpreter process.
type Env = *env

func CreateTypeEnv() Env {
	env := &env{
		atomTable: make(map[atomicType]typeAtom),
	}
	fillRecAtoms(predefTypeEnv, &env.recListAtoms, predefTypeEnv.initializedRecListAtoms)
	fillRecAtoms(predefTypeEnv, &env.recMappingAtoms, predefTypeEnv.initializedRecMappingAtoms)
	// Treat every pre-existing rec atom slot as already populated. NOTE: this means we treat nil "padding" slots to have been filled
	env.populatedRecAtoms = int32(len(env.recListAtoms) + len(env.recMappingAtoms) + len(env.recFunctionAtoms))
	for _, each := range predefTypeEnv.initializedCellAtoms {
		env.cellAtom(each.atomicType)
	}
	for _, each := range predefTypeEnv.initializedListAtoms {
		env.listAtom(each.atomicType)
	}
	env.preallocatedTypeVals = newPreallocatedTypeVals(env)
	return env
}

type env struct {
	recListAtoms      []*ListAtomicType
	recListAtomsMutex sync.Mutex

	recMappingAtoms      []*MappingAtomicType
	recMappingAtomsMutex sync.Mutex

	recFunctionAtoms      []*functionAtomicType
	recFunctionAtomsMutex sync.Mutex

	// populatedRecAtoms counts the number of recursive atom slots  that have been filled
	// with a non-nil atomic type. The env is "ready" for emptiness checks
	// when this equals the total number of allocated rec atom slots.
	populatedRecAtoms int32

	distinctAtoms     int
	distinctAtomMutex sync.Mutex
	atomTableMutex    sync.Mutex
	atomTable         map[atomicType]typeAtom

	preallocatedTypeVals preallocatedTypeVals

	// frozen seals the persistent atom stores once the runtime begins
	// interpretation. After freeze, atomTable and the rec-atom slices are
	// read-only and any new atom is allocated in the ephemeral store instead.
	frozen          atomic.Bool
	ephemeralMu     sync.Mutex
	ephemeralSlots  []ephemeralSlot
	frozenAtomCount int
}

// ephemeralSlot holds a single runtime (ephemeral) type atom via a weak
// pointer so it can be garbage collected with its owning value/type. A slot's
// position gives the atom a stable index (frozenAtomCount + position); reusing a
// freed slot bumps gen so the atom's canonicalKey stays unique across reuse.
type ephemeralSlot struct {
	gen uint64
	ptr weak.Pointer[typeAtom]
}

// Freeze seals the persistent atom stores. It is called by the runtime state
// machine when it first transitions into the initializing state, before any
// interpretation runs. It is idempotent.
func (e *env) Freeze() {
	e.atomTableMutex.Lock()
	defer e.atomTableMutex.Unlock()
	if e.frozen.Load() {
		return
	}
	e.frozenAtomCount = len(e.atomTable)
	e.frozen.Store(true)
}

// assertNotFrozenForRecAtom guards recursive-atom allocation. Recursive types
// are stored permanently in the env and cannot be created at runtime, so any
// attempt to allocate a rec atom after freeze (e.g. Definition.GetSemType) is a
// bug.
func (e *env) assertNotFrozenForRecAtom() {
	if e.frozen.Load() {
		panic("cannot define recursive types at runtime")
	}
}

func (e *env) recListAtomCount() int {
	return len(e.recListAtoms)
}

func (e *env) recMappingAtomCount() int {
	return len(e.recMappingAtoms)
}

func (e *env) recFunctionAtomCount() int {
	return len(e.recFunctionAtoms)
}

func (e *env) distinctAtomCount() int {
	e.distinctAtomMutex.Lock()
	defer e.distinctAtomMutex.Unlock()
	return e.distinctAtoms
}

func (e *env) distinctAtomCountGetAndIncrement() int {
	e.distinctAtomMutex.Lock()
	defer e.distinctAtomMutex.Unlock()
	e.distinctAtoms++
	return e.distinctAtoms
}

func (e *env) recFunctionAtom() recAtom {
	e.assertNotFrozenForRecAtom()
	e.recFunctionAtomsMutex.Lock()
	defer e.recFunctionAtomsMutex.Unlock()
	result := len(e.recFunctionAtoms)
	e.recFunctionAtoms = append(e.recFunctionAtoms, nil)
	return createRecAtom(result, recAtomKindFunction)
}

func (e *env) setRecFunctionAtomType(rec recAtom, atomicType *functionAtomicType) {
	e.recFunctionAtomsMutex.Lock()
	defer e.recFunctionAtomsMutex.Unlock()
	e.recFunctionAtoms[rec.index()] = atomicType
	atomic.AddInt32(&e.populatedRecAtoms, 1)
}

func (e *env) getRecFunctionAtomType(rec recAtom) *functionAtomicType {
	e.recFunctionAtomsMutex.Lock()
	defer e.recFunctionAtomsMutex.Unlock()
	return e.recFunctionAtoms[rec.index()]
}

func (e *env) listAtom(atomicType *ListAtomicType) *typeAtom {
	return e.typeAtom(atomicType)
}

func (e *env) mappingAtom(atomicType *MappingAtomicType) *typeAtom {
	return e.typeAtom(atomicType)
}

func (e *env) functionAtom(atomicType *functionAtomicType) *typeAtom {
	return e.typeAtom(atomicType)
}

func (e *env) cellAtom(atomicType *cellAtomicType) *typeAtom {
	return e.typeAtom(atomicType)
}

// typeAtom returns the atom for atomicType. Persistent atoms are stored in the
// atomTable by value and a fresh pointer to a copy is returned (atom identity is
// by index, not pointer). After freeze, new atoms are ephemeral so the env can
// only retain them weakly.
func (e *env) typeAtom(atomicType atomicType) *typeAtom {
	if e.frozen.Load() {
		// After freezing atom table is immutable so we can do lookup lock free
		if ta, ok := e.atomTable[atomicType]; ok {
			return &ta
		}
		return e.newEphemeralAtom(atomicType)
	}
	e.atomTableMutex.Lock()
	if e.frozen.Load() {
		e.atomTableMutex.Unlock()
		if ta, ok := e.atomTable[atomicType]; ok {
			return &ta
		}
		return e.newEphemeralAtom(atomicType)
	}
	defer e.atomTableMutex.Unlock()
	ta, ok := e.atomTable[atomicType]
	if ok {
		return &ta
	}
	ta = createTypeAtom(len(e.atomTable), atomicType)
	e.atomTable[atomicType] = ta
	return &ta
}

// newEphemeralAtom allocates a type atom that the env retains only via a
// weak pointer. It reuses a slot whose previous occupant has been collected,
// otherwise appends a new slot. A slot whose gen has reached the max is retired
// (skipped) rather than wrapped, since reusing a gen could alias a stale memo
// entry on the atom's canonicalKey.
func (e *env) newEphemeralAtom(atomicType atomicType) *typeAtom {
	// In theory we can improve through put by having per slot rw locks and one lock for resizing the array
	// Idea is we can create ephemeral atoms concurrently, but I don't feel like dealing with the complexity
	// is worth it (array resizing, etc)
	e.ephemeralMu.Lock()
	defer e.ephemeralMu.Unlock()
	for i := range e.ephemeralSlots {
		slot := &e.ephemeralSlots[i]
		if slot.ptr.Value() != nil || slot.gen == math.MaxUint64 {
			continue
		}
		slot.gen++
		p := createEphemeralTypeAtom(e.frozenAtomCount+i, slot.gen, atomicType)
		slot.ptr = weak.Make(p)
		return p
	}
	i := len(e.ephemeralSlots)
	p := createEphemeralTypeAtom(e.frozenAtomCount+i, 1, atomicType)
	e.ephemeralSlots = append(e.ephemeralSlots, ephemeralSlot{gen: 1, ptr: weak.Make(p)})
	return p
}

func (e *env) listAtomType(atom atom) *ListAtomicType {
	if recAtom, ok := atom.(*recAtom); ok {
		return e.getRecListAtomType(*recAtom)
	}
	return atom.(*typeAtom).AtomicType.(*ListAtomicType)
}

func (e *env) functionAtomType(atom atom) *functionAtomicType {
	if recAtom, ok := atom.(*recAtom); ok {
		return e.getRecFunctionAtomType(*recAtom)
	}
	return atom.(*typeAtom).AtomicType.(*functionAtomicType)
}

func (e *env) mappingAtomType(atom atom) *MappingAtomicType {
	if recAtom, ok := atom.(*recAtom); ok {
		return e.getRecMappingAtomType(*recAtom)
	}
	return atom.(*typeAtom).AtomicType.(*MappingAtomicType)
}

func (e *env) recListAtom() recAtom {
	e.assertNotFrozenForRecAtom()
	e.recListAtomsMutex.Lock()
	defer e.recListAtomsMutex.Unlock()
	result := len(e.recListAtoms)
	e.recListAtoms = append(e.recListAtoms, nil)
	return createRecAtom(result, recAtomKindList)
}

func (e *env) recMappingAtom() recAtom {
	e.assertNotFrozenForRecAtom()
	e.recMappingAtomsMutex.Lock()
	defer e.recMappingAtomsMutex.Unlock()
	result := len(e.recMappingAtoms)
	e.recMappingAtoms = append(e.recMappingAtoms, nil)
	return createRecAtom(result, recAtomKindMapping)
}

func (e *env) setRecListAtomType(rec recAtom, atomicType *ListAtomicType) {
	e.recListAtomsMutex.Lock()
	defer e.recListAtomsMutex.Unlock()
	e.recListAtoms[rec.index()] = atomicType
	atomic.AddInt32(&e.populatedRecAtoms, 1)
}

func (e *env) setRecMappingAtomType(rec recAtom, atomicType *MappingAtomicType) {
	e.recMappingAtomsMutex.Lock()
	defer e.recMappingAtomsMutex.Unlock()
	e.recMappingAtoms[rec.index()] = atomicType
	atomic.AddInt32(&e.populatedRecAtoms, 1)
}

// IsReady reports whether every allocated recursive atom slot has been
// populated with a non-nil atomic type. Emptiness checks (semtypes.IsEmpty)
// require a ready env; calling them earlier may dereference an unset
// recursive atom and panic.
func (e *env) IsReady() bool {
	total := e.recListAtomCount() + e.recMappingAtomCount() + e.recFunctionAtomCount()
	return int(atomic.LoadInt32(&e.populatedRecAtoms)) == total
}

func (e *env) getRecListAtomType(rec recAtom) *ListAtomicType {
	e.recListAtomsMutex.Lock()
	defer e.recListAtomsMutex.Unlock()
	return e.recListAtoms[rec.index()]
}

func (e *env) getRecMappingAtomType(rec recAtom) *MappingAtomicType {
	e.recMappingAtomsMutex.Lock()
	defer e.recMappingAtomsMutex.Unlock()
	return e.recMappingAtoms[rec.index()]
}

func (e *env) cellAtomType(atom atom) *cellAtomicType {
	return atom.(*typeAtom).AtomicType.(*cellAtomicType)
}

// Public/package methods

// initializeEnv populates the environment with predefined atoms
// func (p *predefinedTypeEnv) initializeEnv(env Env) {
// 	fillRecAtoms(p, &env.recListAtoms, p.initializedRecListAtoms)
// 	fillRecAtoms(p, &env.recMappingAtoms, p.initializedRecMappingAtoms)
// 	for _, each := range p.initializedCellAtoms {
// 		env.cellAtom(each.atomicType)
// 	}
// 	for _, each := range p.initializedListAtoms {
// 		env.listAtom(each.atomicType)
// 	}
// }

// fillRecAtoms fills the environment rec atom list with initialized rec atoms
func fillRecAtoms[E atomicType](env *predefinedTypeEnv, envRecAtomList *[]E, initializedRecAtoms []E) {
	count := env.ReservedRecAtomCount()
	for i := range count {
		if i < len(initializedRecAtoms) {
			*envRecAtomList = append(*envRecAtomList, initializedRecAtoms[i])
		} else {
			var zero E
			*envRecAtomList = append(*envRecAtomList, zero)
		}
	}
}
