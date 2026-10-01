# Frontend recovery through semantic analysis

## Goals

- Make recovered ASTs safe to pass through symbol resolution, top-level type resolution, inner-node type resolution, and semantic analysis despite syntax errors and cascading semantic errors.
- Represent malformed syntax with opaque bad expressions/actions, statements, types, or top-level nodes, never malformed identifiers retained inside ordinary nodes.
- Continue processing independent declarations, class/service members, and statements, gathering diagnostics without panics or internal errors.
- Preserve independently resolved function signatures when only their bodies fail.

## Non-goals

- Introducing another parser recovery mechanism or new public compilation entry points.
- Changing diagnostic gates in CLI/project compilation or normal corpus runners.
- Running CFG generation/analysis, desugaring, BIR generation, or execution on recovered erroneous ASTs.
- Making otherwise unsupported constructs that report `Unimplemented` recoverable through these phases.
- Eliminating expected cascading diagnostics from references to discarded declarations.
- Adding separate symbol, scope, or type invariant validators to recovery tests.

## Success criteria

- Each syntax error reported by nodebuilder has an enclosing bad node representing the malformed construct.
- Bad nodes contain no malformed identifiers or children. Legitimate ignore identifiers, such as `int _ = 1;`, remain valid.
- All supported frontend phases run through semantic analysis even when earlier phases report diagnostics.
- Bad nodes themselves do not cause duplicate diagnostics; ordinary semantic errors may still produce cascading diagnostics.
- No supported recovery scenario panics or reports an internal error.
- Malformed class/service members do not invalidate their containing class/service or valid siblings.
- A body error does not invalidate a valid, independently resolved named-function, method, or explicit-signature lambda type, or cause errors in other functions solely by invalidating that signature.
- Recovery corpus tests validate the final AST and accumulated diagnostics and verify that the final enabled phase was reached.

# Design

## Nodebuilder

Use the existing recovering nodebuilder entry point. Parser diagnostics remain available and are reported through the existing diagnostic machinery; recovery must not suppress them.

Replace the smallest enclosing construct with a supported bad-node category when malformed syntax prevents meaningful processing:

| Malformed construct | Replacement |
| --- | --- |
| Variable reference or invocation name/alias | Bad expression |
| Field-access name | Bad expression for that access |
| Named-argument name | Bad expression for the invocation |
| Local-variable name | Bad statement for the declaration |
| Type-reference name/alias | Bad type |
| Named-function signature, annotation, qualifier, or name | Bad top-level node for the declaration/member |
| Anonymous-function signature, annotation, qualifier, or name | Bad expression for the lambda |
| Class/service member that cannot be represented normally | Bad top-level node retained by the containing class/service |
| Other malformed names, including annotation names and XML name patterns | Bad node for the smallest enclosing supported construct |

Function signature invalidation includes malformed parameters, defaults, return types, names, annotations, and qualifiers. It concerns malformed syntax, not semantic errors such as unknown type names.

A malformed body alone does not invalidate a function with a valid signature. Recover within block bodies at statement/expression boundaries; preserve the function when its expression body becomes a bad expression. Class/service methods follow the same rule.

Class/service member collection must recover before adding members to ordinary typed collections. Malformed fields, methods, resource methods, initializers, or inclusions must not acquire names, map keys, symbols, or ordinary inclusion entries. Class/service-level syntax errors may invalidate the containing declaration; errors confined to members may not.

Repair recovering paths that panic, silently discard malformed syntax, or report syntax errors while retaining only ordinary nodes. Malformed identifier detection must report syntax diagnostics even where it previously returned a bad identifier silently.

Checks for duplicate members, invalid `distinct` targets, and missing module-variable initializers are semantic diagnostics, not syntax errors requiring bad nodes. Syntax diagnostics currently reported by later phases likewise become semantic diagnostics.

## Bad top-level storage

`BLangCompilationUnit.TopLevelNodes` continues to hold source declarations, including bad top-level nodes.

Add a `BadTopLevelNodes` slice to both `BLangPackage` and `classDefnBase`. Package assembly appends bad declarations to the package slice and continues with ordinary declarations. Class/service construction appends bad members to the base slice and continues with valid members.

The symbol resolver, type resolver, and semantic analyzer do not enumerate these slices. AST printing includes them as opaque leaves so recovery goldens retain evidence of malformed declarations/members. No new bad-member node category or unified member-container redesign is required.

## Symbol resolution

Symbol resolution remains traversal-based:

- Skip bad top-level nodes, statements, expressions, and types.
- Continue resolving surviving siblings.
- Operations inspecting children directly, including inclusion handling, recognize bad nodes before consuming them.
- Never allocate or look up malformed names.
- Preserve function symbols and signatures independently of body failures.
- References to discarded declarations may report semantic errors and remain unresolved. Later phases must safely handle those ordinary unresolved references.

## Type resolution and semantic analysis

Reporting a diagnostic and signalling failure are separate operations. A bad node signals failure without reporting another diagnostic. Ordinary semantic errors can both report diagnostics and signal failure.

```text
resolve/analyze declaration:
    if declaration prerequisites are unavailable:
        abandon affected declaration/member
        continue with next independent declaration/member

resolve/analyze statement:
    expression or binding failure -> abandon statement
    continue with next sibling statement

resolve/analyze compound statement:
    condition, binding, or pattern failure -> abandon whole statement
    valid body -> recover independently at each inner statement

consume ordinary unresolved subtree:
    unavailable prerequisite -> skip dependent checks
    never treat missing symbol/type/partial result as an internal error
```

Do not terminate a supported phase merely because diagnostics already exist. Do not fabricate successful types or symbols to hide failures. Preserve the existing separation between top-level function-signature resolution and body resolution, and make signature resolution fail if any ordinary or rest-parameter resolution fails.

Semantic analysis must not re-enter subtrees whose prerequisite resolution was abandoned. Independent siblings remain eligible for analysis. Each phase uses its own appropriate failure propagation; no shared panic-based unwinding protocol is assumed.

## Lambda-local signature preservation

The explicit-signature lambda resolver is the only expression-level exception for body isolation. Generic invocation, binary-expression, and other expression failure propagation remain unchanged.

```text
resolveLambdaFunctionExpr(lambda):
    if lambda requires inferred-signature resolution:
        return resolveInferredLambdaFunctionExpr(lambda)

    callableType = resolveFunctionSignature(lambda.function)
    if signature resolution failed:
        return failure

    establish lambda-local binding/capture context
    if block body:
        resolve statements with normal statement recovery
    if expression body:
        mark body wrapper unresolved
        resolve expression against declared return type
        if successful:
            mark body wrapper complete with Never
        otherwise:
            retain diagnostics; abandon body, not callableType

    finalize captures, including those discovered before body failure
    restore enclosing capture context
    return callableType and expression effect successfully

resolveInferredLambdaFunctionExpr(lambda):
    body failure -> failure
    infer final callable type only from successfully resolved prerequisites
```

For explicit-signature expression bodies, the body wrapper's zero determined type denotes incomplete resolution; `Never` denotes completion. Check the wrapper, not an individual expression child that may have been partially resolved. Do not overwrite unresolved children with `Never` to pretend resolution succeeded.

Lambda-specific semantic analysis retains independent signature/default-parameter checks and skips body-dependent checks for an unresolved expression body. Lambda-specific isolation traversal also skips that body. This guard must apply before any analysis initialization that independently walks the body. It must not skip independent signature checks simply by returning early from all lambda analysis.

Inferred-return anonymous functions are excluded from body/signature independence because their callable type depends on the body. Ordinary statement-level recovery still applies inside block bodies.

## Recovery corpus execution

Discover tests separately under `corpus/recovery/`, outside `corpus/bal/`. Do not alter normal corpus discovery or its diagnostic gates.

```text
register source and parse
build recovered compilation unit
resolve symbols
assemble package, retaining bad top-level declarations
resolve top-level types
resolve inner-node types
analyze semantics
assert final enabled phase reached
reject panic or internal-error diagnostics
print AST and accumulated diagnostics
compare goldens and validate @error markers
```

Ignore syntax and ordinary semantic diagnostics only when deciding whether to advance; retain their reporting and validation. Genuine setup failures, panics, internal errors, and unsupported constructs are not ordinary recoverable diagnostics.

During incremental implementation the runner may stop at an explicitly enabled earlier supported phase; the completed feature always drives recovery tests through semantic analysis. Never enable CFG or later stages for these tests.

Reuse normal exact diagnostic-golden comparison and bidirectional `@error` line-range validation. Include cascading semantic errors in the expected diagnostics. Store the final AST and diagnostic goldens alongside their recovery test under `corpus/recovery/`, with distinguishable `.ast.txt` and `.diagnostics.txt` suffixes. Support the existing `-update` corpus convention.

# API changes

Public compilation entry-point signatures remain unchanged. The exported bad-identifier type and constructor are removed. The following lists the storage, entry points, and private recovery contracts; internal dispatchers using these contracts must be updated without introducing new public entry points.

## `ast/bad_nodes.go`

- Remove exported `type BLangBadIdentifier struct` and its methods.
- Remove exported `func NewBLangBadIdentifier(pos diagnostics.Location, value, originalValue string, isLiteral bool) *BLangBadIdentifier`.
- Retain `BLangBadTopLevelNode`, `BLangBadStmt`, `BLangBadExprOrAction`, and `BLangBadTypeNode` as opaque leaves.

## `ast/ast.go`

- Add `BadTopLevelNodes []*BLangBadTopLevelNode` to exported `BLangPackage` and private `classDefnBase`.
- Retain existing ordinary declaration/member fields and constructors. Constructors initialize the new slices according to the repository's slice conventions; existing maps remain initialized.
- No new public accessor methods are required.

## `ast/pretty_printer.go` and `ast/walk.go`

- Existing private signatures remain unchanged:
  - `func (p *PrettyPrinter) printPackage(node *BLangPackage)`
  - `func (p *PrettyPrinter) printClassDefinition(node *BLangClassDefinition)`
  - `func (p *PrettyPrinter) printService(node *BLangService)`
  - `func walkClassDefnBody(v Visitor, b *classDefnBase)`
- Printing includes the new bad-node collections deterministically. Supported semantic traversals do not process these collections.
- Remove bad-identifier dispatch/printing support; retain opaque handling for supported bad categories.

## `nodebuilder/mod.go`

Existing signatures remain unchanged:

- Exported `func GetCompilationUnit(cx *context.CompilerContext, syntaxTree *st.SyntaxTree) *ast.BLangCompilationUnit`.
- Exported `func GetRecoveredCompilationUnit(cx *context.CompilerContext, syntaxTree *st.SyntaxTree) *ast.BLangCompilationUnit`.
- Exported `func ToPackageFromCompilationUnits(cx *context.CompilerContext, compilationUnits []*ast.BLangCompilationUnit) *ast.BLangPackage`.
- Private `func addCompilationUnitNodesToPackage(cx *context.CompilerContext, pkg *ast.BLangPackage, compilationUnit *ast.BLangCompilationUnit)`.

Recovery uses the existing recovered entry point. Assembly retains bad top-level nodes instead of reporting an internal error and stopping.

## `nodebuilder/node_builder.go`

- Remove private `func (n *nodeBuilder) badIdentifier(token st.Token) *ast.BLangBadIdentifier`.
- Existing private signatures remain unchanged:
  - `func (n *nodeBuilder) createIdentifierNodeFromToken(pos diagnostics.Location, token st.Token) ast.IdentifierNode`.
  - `func (n *nodeBuilder) transformModulePart(modulePartNode *st.ModulePart) ast.BLangNode`.
  - `func (n *nodeBuilder) collectClassDefnMembers(memberNodes st.NodeList[st.Node]) classDefnMembers`.
- Identifier construction is called only after the enclosing construct's names have been validated; malformed names trigger enclosing bad-node replacement, never a substitute ordinary identifier.
- Extend private `classDefnMembers` with `BadTopLevelNodes []*ast.BLangBadTopLevelNode` and carry it into `classDefnBase` during class/service construction.
- Existing bad-node factory signatures stay unchanged. Construct-specific transforms adopt the recovery boundaries above.

## `semantics/internal/symbols/symbol_resolver.go`

- No new public or private entry points are required. Existing resolver/traversal signatures stay unchanged.
- Dispatch and direct-child consumers ignore bad nodes, and surviving declarations/members remain resolvable despite existing diagnostics.

## `semantics/internal/types/type_resolver.go`

Existing private signatures remain unchanged:

- `func resolveInvokableSignature(t typeResolver, fn common.FunctionDecl, fnSym model.FunctionSymbol, requiredParams []ast.BLangVariable, depth int) (semtypes.SemType, bool)`.
- `func resolveFunctionSignature(t typeResolver, fn *ast.BLangFunction, depth int) (semtypes.SemType, bool)`.
- `func resolveBlockStatements(t typeResolver, chain *binding, stmts []ast.StatementNode) statementEffect`.
- `func resolveLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool)`.
- `func resolveInferredLambdaFunctionExpr(t typeResolver, chain *binding, e *ast.BLangLambdaFunction, expectedType semtypes.SemType) (semtypes.SemType, expressionEffect, bool)`.

Bad-node dispatch reports failure instead of successful `Never` resolution. Signature resolution propagates both ordinary and rest-parameter failures before consuming their types or updating signature symbols. Block resolution continues siblings. Explicit-signature lambda resolution follows the lambda-local prototype; inferred-signature resolution retains body-dependent failure.

Existing syntax-diagnostic helper contracts in this phase become semantic-diagnostic contracts; no call may continue reporting a syntax diagnostic here.

## `semantics/internal/analysis/semantic_analyzer.go`

- Existing analyzer entry-point and visitor signatures remain unchanged.
- Add private `func lambdaExpressionBodyUnresolved(fn *ast.BLangFunction) bool`, checking whether an expression-body wrapper has a zero determined type.
- Lambda analysis and nested-lambda isolation traversal use that predicate before body-dependent traversal, while preserving independent signature/default checks.
- Other analysis paths honor statement/declaration failure boundaries and unavailable prerequisites.
- Existing syntax-diagnostic helper contracts become semantic-diagnostic contracts; no syntax diagnostics are reported by this phase.

## `corpus/recovery_test.go` (new)

- Add test-only entry point `func TestRecovery(t *testing.T)` (exported by Go naming convention, not a production compilation API).
- Add private runner `func runRecoveryPipeline(env *context.CompilerEnvironment, cx *context.CompilerContext, langlibs *langlib.Symbols, inputPath string, content string) (*testphases.PipelineResult, error)`.
- The runner's success return guarantees execution through semantic analysis, not absence of syntax/semantic diagnostics. It returns an error for setup failure or an internal-error diagnostic; panics fail the test.
- Use the existing corpus update flag and diagnostic/marker utilities. Normal `testphases.RunPipeline` and `RunPipelineWithContent` signatures and behavior remain unchanged.

# Tests

- Missing unary/binary operands, malformed invocations, accesses, and nested expressions produce bad expressions and do not crash supported phases.
- Malformed variable names, invocation aliases, named arguments, annotation names, XML name patterns, and type-reference names/aliases produce enclosing bad nodes and syntax diagnostics.
- Legitimate ignore identifiers remain valid; silently malformed identifier cases now report diagnostics.
- Malformed nested types and inclusions do not trigger unchecked assertions or symbol lookups.
- Named-function invalidation covers names, ordinary/rest parameters, defaults, return types, annotations, and qualifiers.
- Syntactically valid ordinary/rest parameters with unresolved types fail signature resolution safely; an independent valid declaration still resolves.
- Anonymous-function invalidation covers equivalent signature cases.
- Body-only errors preserve independently resolved signatures for named functions, methods, and explicit-signature lambdas.
- Valid functions referencing those named functions/methods do not gain errors solely because a body failed.
- Explicit-signature lambdas retain callable type and signature metadata after expression-body failure; later statements can reference them.
- Lambda capture context is restored and captures discovered before failure are retained; nested lambdas and isolation checking do not walk unresolved bodies.
- Independent lambda signature/default semantic diagnostics remain available despite expression-body failure.
- Inferred-return anonymous functions fail when their body cannot supply a valid return type.
- A malformed class/service field, method, resource method, initializer, or inclusion is retained as a bad member; valid siblings and the containing declaration survive.
- Package assembly retains bad declarations and continues assembling valid declarations.
- Duplicate members, invalid `distinct` targets, missing module-variable initializers, and former later-phase syntax diagnostics are classified as semantic errors.
- Bad nodes signal failure without duplicate diagnostics.
- Failed compound-statement conditions/bindings/patterns skip dependent bodies and resume outer siblings.
- Failures inside valid bodies resume inner sibling statements.
- References to discarded declarations and ordinary subtrees skipped by earlier failures safely yield or retain expected cascading diagnostics.
- Each recovery test reaches semantic analysis despite diagnostics, rejects internal errors/panics, and compares final AST/diagnostic goldens plus `@error` markers.
- Normal corpus discovery and CLI/project compilation retain their existing error gates.
- Recovery execution never invokes CFG, desugaring, BIR generation, or interpretation.
