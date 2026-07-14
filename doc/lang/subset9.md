# Supported language features (subset 9)

**Supported Ballerina code:** see [corpus/bal](../../corpus/bal)—the [corpus/bal/subset9](../../corpus/bal/subset9) directory contains the tests and examples that define what is supported in this subset.

## Module-level declarations

- [Import declarations](https://ballerina.io/spec/lang/master/#import-decl)
- [Function definition](https://ballerina.io/spec/lang/master/#function-defn)
  - Supports [`required-params`](https://ballerina.io/spec/lang/master/#required-params), [`defaultable-params`](https://ballerina.io/spec/lang/master/#defaultable-params), [`included-record-param`](https://ballerina.io/spec/lang/master/#included-record-param) and [`rest-param`](https://ballerina.io/spec/lang/master/#rest-param) in the signature
  - Supports the [`isolated`](https://ballerina.io/spec/lang/master/#isolated-qual) function qualifier (see [isolated functions](https://ballerina.io/spec/lang/master/#isolated_functions))
  - Supports dependently-typed functions using `typedesc` parameters with the [`<>` inferred default](https://ballerina.io/spec/lang/master/#inferred-typedesc-default)
- [Constant declarations](https://ballerina.io/spec/lang/master/#module-const-decl)
- [Module variable declarations](https://ballerina.io/spec/lang/master/#module-var-decl)
- [Type definition](https://ballerina.io/spec/lang/master/#module-type-defn)
- [Enum declarations](https://ballerina.io/spec/lang/master/#module-enum-decl)
- [Class definition](https://ballerina.io/spec/lang/master/#section_8.6)
  - Supports `client` and `isolated` [`class-type-quals`](https://ballerina.io/spec/lang/master/#class-type-quals)
  - Supports `object-field` and `method-defn` members
  - Supports [`remote-method-defn`](https://ballerina.io/spec/lang/master/#remote-method-defn) for client classes

## Statements

- [Assignment](https://ballerina.io/spec/lang/master/#assignment-stmt)
  - See supported [`lvexpr`](#expressions)
- [Destructuring assignment statement](https://ballerina.io/spec/lang/master/#destructuring-assignment-stmt)
  - Only supports [`wildcard-binding-pattern`](https://ballerina.io/spec/lang/master/#wildcard-binding-pattern)
- [Compound Assignment](https://ballerina.io/spec/lang/master/#compound-assignment-stmt)
  - See supported [binary operators](#operators)
- [Break](https://ballerina.io/spec/lang/master/#break-stmt)
- [Continue](https://ballerina.io/spec/lang/master/#continue-stmt)
- [Call](https://ballerina.io/spec/lang/master/#call-stmt)
- [If/else](https://ballerina.io/spec/lang/master/#section_7.18)
- [While](https://ballerina.io/spec/lang/master/#while-stmt)
- [Local variable declarations](https://ballerina.io/spec/lang/master/#local-var-decl-stmt)
- [Return](https://ballerina.io/spec/lang/master/#return-stmt)
- [Panic](https://ballerina.io/spec/lang/master/#panic-stmt)
- [Foreach](https://ballerina.io/spec/lang/master/#section_7.21.1)
  - Currently only supports range, list, map subtypes and [iterable objects](https://ballerina.io/spec/lang/master/#section_5.8.2)
- [Match statement](https://ballerina.io/spec/lang/master/#match-stmt)
  - Currently only supports [const-pattern](https://ballerina.io/spec/lang/master/#const-pattern) and [wildcard-match-pattern](https://ballerina.io/spec/lang/master/#wildcard-match-pattern)

## Expressions

- [Literal](https://ballerina.io/spec/lang/master/#literal)
  - Currently support `nil-literal`, `boolean-literal`, `numeric-literal` and `string-literal` only
- [lvexpr](https://ballerina.io/spec/lang/master/#lvexpr)
- [`Call`](https://ballerina.io/spec/lang/master/#call-expr)
- [Method call](https://ballerina.io/spec/lang/master/#method-call-expr)
- [Client remote method call action](https://ballerina.io/spec/lang/master/#client-remote-method-call-action)
- [Error constructor](https://ballerina.io/spec/lang/master/#error-constructor-expr)
- [Check expression](https://ballerina.io/spec/lang/master/#checking-expr)
- [Type cast expression](https://ballerina.io/spec/lang/master/#type-cast-expr) 
- [New expression](https://ballerina.io/spec/lang/master/#section_6.8.2)
- [List constructor](https://ballerina.io/spec/lang/master/#list-constructor-expr)
  - Currently [spread-list-member](https://ballerina.io/spec/lang/master/#spread-list-member) not supported
- [Mapping constructor](https://ballerina.io/spec/lang/master/#mapping-constructor-expr)
  - Currently [spread-field](https://ballerina.io/spec/lang/master/#spread-field) not supported
- [XML template expression](https://ballerina.io/spec/lang/master/#xml-template-expr)
  - Currently interpolation not supported
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
- Nested expressions (`(expression)`)
- [Shift expression](https://ballerina.io/spec/lang/master/#section_6.25)
- [Type test expression](https://ballerina.io/spec/lang/master/#section_6.28)
- [Range expression](https://ballerina.io/spec/lang/master/#section_6.26)
- [Query expressions](https://ballerina.io/spec/lang/master/#query-expr)
  - Supports `from`, `where`, `let`, `join` (including outer join), `order by`, `limit`, `on conflict` and `select` clauses
  - Supports `list` and `map` as `query-constructor-type`

## Operators

- Binary operators
  - Equality ops `==`, `!=`, `===`, `!==`
  - Multiplicative ops `*`, `%`, `/`
  - Bitwise ops `&`, `|`, `^`
  - Relational ops `<`, `<=`, `>`, `>=`
  - Additive ops `+`, `-`
  - Shift ops `<<`, `>>`, `>>>`
- Unary operators
  - logical `!`
  - numeric ops `+`, `-`
  - bitwise complement `~`


# Subset restrictions

## Import declarations

- Only following lang libraries with given methods/types are supported
  - `ballerina/lang.array`
    - `length`
    - `map`
    - `push`
  - `ballerina/lang.int`
    - `Signed8`
    - `Signed16`
    - `Signed32`
    - `Unsigned8`
    - `Unsigned16`
    - `Unsigned32`
    - `toHexString`
  - `ballerina/lang.map`
    - `length`
    - `keys`
    - `remove`
  - `ballerina/lang.string`
    - `Char`
  - `ballerina/lang.error`
    - `message`
  - `ballerina/lang.value`
  - `ballerina/lang.xml`
    - `Element`
    - `Text`
    - `Comment`
    - `ProcessingInstruction`

## Function/Method call

- `named-args` and `defaultable-params` expect the target type to be atomic.
  - Note type narrowing to a narrowed type may not necessarily result in an atomic type.
- Method call syntax can be used for calling the following langlib functions:
  - `array:length`
  - `array:map`
  - `array:push`
  - `map:length`
  - `map:keys`
  - `map:remove`
  - `error:message`

## Object/class definitions

- Only `client` and `isolated` `object-type-quals` / `class-type-quals` are supported
- Supports `object-field-descriptor`, `method-decl` and `remote-method-decl` members
- Supports `rest-param` and `defaultable-param` in methods
