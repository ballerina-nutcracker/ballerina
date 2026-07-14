# Supported language features (subset 3)

**Supported Ballerina code:** see [corpus/bal](../corpus/bal)—the [corpus/bal/subset3](../corpus/bal/subset3) directory contains the tests and examples that define what is supported in this subset.

## Module level declarations

- [Import declarations](https://ballerina.io/spec/lang/master/#import-decl)
- [Function definition](https://ballerina.io/spec/lang/master/#function-defn)
  - Currently only support [`block-function-body`](https://ballerina.io/spec/lang/master/#block-function-body)
  - Currently only support [`required-params`](https://ballerina.io/spec/lang/master/#required-params) in the signature
- [Constant declarations](https://ballerina.io/spec/lang/master/#module-const-decl)
  - Currently only support literals as constant expressions
  - Currently don't support type declarations in constant declarations

## Statements

- [Assignment](https://ballerina.io/spec/lang/master/#assignment-stmt)
  - See supported [`lvexpr`](#expressions)
- [Destructuring assignment statement](https://ballerina.io/spec/lang/master/#destructuring-assignment-stmt)
  - Only support [`wildcard-binding-pattern`](https://ballerina.io/spec/lang/master/#wildcard-binding-pattern)
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
  - Currently only support range and list

## Expressions

- [Literal](https://ballerina.io/spec/lang/master/#literal)
  - Currently support `nil-literal`, `boolean-literal`, `numeric-literal` (see [restrictions](#numeric-literal)), and `string-literal` only
- [lvexpr](https://ballerina.io/spec/lang/master/#section_7.14.1)
  - Currently only support [variable-reference-lvexpr](https://ballerina.io/spec/lang/master/#variable-reference-lvexpr)
- [`Call`](https://ballerina.io/spec/lang/master/#call-expr)
- [List constructor](https://ballerina.io/spec/lang/master/#list-constructor-expr)
  - Currently [spread-list-member](https://ballerina.io/spec/lang/master/#spread-list-member) not supported
- [Variable reference](https://ballerina.io/spec/lang/master/#variable-reference-expr)
  - Currently `xml-qualified-names` not supported
- [Unary logical expression](https://ballerina.io/spec/lang/master/#unary-logical-expr)
- [Nil lifted expression](https://ballerina.io/spec/lang/master/#nil-lifted-expr)
  - Currently nil lifting is not fully supported
  - Currently [`shift-expr`](https://ballerina.io/spec/lang/master/#shift-expr) not supported
- [Relational expression](https://ballerina.io/spec/lang/master/#relational-expr)
- [Equality expression](https://ballerina.io/spec/lang/master/#equality-expr)
- Nested expressions (`(expression)`)
- [Range expression](https://ballerina.io/spec/lang/master/#section_6.26)

## Operators

- Binary operators
  - Equality ops `==`, `!=`, `===`, `!==`
  - Multiplicative ops `*`, `%`, `/`
  - Bitwise ops `&`, `|`, `^`
  - Relational ops `<`, `<=`, `>`, `>=`,
  - Additive ops `+`, `-`
- Unary operators
  - logical `!`
  - numeric ops `+`, `-`

# Subset restrictions

## Import declarations

- Only fallowing libraries with given methods are supported
  - `ballerina/io`
    - `println`
  - `ballerina/lang.array`
    - `length`
    - `push`

## Method call

- Method call syntax can be used for calling the following langlib functions:
  - `array:length`
  - `array:push`

## numeric-literal

- Currently [`HexIntLiterals`](https://ballerina.io/spec/lang/master/#HexIntLiteral) not supported
