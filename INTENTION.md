## Goal

- **In nodebuilder we allow users to either parse the st recovering or without it. With recovering after getting a syntax error we give a bad node. We need to make it possible to run the later phases with these bad nodes**
  - All syntax errors reported in nodebuilder should get a bad node.
  - Any later phase currently reporting syntax errors will become semantic errors.
- In test runner we should keep driving it forward gathering as many errors as possible to validate it is not crashing.
  - Continue through symbol resolution, top level and inner node type resolution, and semantic analysis despite syntax errors and cascading semantic errors.
  - These phases must not panic or report internal errors because of bad nodes or incomplete results from earlier phases.

## Non goals

- Introducing new recovering mechanism for front end or adding new public APIs
- Changing the error gates in CLI/project compilation.
- Supporting cfg building, cfg analysis, desguar or bir gen with bad nodes.

## Limitation (if any)

- Presence of bad nodes will cause transient failures for example referring a malformed function declaration, could get a function not found error this is expected. This is to be expected.
- This support is limited to only symbol resolver, type resolver and semantic analysis.
  - The stages outside this scope may still crash with internal error if you pass in a bad node.
- The isolation of function body errors applies to functions with valid, independently resolved signatures. Inferred anonymous function return types depend on their bodies.

## Design

### Nodebuilder recovery

- Abandon bad identifiers. Replace the smallest enclosing construct that has a supported bad-node category and cannot be meaningfully processed without that identifier.
  - A malformed variable reference or invocation name/alias becomes a bad expression.
  - A malformed field-access name becomes a bad expression for that access.
  - A malformed named-argument name makes the containing invocation a bad expression.
  - A malformed local variable name makes the declaration a bad statement.
  - A malformed type-reference name/alias becomes a bad type.
  - Other malformed names, including annotation names and XML name patterns, invalidate their enclosing construct rather than surviving as bad identifiers.
- Anything wrong in the function signature should invalidate it: bad default, bad param, bad name, bad return type etc.
  - Malformed function annotations or qualifiers also invalidate the declaration.
  - Replace the whole named function declaration with a bad top level node.
  - Apply the same rule to anonymous functions, replacing the whole anonymous-function expression with a bad expression.
  - A malformed body alone does not invalidate a function with a valid name/signature, annotations and qualifiers. Recover within the body.
  - This is nodebuilder handling malformed syntax/AST construction, not a requirement to detect semantic errors such as unknown type names during nodebuilding.
- Bad expressions, statements, types and top level nodes are opaque leaves; they do not retain malformed identifiers or children.
- Ensure malformed syntax is diagnosed in nodebuilder, including cases that currently produce bad identifiers without any syntax diagnostic. Legitimate ignore identifiers such as the name in `int _ = 1;` remain valid.
- Fix recovering nodebuilder paths that currently panic, discard malformed syntax without a bad node, or retain an ordinary node after reporting a syntax error.

### Symbol resolution

- Ignore bad nodes and resolve the surviving AST. A bad node has no identifiers or children to resolve.
  - Skip bad top level nodes and continue with the next module level declaration.
  - Skip bad statements/expressions/types and continue resolving surviving siblings.
  - Parent operations that inspect children directly, such as type-inclusion resolution, must also recognize bad nodes before consuming them.
- Do not allocate or look up malformed names. Replacing malformed constructs in nodebuilder removes the need to unwind symbol resolution on bad identifiers.
- Preserve symbol and signature information for functions whose bodies contain bad nodes, so references from other functions can still resolve normally.
- References to discarded declarations may report semantic errors. Later phases must safely handle the resulting unresolved ordinary references.

### Type resolution and semantic analysis

- Error at expression will unwind up to statement level and move on to the next statement.
- Declaration-level failures abandon the affected module level declaration and move on to the next module level decl.
- Reporting an error and signalling failure are separate operations.
  - On a bad node, signal failure without reporting another error.
  - Ordinary semantic errors, including cascading errors, can still report diagnostics and signal failure.
- Continue processing independent statements/declarations even when diagnostics already exist.
- Missing symbols, unresolved types and ordinary subtrees left unprocessed by earlier failures must not be treated as internal errors. Skip checks whose prerequisites are unavailable and continue processing independent nodes.
- A body failure must not invalidate an independently resolved function signature.
- Do not assume all phases already implement the same unwinding protocol. Symbol resolution uses traversal; type resolution and semantic analysis must propagate failures to their recovery boundaries.

### Testing

- Recovery tests are a special case in corpus, under `corpus/recovery/`, not part of `corpus/bal/`.
  - Discover these separately, similar to project corpus tests. Normal corpus runners remain unchanged.
- Identify test cases with syntax errors / create new tests that give syntax errors for each scenario.
- Build the ast with node builder recovering, ignore any syntax error reported when deciding whether to run the next phase.
  - Keep the syntax diagnostics available for validation; do not suppress their reporting.
- Pass the it through the phases. Making reasonable assumptions about the cascading diagnostics, objective is for frontend phases to run without crahsing.
  - Add phases gradually, ultimately driving every supported phase through semantic analysis even after cascading semantic errors.
  - Dump the AST after the last enabled phase and compare it against a golden.
  - Use normal corpus diagnostic goldens and `@error` marker validation for syntax errors and cascading semantic errors.
  - Fail on panics/internal errors and ensure the runner reaches the last enabled phase despite diagnostics.
  - The AST dump and normal diagnostic validation are sufficient; do not add separate symbol/scope/type invariant validators for recovery tests.
  - Cover missing expression operands, malformed names, malformed nested types and each function declaration invalidation case, including anonymous functions.
  - Bad expressions/statement may cascade errors withing that function body after that.
  - Bad module level node may cascade errors to all references to that.
  - But a bad expression/statement within a function with a valid, independently resolved signature can't cascade in to errors in other functions (including those referring to it).
    - This is becuase function signature is correct.
- Do not run CFG building/analysis, desguar or bir gen on these recovered erroneous ASTs.
