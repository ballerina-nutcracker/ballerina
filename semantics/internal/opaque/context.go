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

package opaque

import (
	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/semtypes"
)

// cacheOwner identifies the opaque function a monomorphization cache entry
// belongs to. Two functions can pick the same cache keys - map:get and map:remove
// both key on the container type alone - so the cache is partitioned by owner and
// a key never means one function's monomorphization to another. It is the
// function's table slot, so it cannot drift from the definition it selects.
type cacheOwner struct {
	pkg packageKey
	id  int
}

// monoCacheNode is one level of the monomorphization cache trie. A node is
// reached by interning the cache keys in order.
type monoCacheNode struct {
	children map[semtypes.InternHandle]*monoCacheNode
	ref      model.SymbolRef
	stored   bool
}

// Context holds the per-resolver state monomorphization needs: the borrowed
// semtypes.Context the resolver owns, the monomorphization cache, and the XML
// iterator type cache. A Context belongs to exactly one resolver and a resolver
// is confined to one goroutine, so it needs no synchronization.
type Context struct {
	typeCtx          semtypes.Context
	monoInterner     *semtypes.SemTypeInterner
	monoCaches       map[cacheOwner]*monoCacheNode
	xmlIteratorTypes *semtypes.SemTypeCache
}

// NewContext builds a Context over typeCtx. It borrows typeCtx; it does not
// create or own a semtypes.Context.
func NewContext(typeCtx semtypes.Context) *Context {
	return &Context{
		typeCtx:          typeCtx,
		monoInterner:     semtypes.NewSemtypeInterner(),
		monoCaches:       make(map[cacheOwner]*monoCacheNode),
		xmlIteratorTypes: semtypes.NewSemTypeCache(),
	}
}

func (c *Context) typeContext() semtypes.Context { return c.typeCtx }

func (c *Context) typeEnv() semtypes.Env { return c.typeCtx.Env() }

func (c *Context) monoCacheChild(node *monoCacheNode, key semtypes.SemType, create bool) *monoCacheNode {
	handle := c.monoInterner.Intern(key)
	next := node.children[handle]
	if next == nil && create {
		next = &monoCacheNode{children: make(map[semtypes.InternHandle]*monoCacheNode)}
		node.children[handle] = next
	}
	return next
}

func (c *Context) monoCacheRoot(owner cacheOwner, create bool) *monoCacheNode {
	root := c.monoCaches[owner]
	if root == nil && create {
		root = &monoCacheNode{children: make(map[semtypes.InternHandle]*monoCacheNode)}
		c.monoCaches[owner] = root
	}
	return root
}

func (c *Context) monoCacheNodeFor(owner cacheOwner, cacheKey semtypes.SemType,
	cacheKeyRest []semtypes.SemType, create bool) *monoCacheNode {
	node := c.monoCacheRoot(owner, create)
	if node == nil {
		return nil
	}
	node = c.monoCacheChild(node, cacheKey, create)
	for _, key := range cacheKeyRest {
		if node == nil {
			return nil
		}
		node = c.monoCacheChild(node, key, create)
	}
	return node
}

// lookupMono returns the monomorphic symbol owner stored under the given cache
// keys.
func (c *Context) lookupMono(owner cacheOwner, cacheKey semtypes.SemType,
	cacheKeyRest ...semtypes.SemType) (model.SymbolRef, bool) {
	node := c.monoCacheNodeFor(owner, cacheKey, cacheKeyRest, false)
	if node == nil || !node.stored {
		return model.SymbolRef{}, false
	}
	return node.ref, true
}

// storeMono caches ref for owner under the given cache keys.
func (c *Context) storeMono(owner cacheOwner, ref model.SymbolRef, cacheKey semtypes.SemType,
	cacheKeyRest ...semtypes.SemType) {
	node := c.monoCacheNodeFor(owner, cacheKey, cacheKeyRest, true)
	node.ref = ref
	node.stored = true
}

// xmlIteratorType returns the iterator object type for itemTy, building it with
// build on the first request.
func (c *Context) xmlIteratorType(itemTy semtypes.SemType, build func() semtypes.SemType) semtypes.SemType {
	return c.xmlIteratorTypes.GetOrBuild(itemTy, build)
}

// FunctionSemType builds the semtype of a typed function signature.
func FunctionSemType(env semtypes.Env, sig model.TypedFunctionSignature) semtypes.SemType {
	paramListDefn := semtypes.NewListDefinition()
	paramListTy := paramListDefn.Define(env, sig.ParamTypes, semtypes.ListRest(sig.RestParamType),
		semtypes.ListMutability(semtypes.CellMutabilityNone))
	fnDefn := semtypes.NewFunctionDefinition()
	return fnDefn.Define(env, paramListTy, sig.ReturnType,
		semtypes.FunctionQualifiersFrom(env, sig.IsIsolated(), sig.IsTransactional()))
}
