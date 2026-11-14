# Supported language features (subset 10)

**Supported Ballerina code:** see [corpus/bal](../../corpus/bal)—the [corpus/bal/subset10](../../corpus/bal/subset10) directory contains the tests and examples that define what is supported in this subset.

## Module-level declarations

- [Import declarations](https://ballerina.io/spec/lang/master/#import-decl)
- [Function definition](https://ballerina.io/spec/lang/master/#function-defn)
  - Supports [`required-params`](https://ballerina.io/spec/lang/master/#required-params), [`defaultable-params`](https://ballerina.io/spec/lang/master/#defaultable-params), [`included-record-param`](https://ballerina.io/spec/lang/master/#included-record-param) and [`rest-param`](https://ballerina.io/spec/lang/master/#rest-param) in the signature
  - Supports the [`isolated`](https://ballerina.io/spec/lang/master/#isolated-qual) function qualifier (see [isolated functions](https://ballerina.io/spec/lang/master/#isolated_functions))
  - Supports dependently-typed functions using `typedesc` parameters with the [`<>` inferred default](https://ballerina.io/spec/lang/master/#inferred-typedesc-default)
- [Constant declarations](https://ballerina.io/spec/lang/master/#module-const-decl)
- [Annotation declarations and attachments](https://ballerina.io/spec/lang/master/#section_9.1)
  - Supports source-only, runtime, marker and repeated annotations
  - Supports annotation values evaluated from constant and runtime expressions
  - Supports `service remote function` annotations on remote methods of service classes and service declarations, including `object function` annotation compatibility
- [Module variable declarations](https://ballerina.io/spec/lang/master/#module-var-decl)
- [Module listener declarations](https://ballerina.io/spec/lang/master/#section_8.3.1)
- [Service declarations](https://ballerina.io/spec/lang/master/#section_8.3.2)
- [Type definition](https://ballerina.io/spec/lang/master/#module-type-defn)
- [Enum declarations](https://ballerina.io/spec/lang/master/#module-enum-decl)
- [Class definition](https://ballerina.io/spec/lang/master/#section_8.6)
  - Supports `client`, `service`, `isolated`, `readonly` and `distinct` [`class-type-quals`](https://ballerina.io/spec/lang/master/#class-type-quals)
  - Supports `object-field`, `method-defn` and `object-type-inclusion` members
  - Supports [`remote-method-defn`](https://ballerina.io/spec/lang/master/#remote-method-defn) for client classes and `resource-method-defn` for service classes

## Statements

- [Assignment](https://ballerina.io/spec/lang/master/#assignment-stmt)
  - See supported [`lvexpr`](#expressions)
- [Destructuring assignment statement](https://ballerina.io/spec/lang/master/#destructuring-assignment-stmt)
  - Only supports [`wildcard-binding-pattern`](https://ballerina.io/spec/lang/master/#wildcard-binding-pattern)
- [Compound Assignment](https://ballerina.io/spec/lang/master/#compound-assignment-stmt)
  - See supported [binary operators](#operators)
- [Break](https://ballerina.io/spec/lang/master/#break-stmt)
- [Continue](https://ballerina.io/spec/lang/master/#continue-stmt)
- [Lock statement](https://ballerina.io/spec/lang/master/#lock-stmt)
- [Call](https://ballerina.io/spec/lang/master/#call-stmt)
- [If/else](https://ballerina.io/spec/lang/master/#section_7.18)
- [While](https://ballerina.io/spec/lang/master/#while-stmt)
- [Local variable declarations](https://ballerina.io/spec/lang/master/#local-var-decl-stmt)
- [Return](https://ballerina.io/spec/lang/master/#return-stmt)
- [Panic](https://ballerina.io/spec/lang/master/#panic-stmt)
- [Foreach](https://ballerina.io/spec/lang/master/#section_7.21.1)
  - Currently only supports range, list, map, string and XML subtypes, and [iterable objects](https://ballerina.io/spec/lang/master/#section_5.8.2)
- [Match statement](https://ballerina.io/spec/lang/master/#match-stmt)
  - Currently only supports [const-pattern](https://ballerina.io/spec/lang/master/#const-pattern) and [wildcard-match-pattern](https://ballerina.io/spec/lang/master/#wildcard-match-pattern)
- [Named workers](https://ballerina.io/spec/lang/master/#section_7.3.2) ([`named-worker-decl`](https://ballerina.io/spec/lang/master/#named-worker-decl))
  - Only in a `block-function-body`, with optional annotations and return type
  - [`wait`](https://ballerina.io/spec/lang/master/#wait-action) on a worker name has the worker's return type as its eventual type
  - Message passing (send, receive, flush), `fork`, `transactional` workers and `on fail` are not supported

## Expressions

- [Literal](https://ballerina.io/spec/lang/master/#literal)
  - Currently support `nil-literal`, `boolean-literal`, `numeric-literal` and `string-literal` only
- [lvexpr](https://ballerina.io/spec/lang/master/#lvexpr)
- [`Call`](https://ballerina.io/spec/lang/master/#call-expr)
- [Method call](https://ballerina.io/spec/lang/master/#method-call-expr)
- [Client remote method call action](https://ballerina.io/spec/lang/master/#client-remote-method-call-action)
- [Client resource access action](https://ballerina.io/spec/lang/master/#client-resource-access-action)
- [Start action](https://ballerina.io/spec/lang/master/#start-action)
- [Wait action](https://ballerina.io/spec/lang/master/#wait-action)
- [Error constructor](https://ballerina.io/spec/lang/master/#error-constructor-expr)
  - Detail fields that are not given as named arguments take the default values of the detail record type, with the following restrictions:
    - When the contextually expected type is a union of error types, named arguments are not used to select a member, so defaults are applied only when the detail type has a single mapping alternative.
    - When the detail type is an intersection of record types, only one of them may supply defaults for the omitted fields. The specification does not define how defaults combine across an intersection.
    - A default is not applied to a field whose type another intersected record narrows, even when the default value belongs to the narrowed type.
- [Checking expression](https://ballerina.io/spec/lang/master/#checking-expr) (`check` and `checkpanic`)
  - Currently `check` is not supported in [defaultable parameter expressions](https://ballerina.io/spec/lang/master/#defaultable-param) or [record field default expressions](https://ballerina.io/spec/lang/master/#record-type-descriptor), even when the operand cannot produce an error.
- [Trap expression](https://ballerina.io/spec/lang/master/#trap-expr)
- [Type cast expression](https://ballerina.io/spec/lang/master/#type-cast-expr)
- [New expression](https://ballerina.io/spec/lang/master/#section_6.8.2)
  - When selecting among multiple candidate object types, resolving the arguments has the following restrictions:
    - An explicitly typed [anonymous function](https://ballerina.io/spec/lang/master/#anonymous-function-expr) nested inside an [anonymous function with inferred parameter types](https://ballerina.io/spec/lang/master/#infer-anonymous-function-expr) is checked from its signature only during candidate selection, so errors in its body that depend on the candidate don't rule that candidate out.
- [List constructor](https://ballerina.io/spec/lang/master/#list-constructor-expr)
  - When selecting among multiple candidate inherent types, resolving the members has the same restrictions as resolving the arguments of a new expression that selects among candidate object types (see New expression above).
- [Mapping constructor](https://ballerina.io/spec/lang/master/#mapping-constructor-expr)
  - When selecting among multiple candidate inherent types, resolving the field values has the same restrictions as resolving the arguments of a new expression that selects among candidate object types (see New expression above).
  - [spread-field](https://ballerina.io/spec/lang/master/#spread-field) is supported
  - Currently a spread-field operand whose static type is not a single mapping type (e.g. a union of mapping types such as `A|B`, or a type narrowed by a negative type test such as `!is`) is not supported
- [XML template expression](https://ballerina.io/spec/lang/master/#xml-template-expr)
  - Supports interpolation in XML content and attributes
  - XML sequence interpolation from a query result is not supported
- [String template expression](https://ballerina.io/spec/lang/master/#string-template-expr)
- [XML navigation expression](https://ballerina.io/spec/lang/master/#xml-navigate-expr)
- [Annotation access expression](https://ballerina.io/spec/lang/master/#annot-access-expr)
- [Anonymous function expression](https://ballerina.io/spec/lang/master/#anonymous-function-expr)
- [Variable reference](https://ballerina.io/spec/lang/master/#variable-reference-expr)
  - Currently `xml-qualified-names` not supported
- [Field access expression](https://ballerina.io/spec/lang/master/#section_6.10)
- [Optional field access expression](https://ballerina.io/spec/lang/master/#optional-field-access-expr)
- [Member access expression](https://ballerina.io/spec/lang/master/#member-access-expr)
- [Unary logical expression](https://ballerina.io/spec/lang/master/#unary-logical-expr)
- [Nil lifted expression](https://ballerina.io/spec/lang/master/#nil-lifted-expr)
- [Relational expression](https://ballerina.io/spec/lang/master/#relational-expr)
- [Equality expression](https://ballerina.io/spec/lang/master/#equality-expr)
- [Logical expression](https://ballerina.io/spec/lang/master/#logical-expr)
- [Conditional expression](https://ballerina.io/spec/lang/master/#conditional-expr)
- Nested expressions (`(expression)`)
- Nested actions (`(action)`)
- [Shift expression](https://ballerina.io/spec/lang/master/#section_6.25)
- [Type test expression](https://ballerina.io/spec/lang/master/#section_6.28)
- [Range expression](https://ballerina.io/spec/lang/master/#section_6.26)
- [Query expressions](https://ballerina.io/spec/lang/master/#query-expr)
  - Supports `from`, `where`, `let`, `join` (including outer join), `order by`, `limit`, `on conflict` and `select` clauses
  - Supports `group by` and `collect` clauses
  - Supports `list` and `map` as `query-constructor-type`

## Operators

- Binary operators
  - Equality ops `==`, `!=`, `===`, `!==`
  - Multiplicative ops `*`, `%`, `/`
  - Bitwise ops `&`, `|`, `^`
  - Relational ops `<`, `<=`, `>`, `>=`
  - Additive ops `+`, `-`
  - Shift ops `<<`, `>>`, `>>>`
  - Logical ops `&&`, `||`
- Unary operators
  - logical `!`
  - numeric ops `+`, `-`
  - bitwise complement `~`


# Spec deviations

These are intentional departures from the [language specification](https://ballerina.io/spec/lang/2024R1/) that we intend to retain, unlike temporary subset restrictions. Comparisons below were validated with jBallerina 2201.13.4.

## Matching jBallerina

- [Implicit `new` expressions](https://ballerina.io/spec/lang/master/#section_6.8.2) accept an applicable contextually expected type that is a union of object types.
  - For each member of the union, we check whether that member's `init` method accepts the passed-in arguments.
  - If exactly one member does, that member is selected as the contextually expected type.
  - The following argument expressions are currently unsupported when the applicable contextually expected type is a union of class types:
    - Calls to dependently-typed functions, including nested calls
    - Query expressions with multiple `from` clauses
    - Query expressions over collections other than lists or mappings
    - Optional XML attribute access
    - Calls with named arguments when the callee's parameter signature is unavailable
    - Anonymous functions containing client resource access actions with non-atomic receivers or ambiguous dispatch

## Deviations from jBallerina

- Implicit `new` expressions reject an applicable contextually expected type containing both `stream` and `object` subtypes; jBallerina can select an applicable class from such a type.
- A [lock statement](https://ballerina.io/spec/lang/master/#lock-stmt) must have exactly one restricted variable (spec allows for none).

# Subset restrictions

## Import declarations

- Only following lang libraries with given methods/types are supported
  - `ballerina/lang.array`
    - `length`
    - `push`
    - `map`
    - `indexOf`
    - `remove`
    - `removeAll`
    - `toStream`
    - `toBase64`
    - `toBase16`
    - `fromBase64`
    - `fromBase16`
  - `ballerina/lang.boolean`
    - `fromString`
  - `ballerina/lang.decimal`
    - `sum`
    - `max`
    - `min`
    - `abs`
    - `round`
    - `quantize`
    - `floor`
    - `ceiling`
    - `fromString`
  - `ballerina/lang.float`
    - `PI`
    - `E`
    - `NaN`
    - `Infinity`
    - `isFinite`
    - `isInfinite`
    - `isNaN`
    - `sum`
    - `max`
    - `min`
    - `avg`
    - `abs`
    - `round`
    - `floor`
    - `ceiling`
    - `sqrt`
    - `cbrt`
    - `pow`
    - `log`
    - `log10`
    - `exp`
    - `sin`
    - `cos`
    - `tan`
    - `asin`
    - `acos`
    - `atan`
    - `atan2`
    - `sinh`
    - `cosh`
    - `tanh`
    - `fromString`
    - `toHexString`
    - `fromHexString`
    - `toBitsInt`
    - `fromBitsInt`
    - `toFixedString`
    - `toExpString`
  - `ballerina/lang.int`
    - `Signed8`
    - `Signed16`
    - `Signed32`
    - `Unsigned8`
    - `Unsigned16`
    - `Unsigned32`
    - `toHexString`
    - `fromString`
    - `fromHexString`
  - `ballerina/lang.map`
    - `length`
    - `keys`
    - `hasKey`
    - `get`
    - `remove`
  - `ballerina/lang.string`
    - `Char`
    - `length`
    - `indexOf`
    - `includes`
    - `substring`
    - `equalsIgnoreCaseAscii`
    - `toLowerAscii`
    - `toUpperAscii`
    - `trim`
    - `toBytes`
    - `fromBytes`
  - `ballerina/lang.error`
    - `message`
    - `cause`
  - `ballerina/lang.value`
    - `clone`
    - `cloneWithType`
    - `fromJsonWithType`
    - `toString`
    - `toBalString`
  - `ballerina/lang.object`
    - `Iterable`
    - `RawTemplate`
  - `ballerina/lang.xml`
    - `Element`
    - `Text`
    - `Comment`
    - `ProcessingInstruction`
    - `XML_NAMESPACE_URI`
    - `XMLNS_NAMESPACE_URI`
    - `space`
    - `lang`
    - `base`
    - `iterator`
    - `length`
    - `concat`
    - `getName`
    - `setName`
    - `getAttributes`
    - `getChildren`
    - `setChildren`
    - `getDescendants`
    - `data`
    - `getTarget`
    - `getContent`
    - `createElement`
    - `createProcessingInstruction`
    - `createComment`
    - `createText`
    - `strip`
    - `elements`
    - `children`
    - `elementChildren`
    - `text`
    - `fromString`
    - `get`
    - `slice`
    - `map`
    - `forEach`
    - `filter`
  - `ballerina/lang.runtime`
    - `sleep`
    - `onGracefulStop`
    - `StopHandler`

## Invocations

### Named and default arguments

- Named arguments and omitted defaultable arguments are supported for function calls when the callee has an atomic function type with a known parameter signature.
  - For a variable declared with `var`, the signature is retained only when its initializer exposes a known atomic function signature. Supported initializers include anonymous function expressions, function/method/remote-method calls that return an atomic function type, references to variables whose signatures are already known, and grouped forms of these expressions. An explicit conversion to an atomic function type also supplies the converted type's signature.
  - Other expressions that merely evaluate to a compatible function value do not retain the parameter names and defaults required for named and default arguments.
- Named arguments and omitted defaultable arguments are supported for method calls and client remote method call actions only when the receiver type has one positive object atom. The receiver may also be `readonly & T`, where `T` has one positive object atom.
  - Type narrowing does not necessarily produce such a receiver type; for example, a receiver that still has multiple object alternatives is unsupported for named and default arguments.

### Other invocation restrictions

- Langlib functions whose type depends on their arguments (for example `value:clone` and `map:remove`) can only be called; they cannot be used as function values.
- Client resource access actions require an atomic client receiver. Dispatch is not supported when more than one resource method matches the access path.
- Method call syntax can be used for calling the following langlib functions:
  - `array:length`
  - `array:push`
  - `array:map`
  - `array:indexOf`
  - `array:remove`
  - `array:removeAll`
  - `array:toStream`
  - `array:toBase64`
  - `array:toBase16`
  - `decimal:sum`, `decimal:max`, `decimal:min`, `decimal:abs`, `decimal:round`, `decimal:quantize`, `decimal:floor` and `decimal:ceiling`
  - `float:isFinite`, `float:isInfinite`, `float:isNaN`, `float:sum`, `float:max`, `float:min` and `float:avg`
  - `float:abs`, `float:round`, `float:floor`, `float:ceiling`, `float:sqrt`, `float:cbrt`, `float:pow`, `float:log`, `float:log10` and `float:exp`
  - `float:sin`, `float:cos`, `float:tan`, `float:asin`, `float:acos`, `float:atan`, `float:atan2`, `float:sinh`, `float:cosh` and `float:tanh`
  - `float:toHexString`, `float:toBitsInt`, `float:toFixedString` and `float:toExpString`
  - `int:toHexString`
  - `map:length`
  - `map:keys`
  - `map:hasKey`
  - `map:get`
  - `map:remove`
  - `error:message`
  - `error:cause`
  - `string:length`
  - `string:indexOf`, `string:includes`, `string:substring` and `string:equalsIgnoreCaseAscii`
  - `string:toLowerAscii`, `string:toUpperAscii` and `string:trim`
  - `string:toBytes`
  - `value:clone`
  - `value:cloneWithType`
  - `value:fromJsonWithType`
  - `value:toString` and `value:toBalString`
  - `xml:iterator`, `xml:length`, `xml:getName`, `xml:setName`, `xml:getAttributes`, `xml:getChildren`, `xml:setChildren`, `xml:getDescendants`, `xml:data`, `xml:getTarget`, `xml:getContent`, `xml:strip`, `xml:elements`, `xml:children`, `xml:elementChildren`, `xml:text`, `xml:get`, `xml:slice`, `xml:map`, `xml:forEach` and `xml:filter`

## Object/class definitions

- Only `client`, `service` and `isolated` `object-type-quals` / `class-type-quals` are supported
- Supports `object-field-descriptor`, `method-decl`, `remote-method-decl` and `resource-method-decl` members
- Supports `included-record-param`, `rest-param` and `defaultable-param` in methods
