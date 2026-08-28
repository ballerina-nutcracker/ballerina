# xml-filter-expression

## Grammer

```
xml-filter-expr := expression .< xml-name-pattern >
xml-name-pattern := xml-atomic-name-pattern [| xml-atomic-name-pattern]*

xml-atomic-name-pattern :=
  *
  | identifier
  | xml-namespace-prefix NoSpaceColon identifier
  | xml-namespace-prefix NoSpaceColon *
```


## AST
- We will use a fat struct with a tag to represent all name patterns
  - We need to support lowring identifier only case to fully qualified indentifier when there is a default name space in scope
    - Keep symbols and ast nodes seperate (when lowering you can set the kind and symbols correctly but not the Prefix Identifier)


## Runtime/BIR
- We'll lower these directly to BIR (no desugar) instructions
