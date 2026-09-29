# Mapping constructor spread fields

Related issue: [#858](https://github.com/ballerina-nutcracker/ballerina/issues/858).

## Goals

- Support `...expression` fields in mapping constructors throughout compilation, BIR serialization, and execution.
- Accept operands whose static semantic type is a subtype of `map<any|error>`, including records, mapping unions, intersections, aliases, and narrowed mapping types.
- Diagnose statically possible duplicate keys using semantic types, not source-level record/map AST descriptors.
- Include spread members, presence information, and rest types in inferred inherent record types.
- Enforce contextual field/rest typing, required-field presence, defaults, and readonly constraints.
- Preserve evaluation semantics, iteration order, and shallow-copy behavior.
- Use jBallerina tests as behavioral references without reproducing its implementation strategy.

## Non-goals

- Implementing computed-name fields (`[expression]: value`) or their overwrite semantics. They are a separate feature and must not be classified as specific fields by new collision checks.
- General-purpose last-write-wins merging of specific fields and spreads.
- Deep-copying spread members or transferring a source field's readonly qualification automatically.
- Reworking contextual mapping-union selection for constructors unrelated to spreads.
- Expanding the language's definition of constant expressions.
- Adding standard-library APIs or changing parser grammar, which already recognizes spread fields.

## Success criteria

- Valid spread constructors compile and execute, including after BIR roundtrip.
- Invalid operands, possible collisions, incompatible target fields/rests, and missing required fields produce source-located compiler diagnostics, not internal errors or panics.
- Union/intersection handling follows semantic possibility, including semantically empty fields and negative constraints introduced by narrowing.
- Inference tests demonstrate correct member types, optionality, rest types, and runtime inherent-type enforcement.
- Every scenario in the test matrix below has source-level corpus coverage where expressible in the currently supported language.
- Existing mapping-constructor tests remain passing; computed-name support is not accidentally expanded.

A valid example is:

```ballerina
import ballerina/io;

public function main() {
    record {| int x; int y; |} first = {x: 1, y: 2};
    map<int> result = {...first, z: 3};
    io:println(result); // {"x":1,"y":2,"z":3}
}
```

This example was verified on jBallerina 2201.13.4. Declaring `first` as `map<int>` instead is a compile-time error: its static type permits `z`, even though its current value does not contain it. Issue #858 has been corrected accordingly.

# Design

## 1. AST construction and traversal

Represent a spread as a distinct mapping-field node containing its expression and source position. The field node itself has type `never`; the operand retains its resolved value type. Preserve the constructor's source-order field sequence.

All visitors that currently assume every field is a key/value field must dispatch on the field kind. Symbol resolution, type resolution, semantic analysis, isolation checks, pretty-printing, and desugaring must visit the spread operand. A spread introduces no synthetic source-level field-name symbols.

Constant-expression validation must not silently skip a spread. Preserve language legality for constant contexts and diagnose nonconstant constructors normally; this feature does not make arbitrary spread operands constant expressions.

## 2. Semantic mapping queries

Use semantic types as the authority. AST nodes supply expressions, source locations, and decoded names only.

For an inhabited mapping type `T`, distinguish:

- **Possible presence:** some value of `T` contains key `k`.
- **Guaranteed presence:** every value of `T` contains key `k`.
- **Member value type:** the possible values at `k`, excluding absence (`undef`).
- **Remaining-key value type:** member values for names outside a finite set of distinguished names.

Absence is not a nil member value. An empty mapping type is not the same as an inhabited empty record. An uninhabited alternative contributes neither possible names nor values; a wholly uninhabited operand follows existing unreachable-expression rules rather than being treated as an empty spread.

### Unions, intersections, and negative constraints

- A union permits a key if any inhabited alternative permits it.
- An intersection permits a key only if the constraints jointly permit its presence with an inhabited member value type. Merely declaring a name in every intersected atom is insufficient.
- Rest descriptors may permit a name without explicitly declaring it.
- Explicit named exclusions override that atom's rest descriptor.
- Required presence must be established across all inhabited alternatives, not borrowed from one union branch.
- Negative constraints must be respected when deciding possibility and presence. A positive-only approximation must not generate false duplicate errors.

For example, intersecting optional `int x?` and optional `string x?` excludes `x`, since the intersected member value type is empty. In contrast, two independent spreads respectively permitting `int x` and `string x` collide: values need not overlap for names to duplicate.

### Candidate names are an index, not a proof

Gather the finite distinguished names from relevant mapping atoms, including names needed to preserve exclusions and negative constraints. Intersected positive atoms from `MappingAlternatives` are useful inputs, but do not alone describe the complete operand type.

Partition the key space into these named keys and the remaining names. Query semantic possibility/value types in each partition. Include explicit constructor keys and target field names where relevant. This accounts for both named and arbitrary rest keys.

The existing `MappingAlternative.Type()` describes the positive intersection, and the existing alternative-field validator panics on negative atoms. Neither behavior is an acceptable exact oracle for spread operands. Likewise, existing projection helpers must not be assumed exact for this purpose without validating their handling of negative constraints. New spread queries must have full-semantic-type contracts.

Do not expose or store `model.Symbol` as a map key. No symbol-keyed data is needed for the semantic shape queries.

## 3. Operand resolution and duplicate checks

Resolve each operand and require it to be a subtype of `map<any|error>`. A union containing a nonmapping possibility is invalid. Do not infer validity from a runtime value or a syntactic record declaration.

For each specific field and spread, regardless of textual order:

- Reject if the spread can contain the specific field's decoded key.
- Optional inhabited fields can collide.
- Fields with semantically empty value types cannot collide; use semantic emptiness, not only the literal `never` tag.

For each pair of spreads, reject if their possible key sets overlap, including overlap in the remaining-name partition. Do not require their member value types to overlap.

Examples that must be valid:

```ballerina
record {| never x?; |} empty = {};
var a = {x: 1, ...empty};

record {| never x?; int...; |} other = {y: 2};
var b = {x: 1, ...other};
```

A diagnostic should identify the conflicting key when one is available, otherwise identify overlapping spread key domains. Attach it to a relevant field/spread location. Duplicate specific-field diagnostics retain existing behavior.

## 4. Bottom-up inherent-type inference

Infer an atomic record construction shape incorporating:

- Existing specific-field inference and explicit readonly qualifiers.
- Each possible named member of every spread, with its semantic value type.
- Guaranteed versus optional presence across alternatives.
- The union of spread rest/member contributions outside the distinguished names.
- Explicit impossible-name exclusions where dropping them would let an inhabited inferred rest descriptor admit those names.

Do not collapse a record spread to `map<union-of-all-values>`: that loses per-field types and presence. Do not construct an inherent type that is only a union of source alternatives; produce the record construction shape needed by the existing runtime.

Spread source field-level readonly qualification does not automatically qualify the new field. Readonly types of the stored values remain part of the member type. Keep existing broadening rules for ordinary specific fields; spreading an already typed mapping does not re-evaluate or contextually convert its stored members.

## 5. Contextual construction

Preserve existing contextual inherent-type selection policy, extending its inputs to account for spread members and guaranteed presence. Do not arbitrarily select one spread-union alternative. Retain missing/ambiguous inherent-type diagnostics rather than guessing.

Once a target construction shape is selected:

- Provide each spread expression the appropriate contextual `map<T>`, where `T` encompasses the target's possible member values. This supports contextual typing of nested operand expressions; it does not replace per-key checks.
- Where the operand is itself a mapping constructor with known keys, type it against exactly those keys, each with the type the target holds for it. An open `map<T>` would replace the operand's inferred shape with one that can supply any key, so `record {| int x; |} r = {...{x: 1}};` would be rejected.
- Validate possible spread members against corresponding declared target fields.
- Validate extra named source members against target rest.
- Validate source remaining-key values against target rest.
- Where source rest can populate a declared target name, validate that contribution against that named target field too, respecting source exclusions.
- A target with effective rest type `never` rejects inhabited extra-key possibilities, even if the source's current value happens to have no extra keys.
- Judge closure by the effective rest type, not `{| |}` delimiters alone; exclusive record syntax can contain an explicit rest descriptor.
- Count a spread as satisfying a required field only when presence is guaranteed. Optional fields, arbitrary map entries, and excluded fields do not establish required presence.
- Preserve target defaults as a separate source of guaranteed initialization.
- Enforce readonly target/member constraints without freezing or modifying the source mapping implicitly.

Whole-shape validation must also preserve applicable semantic constraints; flattening a summary is not permission to discard target exclusions or source negative constraints during validation.

Resolution must propagate operand expression effects through the existing binding/effect machinery. Re-resolution for contextual typing must not result in duplicate runtime evaluation.

## 6. Desugaring, BIR, and execution

Retain spreads as explicit construction entries through BIR. The existing BIR entry interface and serialized key/value discriminator provide an appropriate extension point. This avoids lowering construction into ordinary post-construction stores, which would complicate readonly initialization, required fields, and defaults.

Desugar each operand normally and preserve source-order evaluation, including ordering relative to specific-field expressions and hoisted operand work. Emit exactly one evaluation per operand.

Following the existing evaluate-operands-then-construct BIR model, entry expressions are evaluated in source order before `NewMap` expands their stored mapping references. Do not silently introduce snapshots at each spread expression. Include a differential regression for a later expression mutating an earlier spread source, since this ordering is observable and jBallerina also retains references before expansion.

At construction:

1. Process entries in field order.
2. Append specific entries and expand each spread in the source mapping's iteration order.
3. Track actually supplied names, including spread names.
4. Evaluate defaults only for names still absent, using the existing default ordering.
5. Construct the destination with its selected inherent type and readonly state.

This is a shallow copy of entries into a new mapping. Member references remain shared. Merely spreading a source does not mutate it. Subsequent updates must obey the destination's inherent type and readonly rules.

Spread collisions are compile-time errors, not a new runtime overwrite facility. Runtime panics in the tests concern ordinary operand evaluation or inherent-type/readonly updates, not statically admitted duplicate spreads.

Serialization and deserialization must preserve the entry kind, operand, order, target type, defaults, and readonly state. Use the currently reserved non-key/value entry encoding and document its payload; retain existing key/value encoding.

# API changes

The signatures below define the intended API surface. Existing signatures listed as unchanged acquire the stated spread behavior. New exports are limited to types/functions needed across package boundaries. No existing API is removed. Further implementation-local helper extraction must preserve these contracts and be recorded in this section if added during implementation.

## `ast/expressions.go`

Add an exported node, needed by nodebuilder, semantics, desugar, and birgen:

```go
type BLangMappingSpreadField struct {
    bLangNodeBase
    Expr BLangExpression
}

func (b *BLangMappingSpreadField) IsKeyValueField() bool
```

`IsKeyValueField` returns false. The embedded node base provides the existing node contract. `Expr` and position must be initialized during building. `BLangMappingConstructorExpr.Fields []MappingField` and the `MappingField` interface remain unchanged.

## `nodebuilder/node_builder.go`

Existing private signatures remain:

```go
func (n *nodeBuilder) transformMappingConstructorExpression(mappingConstructorBLangExpression *st.MappingConstructorExpressionNode) ast.BLangNode
func (n *nodeBuilder) transformSpreadField(spreadFieldNode *st.SpreadFieldNode) ast.BLangNode
```

Replace unsupported-spread diagnostics with construction of the spread node and operand. Preserve field order and positions.

## `ast/walk.go`

Existing exported signature remains:

```go
func Walk(v Visitor, node BLangNode)
```

Visit spread nodes and their operands exactly once. Preserve existing key/value traversal. Existing symbol resolution uses this traversal and needs no new symbol API.

## `ast/pretty_printer.go`

Existing signatures remain:

```go
func (p *PrettyPrinter) PrintInner(node BLangNode)
func (p *PrettyPrinter) printMappingConstructor(node *BLangMappingConstructorExpr)
```

Recognize and print spread nodes in source order. Add a private printer:

```go
func (p *PrettyPrinter) printMappingSpreadField(node *BLangMappingSpreadField)
```

Print the operand with an identifiable spread representation for AST/desugared goldens.

## `semtypes/mapping_spread.go` (new)

Add the following exported cross-package contracts:

```go
type MappingSpreadShape struct {
    Fields []Field
    Rest SemType
    Inhabited bool
}

func MappingShapeForSpread(cx Context, ty SemType) MappingSpreadShape
func MappingSpreadAllowsKey(cx Context, ty SemType, name string) bool
func MappingSpreadsOverlap(cx Context, left, right SemType) (name string, overlap bool)
```

- Preconditions: operands are semantic subtypes of mapping; callers diagnose nonmapping operands first.
- `MappingShapeForSpread` returns sorted distinguished fields with value types and optionality, plus remaining-key value type. Preserve impossible named exclusions where relevant. An uninhabited type returns `Inhabited == false`, no fields, and `Rest == Never`; an inhabited empty record returns `Inhabited == true`.
- A nonoptional field denotes guaranteed presence, not merely occurrence in one atom. Summary field types encompass all permitted values without importing source field-level readonly flags. The summary is an inferred atomic construction shape, not a replacement for the original semantic type in exact collision queries.
- Summary member types account for negative constraints. The projection helper ignores negated mapping atoms, so the shape narrows it by discarding the parts no value of the type can hold at the key, decided by emptiness. Where the split is bounded, the remaining part stays at the projection's own over-approximation.
- `MappingSpreadAllowsKey` tests possible presence against the complete type, including negative constraints. It returns false for uninhabited types and semantically excluded fields.
- `MappingSpreadsOverlap` tests possible name overlap between independent mapping values. When overlap is found, `name` is a deterministic diagnostic witness if available; the boolean is authoritative, since the empty string is itself a valid key. Do not compare member value-type intersection between the two spreads.
- Implementations handle mapping top, recursive member types, unions, intersections, and negative constraints without panicking or enumerating an infinite rest key space.

No change to the public contracts of existing `MappingAlternatives`, projection helpers, or mapping-atom accessors is required. Do not change their meaning silently to implement these new contracts.

## `semantics/internal/types/type_resolver.go`

Existing private signatures remain:

```go
func resolveMappingConstructorExpr(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool)
func resolveMappingConstructorBottomUp(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr) (semtypes.SemType, expressionEffect, bool)
func resolveMappingConstructorWithExpectedType(t typeResolver, chain *binding, e *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool)
func selectMappingInherentType(t typeResolver, expr *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) (semtypes.SemType, *semtypes.MappingAtomicType, bool)
```

Replace key/value-only assumptions with field-kind dispatch. Resolve spread operands, preserve effects, and use semantic shapes for inference and contextual selection. Account for optionality and rests rather than passing spread fields through the existing specific-field-only `MappingFieldInfo` representation. Avoid the negative-atom panic path when processing spread types. Preserve default lookup and the selected atomic construction type.

## `semantics/internal/analysis/semantic_analyzer.go`

Existing private signature remains:

```go
func analyzeMappingConstructorExpr[A analyzer](a A, expr *ast.BLangMappingConstructorExpr, expectedType semtypes.SemType) bool
```

Add semantic operand, collision, target compatibility, and guaranteed-presence checks. Specific-field naming restrictions do not become restrictions on names copied by spreads. Use the original operand semantic types for collision queries, not inferred flattened shapes. Validate the resolved constructor against its expected type as before.

## `semantics/internal/analysis/lock_analyzer.go`

Existing private signature remains:

```go
func isIsolatedExpressionInner(a analyzer, expr ast.BLangExpression, checkInvocableOperands bool) bool
```

Inspect spread operands using the same isolation rules as other constructor operand expressions, rather than reporting an unexpected field kind.

## `semantics/internal/common/common.go`

Existing exported-in-internal-package signature remains:

```go
func ValidateConstantExpr(ctx *context.CompilerContext, expr ast.BLangExpression, onNonConst func(ast.BLangExpression))
```

Do not skip spread fields during constant legality checking. Report a nonconstant constructor through `onNonConst` where the language's constant-expression rules exclude it. The existing constant evaluator's nonconstant error path remains valid; no new constant-evaluation API is introduced.

## `desugar/expression.go`

Existing private signature remains:

```go
func walkMappingConstructorExpr(cx *functionContext, expr *ast.BLangMappingConstructorExpr) desugaredNode[ast.BLangActionOrExpression]
```

Desugar spread operands and retain spread nodes. Preserve evaluation order when collecting initialization statements; earlier non-hoisted operand evaluations must not move after later hoisted work.

## `bir/non_terminator.go`

Existing `MappingConstructorEntry` interface is unchanged:

```go
type MappingConstructorEntry interface {
    IsKeyValuePair() bool
    ValueOp() *BIROperand
}
```

Add an exported entry type and constructor, needed by birgen and codec:

```go
type MappingConstructorSpreadEntry struct {
    valueOp *BIROperand
}

func NewMappingConstructorSpreadEntry(valueOp *BIROperand) *MappingConstructorSpreadEntry
func (m *MappingConstructorSpreadEntry) IsKeyValuePair() bool
func (m *MappingConstructorSpreadEntry) ValueOp() *BIROperand
```

The constructor initializes the sole field. `IsKeyValuePair` returns false; `ValueOp` is the already evaluated source mapping reference. The existing `NewMap` instruction and its exported factory retain their signatures and accept the extended entry interface:

```go
func NewMapConstructor(typ semtypes.SemType, lhsOp *BIROperand, values []MappingConstructorEntry, defaults []MappingConstructorDefaultEntry, isReadonly bool, pos Location) *NewMap
```

## `birgen/bir_gen.go`

Existing private signatures remain:

```go
func mappingConstructorExpression(ctx context, curBB *bir.BIRBasicBlock, expr *ast.BLangMappingConstructorExpr) (expressionEffect, bool)
func mappingConstructorExpressionInner(ctx context, curBB *bir.BIRBasicBlock, mapType semtypes.SemType, fields []mappingField, defaults []bir.MappingConstructorDefaultEntry, pos bir.Location) (expressionEffect, bool)
```

The AST constructor path emits mixed key/value and spread entries in order. Preserve the existing inner helper for callers constructing key/value-only mappings; do not force spreads through its static-key representation. Both paths retain the existing construction instruction's inherent type, defaults, and readonly behavior.

## `bir/pretty_print.go`

Existing exported signature remains:

```go
func (p *PrettyPrinter) PrintNewMap(m *NewMap) string
```

Print spread operands distinctly and preserve mixed entry order.

## `bir/codec/serializer.go` and `bir/codec/deserializer.go`

Existing private signatures remain:

```go
func (bw *birWriter) writeInstruction(buf *bytes.Buffer, instr bir.BIRInstruction)
func (br *birReader) readInstruction(varMap map[int32]*bir.BIRLocalVariableDcl) bir.BIRInstruction
```

For a mapping entry, `true` retains the key-operand/value-operand payload. `false` gains a single value-operand payload and reconstructs `MappingConstructorSpreadEntry` rather than panicking. Document this in `bir/codec/SPEC.md`; no new discriminator is needed.

## `runtime/internal/exec/non_terminators.go`

Existing private signature remains:

```go
func execNewMap(ctx *extern.Context, newMap *bir.NewMap, frame *Frame)
```

Expand spread entries with the existing ordered `values.Map` APIs, record actual supplied names, and apply defaults afterwards. Retain atomic inherent-type construction and readonly initialization. No platform access or new values-layer public API is required.

## Helpers added during implementation

These preserve the contracts above and exist only to keep the spread paths from duplicating
logic across packages.

`semtypes`:

```go
func (f Field) Name() string
func (f Field) Type() SemType
func (f Field) Readonly() bool
func (f Field) Optional() bool
func (m *MappingAtomicType) RestInnerVal() SemType
func (m *MappingAtomicType) AllMemberInnerVal() SemType
func MappingFieldTypeAllowed(cx Context, actual, expected SemType) bool
func MappingOfMemberType(env Env, memberTy SemType) SemType
func MappingOfFieldTypes(env Env, fields []MappingFieldInfo) SemType
```

`Field` accessors expose the summary fields `MappingSpreadShape` returns. `RestInnerVal` and
`AllMemberInnerVal` read the rest cell and the union of every member cell of an atom.
`MappingFieldTypeAllowed` exports the existing alternative-selection field rule unchanged.
`MappingOfMemberType` builds the `map<T>` used as a spread operand's contextual type; unlike
`MappingDefinition.Define` it keeps a mutable rest cell for a never member type, so a record
declaring an impossible optional field is still admitted. `MappingOfFieldTypes` builds the
closed mapping type used instead when the operand is itself a constructor whose keys are known.

`semantics/internal/common/mapping_spread.go` (new), shared by the type resolver and the
semantic analyzer so that both decide from the same resolved description:

```go
type MappingConstructorEntry struct {
    Spread     bool
    Name       string
    Type       semtypes.SemType
    Shape      semtypes.MappingSpreadShape
    Pos        diagnostics.Location
    IsReadonly bool
}

type MappingConstructorDiagnostic struct {
    Message string
    Pos     diagnostics.Location
}

func MappingConstructorEntries(ctx *context.CompilerContext, cx semtypes.Context, expr *ast.BLangMappingConstructorExpr) ([]MappingConstructorEntry, bool)
func HasSpreadField(expr *ast.BLangMappingConstructorExpr) bool
func CheckMappingConstructorKeys(cx semtypes.Context, entries []MappingConstructorEntry) (MappingConstructorDiagnostic, bool)
func MappingSpreadContextualType(ctx *context.CompilerContext, env semtypes.Env, target *semtypes.MappingAtomicType, operand ast.BLangExpression) semtypes.SemType
func CheckMappingConstructorAgainstTarget(cx semtypes.Context, entries []MappingConstructorEntry, target *semtypes.MappingAtomicType, defaults []string, pos diagnostics.Location) (MappingConstructorDiagnostic, bool)
```

Collision queries use `MappingConstructorEntry.Type`, the operand's own semantic type, never the
summarised `Shape`. `CheckMappingConstructorAgainstTarget` is used silently to filter inherent
type alternatives and with reporting to diagnose the selected one.

`semantics/internal/types/type_resolver.go` adds the private
`resolveMappingSpreadOperand`, `checkMappingConstructorKeys`, `resolveMappingKeyWithValueType`
and the `inferredMappingShape` accumulator, which merges repeated names instead of emitting a
mapping atom with a repeated name.

`semantics/internal/analysis/semantic_analyzer.go` adds the private
`analyzeMappingSpreadFields`.

`birgen/bir_gen.go` extracts the private `mappingKeyValueEntry` and `newMapInstruction`, shared
by the AST constructor path and the key/value-only inner helper.

# Tests

## Test organization

- Add compiler/runtime corpus sources under `corpus/bal/`, with `*-v.bal`, `*-e.bal`, and applicable `*-p.bal` names without leading-zero numeric components.
- Use `@output`, `@error`, and `@panic` markers. Error assertions require proper source locations and ordinary compiler diagnostics, not identical jBallerina wording.
- Add/update AST, CFG, desugared, BIR, and integration goldens through the existing stage test drivers. Include BIR roundtrip execution.
- Prefer source-level corpus cases over semtypes-only Go tests, including narrowed-type cases. Do not invent unreachable type fixtures merely to exercise an internal representation.
- Run comparison cases with jBallerina where supported. Its concrete record/map dispatch limitations do not restrict the required semantic union/intersection support here.

## Operands and basic construction

- Corrected issue example with a closed source record and a specific `z` field.
- Single map spread, single record spread, multiple disjoint closed-record spreads, nested spread constructors, and spread-returning function calls.
- Empty record, empty `map<never>`, and empty values whose broad map type still permits arbitrary keys.
- Mapping type aliases, readonly mapping operands, and error-valued maps (`map<any|error>`, not only `map<any>`).
- Reject scalar/list/error operands and unions with a nonmapping branch.
- Spread operand symbol errors, expression errors, and isolation violations receive normal diagnostics.
- Constant-context constructors containing spreads are not silently accepted by skipping their operands; retain the language's constant-expression legality.

## Duplicate names

- Required spread member against specific field in both orders.
- Optional inhabited spread member against specific field in both orders, including an operand whose current value omits the member.
- Required/optional combinations across two spreads; also a third spread to ensure checks are not limited to adjacent fields.
- Maps against specific fields and against maps/records.
- Open-record named/rest collisions, rest/rest collisions, and closed empty-record spreads that cannot collide.
- Two separate spreads with the same possible key but disjoint member value types still fail.
- Identifier, shorthand, string-literal, escaped identifier, Unicode escape, and empty-string keys use decoded equality.
- Original issue example using `map<int> first` is a compile-error regression.

## Empty fields and exclusions

- Closed `record {| never x?; |}` spread alongside specific `x` succeeds.
- Open record with `never x?` and inhabited rest alongside specific `x` succeeds.
- Shared excluded names between two spreads do not collide; other inhabited shared names still do.
- Optional `never|never`, `[never, int]`, and a nested record with a required `never` member are treated as semantically impossible values.
- Exclusions remain effective in inferred rest-bearing records, not only during duplicate detection.
- Excluded fields do not satisfy required target fields.

## Unions, intersections, and narrowing

- Collision permitted by any inhabited union alternative is rejected; exclusion by only one alternative is insufficient.
- A union whose alternatives all exclude the conflicting key is accepted.
- Fields required in only some union alternatives become optional in the inferred result.
- Fields required in every inhabited union alternative remain required, with unioned value types.
- Intersection where rest permits a name not explicitly declared in one constituent.
- Intersection where an explicit exclusion removes a name permitted by another constituent/rest.
- Intersection of optional `int x?` and optional `string x?` permits no `x` and can coexist with specific `x`.
- Intersection with inhabited overlapping member values still collides.
- Uninhabited union alternatives do not add possible keys or weaken guaranteed presence.
- Wholly uninhabited operands follow existing unreachable-expression behavior, not empty-map behavior.
- Source-expressible negative narrowing constraints remove impossible keys/alternatives without false positives or internal panics.
- Recursive mapping member types terminate and preserve field-level collision behavior.

## Contextual target compatibility

- Matching/mismatching declared target member types, including a member permitted by only one source union alternative.
- Extra named source member accepted/rejected by target rest independently of source rest compatibility.
- Source rest accepted/rejected by target rest independently of named members.
- Source rest contributing to a declared target name with a narrower type; an explicit source exclusion suppresses that contribution.
- Inhabited source rest rejected by a closed target even when its runtime value has no extra keys.
- Source `never` rest accepted; valid widening such as `int...` to `(int|string)...` accepted.
- Exclusive record syntax with an explicit rest descriptor is treated according to that rest type.
- Required fields supplied by required spread members succeed; optional members, map possibilities, and excluded members do not establish presence.
- Nested constructor operands receive contextual member typing, including numeric/context-sensitive expressions supported by the existing resolver.
- Contextual union selection with spread-provided required members, extra members, optionality, and defaults; missing/ambiguous candidates report normal diagnostics.
- Names copied from a spread are not rejected by restrictions applying only to explicitly written identifier keys for target rest fields.

## Inherent-type inference and updates

- Verify individually typed named members, rather than only output equality.
- Verify required/optional member access and assignments.
- Verify union member-value contributions and optionality across alternatives.
- Verify map constraints, open-record rests, extra named fields, and their combined inferred rest type.
- Verify empty-field exclusions survive inference when rest is inhabited.
- Use positive and negative assignments/type tests to demonstrate the inferred shape.
- Use a widened mutable alias to verify valid updates succeed and incompatible updates panic according to the actual inherent type.
- Cover nested mapping-valued members and ensure spreading does not flatten their types.

## Defaults and readonly

- Spread supplies a required target field and remaining defaultable fields initialize.
- Required and present optional spread members suppress their defaults.
- Absent optional spread members receive target defaults.
- Default functions run exactly once only when needed; side effects demonstrate evaluation order.
- Target aliases retain defaults.
- Readonly contextual construction succeeds with compatible values and rejects incompatible nested mutable values.
- Source readonly field qualification does not automatically make the new destination field readonly; readonly member value types remain readonly.
- Spreading alone does not freeze or mutate the source mapping.
- Inherent-type/readonly mutation panics are tested through legal widened aliases where expressible.

## Runtime ordering and transport

- Each spread operand executes exactly once, interleaved correctly with specific-field expressions.
- Desugaring of nested effectful operands does not reorder earlier evaluations behind later hoisted work.
- Operand panic stops subsequent operand/default evaluation and retains the correct source stack location.
- Destination insertion order follows field order and each spread's mapping iteration order, including empty spreads.
- A later expression mutating an earlier spread source agrees with evaluate-operands-then-expand reference semantics; compare against jBallerina.
- Destination is a distinct mapping; adding/removing/replacing a destination entry does not modify the source.
- Nested mutable members remain shared, proving shallow rather than deep copying.
- Mixed entry kinds, empty spreads, defaults, readonly state, and inferred inherent types survive BIR serialization/deserialization and execute identically.
- Existing nonspread mapping construction behavior and computed-name unsupported diagnostics remain unchanged.

## jBallerina reference suite

Inspect/adapt scenarios from `tests/jballerina-unit-test/src/test/resources/test-src/expressions/mappingconstructor/`:

- `spread_op_field.bal`
- `spread_op_field_code_analysis_negative.bal`
- `spread_op_field_semantic_analysis_negative.bal`
- `mapping_constructor_duplicate_fields.bal`
- `mapping_constructor_infer_record.bal`
- `readonly_field.bal`

These are behavioral references. In particular, preserve the optional-empty-field, excluded-open-field, rest compatibility, inferred optionality, and default-field cases. Add the semantic union/intersection/narrowing regressions separately instead of inheriting jBallerina's operand-kind dispatch limitations.
