# Supported language features (subset 7)

**Supported Ballerina code:** see [corpus/bal](../corpus/bal)—the [corpus/bal/subset7](../corpus/bal/subset7) directory contains the tests and examples that define what is supported in this subset.

## Module-level declarations

- [Import declarations](https://ballerina.io/spec/lang/master/#import-decl)
- [Function definition](https://ballerina.io/spec/lang/master/#function-defn)
  - Currently only supports [`required-params`](https://ballerina.io/spec/lang/master/#required-params) and [`defaultable-params`](https://ballerina.io/spec/lang/master/#defaultable-params) in the signature
- [Constant declarations](https://ballerina.io/spec/lang/master/#module-const-decl)
- [Module variable declarations](https://ballerina.io/spec/lang/master/#module-var-decl)
- [Class definition](https://ballerina.io/spec/lang/master/#section_8.6)
  - `class-type-quals` not supported
  - Only `object-field` and `method-defn` members supported

## Statements

- [Assignment](https://ballerina.io/spec/lang/master/#assignment-stmt)
  - See supported [`lvexpr`](#expressions)
- [Destructuring assignment statement](https://ballerina.io/spec/lang/master/#destructuring-assignment-stmt)
  - Only supports [`wildcard-binding-pattern`](https://ballerina.io/spec/lang/master/#wildcard-binding-pattern)
- [Compound Assignment](https://ballerina.io/spec/lang/master/#compound-assignment-stmt)
  - See supported [binary operators](#operators)
  - Currently don't fully support [nil lifting](https://ballerina.io/spec/lang/master/#nil_lifting)
- [Break](https://ballerina.io/spec/lang/master/#break-stmt)
- [Continue](https://ballerina.io/spec/lang/master/#continue-stmt)
- [Call](https://ballerina.io/spec/lang/master/#call-stmt)
- [While](https://ballerina.io/spec/lang/master/#while-stmt)
- [Local variable declarations](https://ballerina.io/spec/lang/master/#local-var-decl-stmt)
  - Currently don't support `final`
- [Return](https://ballerina.io/spec/lang/master/#return-stmt)
- [Foreach](https://ballerina.io/spec/lang/master/#section_7.21.1)
  - Currently only supports range, list and map subtypes

## Expressions

- [Literal](https://ballerina.io/spec/lang/master/#literal)
  - Currently support `nil-literal`, `boolean-literal`, `numeric-literal` and `string-literal` only
- [lvexpr](https://ballerina.io/spec/lang/master/#section_7.14.1)
  - Currently only supports [variable-reference-lvexpr](https://ballerina.io/spec/lang/master/#variable-reference-lvexpr)
- [`Call`](https://ballerina.io/spec/lang/master/#call-expr)
- [List constructor](https://ballerina.io/spec/lang/master/#list-constructor-expr)
  - Currently [spread-list-member](https://ballerina.io/spec/lang/master/#spread-list-member) not supported
- [Variable reference](https://ballerina.io/spec/lang/master/#variable-reference-expr)
  - Currently `xml-qualified-names` not supported
- [Unary logical expression](https://ballerina.io/spec/lang/master/#unary-logical-expr)
- [Nil lifted expression](https://ballerina.io/spec/lang/master/#nil-lifted-expr)
  - Currently nil lifting is not fully supported
- [Relational expression](https://ballerina.io/spec/lang/master/#relational-expr)
- [Equality expression](https://ballerina.io/spec/lang/master/#equality-expr)
- Nested expressions (`(expression)`)
- [Shift expression](https://ballerina.io/spec/lang/master/#section_6.25)
- [Type test expression](https://ballerina.io/spec/lang/master/#section_6.28)
- [Field access expression](https://ballerina.io/spec/lang/master/#section_6.10)
- [Range expression](https://ballerina.io/spec/lang/master/#section_6.26)
- [New expression](https://ballerina.io/spec/lang/master/#section_6.8.2)
- [Query expressions](https://ballerina.io/spec/lang/master/#query-expr)
  - Currently only supports the `from`, `where`, `let`, and `select` clauses.
  - Currently only `map` is supported for `query-constructor-type`.

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

# Subset restrictions

## Import declarations

- Only following libraries with given methods are supported
  - `ballerina/io`
    - `println`
  - `ballerina/lang.array`
    - `length`
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

## Method call

- Method call syntax can be used for calling the following langlib functions:
  - `array:length`
  - `array:push`
  - `map:length`
  - `map:keys`
  - `map:remove`

## Object type definitions

- `object-type-quals` not supported.
- Only `object-field-descriptor` and `method-decl` member descriptors supported
