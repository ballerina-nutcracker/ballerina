# Intent: complete the `ballerina/lang.array` API surface

## Background: the two ways a langlib function can be declared

`lang.array` is split across three places:

| Place | Purpose |
| --- | --- |
| `lib/langlibs/ballerina/lang.array/0.0.1/any/lang.array.bal` | Ballerina-source declarations (`= external`) for functions whose signature is expressible in the normal type system, plus plain type/enum definitions |
| `model/opaque.go` (`OpaqueSymbols`) | Go-defined *opaque* function symbols for functions whose signature cannot be written in Ballerina source (the `@typeParam` ones) |
| `semantics/internal/types/type_resolver.go` (`arrayOpaqueMonomorphizers`) | Per-call-site monomorphization of each opaque function: validates the argument, computes the concrete signature, caches it |
| `lib/langlibs/go/lang.array/array.go` | The native implementation registered under the function name, for both kinds |

The decision rule is mechanical:

* **Normal Ballerina extern** — every parameter and the return type are fixed,
  concrete types (`(any|error)[]`, `byte[]`, `string`, `int`, `boolean`).
  Declare it in `lang.array.bal`, register a `NativeFunc` in `array.go`. Nothing
  else is needed. Defaultable parameters work here (see
  `string:substring(..., int endIndex = length(str))`).
* **Opaque function** — the signature mentions `Type`, `Type1` or
  `AnydataType`, i.e. the parameter or return type is derived from the *member
  type* of the array argument (`semtypes.ListProj(cx, containerTy, semtypes.Int)`).
  These need an entry in `model.OpaqueSymbols`, an `OpaqueFn…` id constant, a
  monomorphizer in the `arrayOpaqueMonomorphizers` table, and the native impl.
  The function must **not** appear in `lang.array.bal` — the opaque symbol is
  injected into the package scope before AST symbols are resolved
  (`injectOpaqueSymbols`), so a source declaration of the same name would be a
  redeclaration.

Two secondary consequences of that rule:

* A function that takes a **`@isolatedParam` function argument** must be opaque
  regardless of its other types: `@isolatedParam` is not an annotation the
  compiler reads from source; the information lives only on
  `OpaqueFunctionSymbol.IsIsolatedParam`, which `lock_analyzer.go` consults
  (`isolatedParamLambdas`, `isolatedInvocationViolationInner`).
* Opaque functions **do not support defaultable parameters**
  (`defaultable_params_handler.go` bails out for `*model.OpaqueFunctionSymbol`;
  see also the comment on `opaqueFunctionParams`). The existing workaround is
  arity-based monomorphization — `monomorphizeArrayIndexOf` monomorphizes a 2-
  or 3-parameter signature depending on what the call site passed, folds an
  arity marker into the cache key, and lets the Go extern default the missing
  argument.

## Out of scope: defaultable parameters for opaque functions

**Adding defaultable-parameter support to opaque functions is explicitly out of
scope for this work.** Opaque symbols carry no function signature of their own,
so there is nowhere to hang a default-value expression; giving them real
(untyped) signatures is the proper fix and is tracked separately (see the
comment on `opaqueFunctionParams`). Nothing in this intent should touch
`defaultable_params_handler.go` or the opaque-symbol representation.

Concretely, this means:

* Every opaque function below that the spec declares with a defaultable
  parameter — `slice` (`endIndex = arr.length()`), `lastIndexOf`
  (`startIndex = arr.length() - 1`) and `sort` (`direction = ASCENDING`,
  `key = ()`) — is implemented with **arity-based monomorphization only**,
  exactly as `indexOf` already is: monomorphize the signature for the arity the
  call site actually passed, fold an arity marker into the cache key, and let
  the Go extern supply the default for the arguments that were not passed.
* Named arguments for the omitted parameters are consequently **not supported**
  on these functions. `opaqueFunctionParams` exists only so that the names the
  call site *can* use resolve; leaving a name unset keeps the diagnostic honest.
* The defaults are therefore duplicated in Go rather than declared in Ballerina
  source. That duplication is accepted here and should be removed when opaque
  symbols gain real signatures, not worked around now.

Defaultable parameters remain fully available to the normal Ballerina externs in
section A — that path already works (`string:substring`).

## Current state

Implemented (opaque): `push`, `map`, `indexOf`, `remove`, `removeAll`, `toStream`.

Implemented (normal extern in `lang.array.bal`): `length`, `toBase16`,
`toBase64`, `fromBase16`, `fromBase64`.

## Missing

### A. Can be added as normal Ballerina externs

These need only a declaration in `lang.array.bal` plus a native function in
`lib/langlibs/go/lang.array/array.go`.

| Symbol | Declaration | Notes |
| --- | --- | --- |
| `setLength` | `public isolated function setLength((any|error)[] arr, int length) returns () = external;` | Needs a new `(*values.List).SetLength` that reuses `checkCanShrinkTo` when shrinking and the `filler` factory when growing (same checks as `FillingSet`/`Clear`). Panics on a readonly or fixed-length list. |
| `SortDirection` | `public enum SortDirection { ASCENDING = "ascending", DESCENDING = "descending" }` | Plain enum; enums are supported (`corpus/bal/subset8/08-enum/`). Needed by `sort`. |
| `OrderedType` | `public type OrderedType ()|boolean|int|float|decimal|string|OrderedType[];` | Plain (recursive) type definition; needed by `sort`'s `key` parameter type. Verify the recursive union resolves in a langlib bundle before relying on it. |

Note that `length` and `removeAll` also have fully concrete spec signatures, but
`removeAll` is deliberately opaque today so the resolver can reject a `readonly`
argument at compile time; leave it as it is.

### B. Must be opaque functions

Each of these needs: an `OpaqueFnArray…` id in `model/opaque.go`, an entry in
the `lang.array` case of `OpaqueSymbols` (with the right `IsIsolatedParam`
predicate), a monomorphizer registered in `arrayOpaqueMonomorphizers`, an entry
in `opaqueFunctionParams` if it should accept named arguments, and a
`RegisterExternFunction` in `array.go`.

| Symbol | Why opaque | Monomorphized signature | Implementation notes |
| --- | --- | --- | --- |
| `iterator` | Return type is an object whose `next` returns `record {| Type value; |}?` | `(containerTy) -> object { public isolated function next() returns record {| M value; |}?; }` | Build the object type the way `createXMLIteratorType` does (and cache it the same way). Runtime: return a `values.NewObject` carrying the list plus a cursor, with `next` bound to a second registered extern (`$arrayIterator.next`), exactly as `lang.xml`'s `xmlIterator`/`xmlIteratorNext` pair does. Not needed by `foreach` — that is lowered natively — this is purely the public API. |
| `enumerate` | Returns `[int, Type][]` | `(containerTy) -> [int, M][]` | Build the tuple type with `semtypes.NewListDefinition().Define(env, []SemType{Int, M}, …)` and the result array with a `ListRest` of that tuple. |
| `forEach` | `function(Type) returns ()` param, `@isolatedParam` | `(containerTy, function(M) returns ()) -> ()` | `IsIsolatedParam` → index 1. Loop with `ctx.InvokeFunctionValue`. |
| `filter` | `function(Type) returns boolean` param, `@isolatedParam`; returns `Type[]` | `(containerTy, function(M) returns boolean) -> M[]` | `IsIsolatedParam` → index 1. Result list built like `arrayMap` (fresh `ListRest(M)` definition + `FillerFactoryFor`). |
| `reduce` | `Type1` accumulator inferred from `initial`; `@isolatedParam` | `(containerTy, function(A, M) returns A, A) -> A` | `IsIsolatedParam` → index 1. `A` comes from resolving the `initial` argument (and/or `expectedType`), the way `monomorphizeArrayMap` derives its result member type. Cache key must include `A`. |
| `some` | `function(Type) returns boolean` param, `@isolatedParam` | `(containerTy, function(M) returns boolean) -> boolean` | `IsIsolatedParam` → index 1. Short-circuits on the first `true`; `false` for an empty array. |
| `every` | same as `some` | `(containerTy, function(M) returns boolean) -> boolean` | `IsIsolatedParam` → index 1. Short-circuits on the first `false`; `true` for an empty array. |
| `slice` | Returns `Type[]`; has a defaultable `endIndex` | `(containerTy, int[, int]) -> M[]` | Arity-based monomorphization like `indexOf`; `endIndex` defaults to `arr.length()` **in the extern**, not in the signature (defaultable params are out of scope — see above). Panics on `startIndex < 0`, `endIndex > length`, `startIndex > endIndex`. |
| `lastIndexOf` | `AnydataType` param; defaultable `startIndex` | `(containerTy, M[, int]) -> int?` | Same `anydata[]` subtype check as `indexOf`, same arity trick (`startIndex` defaults to `length - 1` in the extern), searching backwards with `values.DeepEquals`. |
| `reverse` | Returns `Type[]` | `(containerTy) -> M[]` | Fresh list, no mutation of the source. |
| `sort` | Returns `Type[]`; `key` is an `isolated function` returning `OrderedType`; two defaultable params | `(containerTy[, SortDirection][, (isolated function(M) returns OrderedType)?]) -> M[]` | The hardest one. Both defaults are handled by arity-based monomorphization plus extern-side defaulting (`ASCENDING`, `()`) — not by defaultable parameters, which are out of scope. Note the arity table is two-dimensional here (`direction` may be given without `key`, but not the reverse), so the cache key needs both markers. Also needs a decision on whether `key` is an `isolatedParam`. Runtime can reuse `values.CompareK(x, y, ascending)` — the same comparison the query `sort` clause uses via `lang.__internal:querySort` (see `compareQuerySortValues`). Should panic when the member type is not an ordered type and no `key` is given. |
| `pop` | Returns `Type` | `(containerTy) -> M` | Reject `readonly` at compile time (as `remove` does). Runtime is `RemoveAt(len-1)`; panic on empty. |
| `shift` | Returns `Type` | `(containerTy) -> M` | Reject `readonly` at compile time. Runtime is `RemoveAt(0)` — `checkShiftedMemberTypes` already gives the right tuple behaviour; panic on empty. |
| `unshift` | `Type...` rest parameter | `(containerTy) -> ()` with `RestParamType: M` | Reject `readonly` at compile time. Needs a new `(*values.List).InsertAt`/`Prepend` that type-checks every member against its *new* index (the mirror of `checkShiftedMemberTypes`). Register `[{Name: "arr"}, {Name: "vals", Flag: ParamFlagRestParam}]` in `opaqueFunctionParams`, and note that `storeMonomorphizedOpaqueFn` currently hard-codes the rest-param flag as `sym.Name() == "push"` — that check has to become a set that includes `unshift`. |

## Runtime (`values`) gaps this exposes

`values.List` today offers `Len`, `Get`, `FillingGet`, `FillingSet`, `Append`,
`RemoveAt`, `Clear`. The work above needs:

* `SetLength(tc, n)` — shrink via `checkCanShrinkTo`, grow via `filler`.
* `InsertAt(tc, idx, vs...)` (or `Prepend`) for `unshift`, with the
  inherent-member-type check applied at the shifted indices.

Everything else (`pop`, `shift`, `slice`, `reverse`, `sort`, `filter`,
`enumerate`) composes from the existing operations plus `NewList` +
`FillerFactoryFor`, as `arrayMap` already does.

## Pre-existing gap noticed while surveying

`monomorphizeArrayPush` does not reject a `readonly` container at compile time,
whereas `monomorphizeArrayRemove` and `monomorphizeArrayRemoveAll` do. Pushing
to a `readonly` array is therefore a runtime panic rather than a compile error,
and there is no `array-push-readonly-e.bal` corpus test. Worth fixing alongside
the new mutating functions (`pop`, `shift`, `unshift`, `setLength`) so all of
them behave consistently.

## Testing

Per `AGENTS.md`, prefer corpus tests over unit tests. New cases go in
`corpus/bal/subset10/10-langlibs/` following the existing
`array-<function>-<case>-{v,e,p}.bal` naming, with goldens regenerated per stage
(`-update`). Each function wants at least:

* a `-v` test covering the ordinary path and the empty-array edge case;
* a `-p` test for each documented panic (empty `pop`/`shift`, out-of-range
  `slice`, fixed-length `setLength`, tuple mandatory-member violations);
* an `-e` test for the compile-time rejections (`readonly` receiver for the
  mutating functions, non-`anydata` receiver for `lastIndexOf`).

Tuple and fixed-length-array receivers deserve explicit coverage for every
length-changing function, since that is where `checkCanShrinkTo` and
`checkShiftedMemberTypes` do the interesting work.
