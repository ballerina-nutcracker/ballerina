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

package semantics

import (
	"ballerina/ast"
	"ballerina/context"
	"ballerina/model"
	"ballerina/semtypes"
	"ballerina/tools/diagnostics"
)

// loopTypeResolver wraps the resolver active for the body of a while/foreach
// loop. Every operation other than recordBreak / recordContinue delegates to
// the parent. break / continue chains are accumulated locally so the loop arm
// can fold them into the post-loop chain.
type loopTypeResolver struct {
	parentResolver typeResolver
	breaks         []*binding
	continues      []*binding
}

func (l *loopTypeResolver) recordBreak(chain *binding) {
	l.breaks = append(l.breaks, chain)
}

func (l *loopTypeResolver) recordContinue(chain *binding) {
	l.continues = append(l.continues, chain)
}

func (l *loopTypeResolver) typeContext() semtypes.Context { return l.parentResolver.typeContext() }
func (l *loopTypeResolver) expectedReturnType() semtypes.SemType {
	return l.parentResolver.expectedReturnType()
}
func (l *loopTypeResolver) parent() typeResolver { return l.parentResolver }
func (l *loopTypeResolver) nextMonoFnName(origName string) string {
	return l.parentResolver.nextMonoFnName(origName)
}
func (l *loopTypeResolver) typeEnv() semtypes.Env { return l.parentResolver.typeEnv() }
func (l *loopTypeResolver) xmlIteratorTypeCache() *semtypes.SemTypeCache {
	return l.parentResolver.xmlIteratorTypeCache()
}

func (l *loopTypeResolver) semanticError(msg string, loc diagnostics.Location) {
	l.parentResolver.semanticError(msg, loc)
}

func (l *loopTypeResolver) internalError(msg string, loc diagnostics.Location) {
	l.parentResolver.internalError(msg, loc)
}

func (l *loopTypeResolver) unimplemented(msg string, loc diagnostics.Location) {
	l.parentResolver.unimplemented(msg, loc)
}

func (l *loopTypeResolver) syntaxError(msg string, loc diagnostics.Location) {
	l.parentResolver.syntaxError(msg, loc)
}

func (l *loopTypeResolver) symbolType(ref model.SymbolRef) semtypes.SemType {
	return l.parentResolver.symbolType(ref)
}

func (l *loopTypeResolver) setSymbolType(ref model.SymbolRef, ty semtypes.SemType) {
	l.parentResolver.setSymbolType(ref, ty)
}

func (l *loopTypeResolver) getSymbol(ref model.SymbolRef) model.Symbol {
	return l.parentResolver.getSymbol(ref)
}

func (l *loopTypeResolver) unnarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return l.parentResolver.unnarrowedSymbol(ref)
}

func (l *loopTypeResolver) symbolName(ref model.SymbolRef) string {
	return l.parentResolver.symbolName(ref)
}

func (l *loopTypeResolver) createNarrowedSymbol(ref model.SymbolRef) model.SymbolRef {
	return l.parentResolver.createNarrowedSymbol(ref)
}

func (l *loopTypeResolver) createFunctionSymbol(space *model.SymbolSpace, name string, sig model.FunctionSignature, fnTy semtypes.SemType) model.SymbolRef {
	return l.parentResolver.createFunctionSymbol(space, name, sig, fnTy)
}

func (l *loopTypeResolver) compilerContext() *context.CompilerContext {
	return l.parentResolver.compilerContext()
}

func (l *loopTypeResolver) lookupImportedSymbols(name string) (model.ExportedSymbolSpace, bool) {
	return l.parentResolver.lookupImportedSymbols(name)
}

func (l *loopTypeResolver) addImplicitImport(name string, imp ast.BLangImportPackage) {
	l.parentResolver.addImplicitImport(name, imp)
}

func (l *loopTypeResolver) hasImplicitImport(name string) bool {
	return l.parentResolver.hasImplicitImport(name)
}

func (l *loopTypeResolver) trackCapturedVar(ref model.SymbolRef) {
	l.parentResolver.trackCapturedVar(ref)
}

func (l *loopTypeResolver) getCapturedVars() map[model.SymbolRef]bool {
	return l.parentResolver.getCapturedVars()
}

func (l *loopTypeResolver) setCapturedVars(vars map[model.SymbolRef]bool) {
	l.parentResolver.setCapturedVars(vars)
}

func (l *loopTypeResolver) ensureResolved(ref model.SymbolRef, depth int) bool {
	return l.parentResolver.ensureResolved(ref, depth)
}

func (l *loopTypeResolver) setMappingAtomBType(mat *semtypes.MappingAtomicType, bType ast.BType) {
	l.parentResolver.setMappingAtomBType(mat, bType)
}

func (l *loopTypeResolver) getMappingAtomBType(mat *semtypes.MappingAtomicType) (ast.BType, bool) {
	return l.parentResolver.getMappingAtomBType(mat)
}

func (l *loopTypeResolver) setMappingAtomSymRef(mat *semtypes.MappingAtomicType, ref model.SymbolRef) {
	l.parentResolver.setMappingAtomSymRef(mat, ref)
}

func (l *loopTypeResolver) getMappingAtomSymRef(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	return l.parentResolver.getMappingAtomSymRef(mat)
}

func (l *loopTypeResolver) setClassAtomSymbol(mat *semtypes.MappingAtomicType, symbol model.SymbolRef) {
	l.parentResolver.setClassAtomSymbol(mat, symbol)
}

func (l *loopTypeResolver) getClassAtomSymbol(mat *semtypes.MappingAtomicType) (model.SymbolRef, bool) {
	return l.parentResolver.getClassAtomSymbol(mat)
}

func (l *loopTypeResolver) currentScope() model.Scope     { return l.parentResolver.currentScope() }
func (l *loopTypeResolver) setCurrentScope(s model.Scope) { l.parentResolver.setCurrentScope(s) }

func (l *loopTypeResolver) nextDefaultFnName() string {
	return l.parentResolver.nextDefaultFnName()
}

func (l *loopTypeResolver) lookupClassMethodSymbol(receiverTy semtypes.SemType, methodName string) (model.SymbolRef, bool) {
	return l.parentResolver.lookupClassMethodSymbol(receiverTy, methodName)
}

func (l *loopTypeResolver) ensureNotEmpty(ty semtypes.SemType, onEmpty func()) bool {
	return l.parentResolver.ensureNotEmpty(ty, onEmpty)
}

// validateLoopAssignments emits diagnostics for every assignment, inside the loop
// body, to a variable narrowed outside the loop, when the assignment's effect
// reaches the top of the loop — i.e. it is on the body's natural-completion
// path or on a continue path. Break paths are excluded because they exit the
// loop and therefore cannot leak into the next iteration.
func validateLoopAssignments(t typeResolver, loopT *loopTypeResolver, bodyEffect statementEffect, loopEntry *binding) {
	chains := make([]*binding, 0, 1+len(loopT.continues))
	if !bodyEffect.nonCompletion {
		chains = append(chains, bodyEffect.binding)
	}
	chains = append(chains, loopT.continues...)
	reportOutsideLoopAssignments(t, chains, loopEntry)
}
