# Supported language features (subset 1)

Subset 1 is the **first milestone**: a minimal core of the language (no destructuring, type cast, list constructor, or nil lifting).

**Supported Ballerina code:** see [corpus/bal](../corpus/bal)—the [corpus/bal/subset1](../corpus/bal/subset1) directory contains the tests and examples that define what is supported in this subset.

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
- [If-else statement](https://ballerina.io/spec/lang/master/#if-else-stmt)

## Expressions

- [Literal](https://ballerina.io/spec/lang/master/#literal)
  - Currently support `nil-literal`, `boolean-literal`, `numeric-literal` (see [restrictions](#numeric-literal)), and `string-literal` only
- [lvexpr](https://ballerina.io/spec/lang/master/#section_7.14.1)
  - Currently only support [variable-reference-lvexpr](https://ballerina.io/spec/lang/master/#variable-reference-lvexpr)
- [`Call`](https://ballerina.io/spec/lang/master/#call-expr)
  - Currently only support `function-call-expr`
- [Variable reference](https://ballerina.io/spec/lang/master/#variable-reference-expr)
  - Currently `xml-qualified-names` not supported
- [Unary logical expression](https://ballerina.io/spec/lang/master/#unary-logical-expr)
- [Relational expression](https://ballerina.io/spec/lang/master/#relational-expr)
- [Equality expression](https://ballerina.io/spec/lang/master/#equality-expr)
- Nested expressions (`(expression)`)

## Operators

- Binary operators
  - Equality ops `==`, `!=`, `===`, `!==`
  - Multiplicative ops `*`, `%`, `/`
  - Bitwise ops `&`, `|`, `^`
  - Relational ops `<`, `<=`, `>`, `>=`
  - Additive ops `+`, `-`
- Unary operators
  - Logical `!`
  - Numeric `+`, `-`

# Subset restrictions

## Import declarations

- Only `ballerina/io` is supported
  - Only `println` function is supported

## numeric-literal

- Currently [`HexIntLiterals`](https://ballerina.io/spec/lang/master/#HexIntLiteral) not supported

## Not in subset 1

- [Destructuring assignment](https://ballerina.io/spec/lang/master/#destructuring-assignment-stmt)
- [Type cast expression](https://ballerina.io/spec/lang/master/#section_6.20)
- [List constructor](https://ballerina.io/spec/lang/master/#list-constructor-expr)
- [Nil lifted expression](https://ballerina.io/spec/lang/master/#nil-lifted-expr) (nil lifting)
- [Foreach](https://ballerina.io/spec/lang/master/#section_7.21.1)
- [Range expression](https://ballerina.io/spec/lang/master/#section_6.26)
