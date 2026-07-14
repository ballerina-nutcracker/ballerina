# BIR Serializer Specification

## File Format Structure

The BIR binary file has the following structure:

```
+------------------+
| Magic (4 bytes)  | 0xBA 0x10 0xC0 0xDE
+------------------+
| Version (4 bytes)| int32 (currently 75)
+------------------+
| Constant Pool    | See Constant Pool Format
+------------------+
| Package Data     | See Package Structure
+------------------+
```

All multi-byte values are stored in **big-endian** byte order.

## Constant Pool Format

The constant pool is written before the package data and contains deduplicated entries for strings, packages, and shapes.

### Constant Pool Header

```
+------------------+
| Entry Count      | int64 (number of entries)
+------------------+
| Entries...       | Variable length
+------------------+
```

### Constant Pool Entry Types

Each entry starts with a 1-byte tag:

- `1` = STRING
- `2` = PACKAGE
- `3` = SHAPE (not yet implemented)

### String Entry (Tag 1)

```
+------------------+
| Tag              | uint8 (1)
+------------------+
| Length           | int64 (string length in bytes)
+------------------+
| UTF-8 Bytes      | Variable length
+------------------+
```

### Package Entry (Tag 2)

```
+------------------+
| Tag              | uint8 (2)
+------------------+
| Org Name Index   | int32 (CP index to string)
+------------------+
| Pkg Name Index   | int32 (CP index to string)
+------------------+
| Module Name Index| int32 (CP index to string)
+------------------+
| Version Index    | int32 (CP index to string)
+------------------+
```

### Shape Entry (Tag 3)

Not yet implemented.

## Package Structure

After the constant pool, the package data follows:

```
+------------------+
| Package CP Index | int32 (index to package entry in CP)
+------------------+
| Global Variables | See Global Variables
+------------------+
| Functions        | See Functions
+------------------+
```

### Global Variables

```
+------------------+
| Count            | int64 (number of global variables)
+------------------+
| For each variable: |
|   Position       | See Position
|   Kind           | uint8
|   Name CP        | int32
|   Flags          | int64
|   Origin         | uint8
|   Type CP        | int32 (currently -1, not implemented)
+------------------+
```

### Functions

```
+------------------+
| Count            | int64 (number of functions)
+------------------+
| For each function: |
|   Position       | See Position
|   Name CP        | int32
|   Original Name  | int32
|   Flags           | int64
|   Origin          | uint8
|   Required Params | See Required Parameters
|   Function Body   | See Function Body
+------------------+
```

#### Required Parameters

```
+------------------+
| Count            | int64
+------------------+
| For each param:  |
|   Name CP        | int32
|   Flags          | int64
+------------------+
```

#### Function Body

```
+------------------+
| Body Length      | int64 (total length in bytes)
+------------------+
| Args Count       | int64
+------------------+
| Has Return Var    | bool
| [If has return:]  |
|   Kind           | uint8
|   Type CP        | int32
|   Name CP        | int32
+------------------+
| Local Vars Count | int64
| Local Vars...    | See Local Variable
+------------------+
| Basic Blocks     | See Basic Blocks
+------------------+
```

#### Local Variable

```
+------------------+
| Kind             | uint8
+------------------+
| Type CP          | int32
+------------------+
| Name CP          | int32
+------------------+
| [If ARG kind:]    |
|   Meta Var Name  | int32
+------------------+
| [If LOCAL kind:]  |
|   Meta Var Name  | int32
|   End BB ID CP   | int32
|   Start BB ID CP | int32
|   Ins Offset     | int64
+------------------+
```

#### Basic Blocks

```
+------------------+
| Count            | int64
+------------------+
| For each block:  |
|   ID CP          | int32
|   Ins Count      | int64
|   Instructions   | See Instructions
|   Terminator     | See Terminator
+------------------+
```

## Constant Value Format

Constant values are written with a type tag followed by the value:

```
+------------------+
| Type Tag         | int8 (TypeTags enum)
+------------------+
| Value            | See Value Encoding
+------------------+
```

### Value Encoding

The value encoding depends on the type tag:

#### Primitive Types (Written Inline)

- **Integer types** (`INT`, `SIGNED32_INT`, `SIGNED16_INT`, `SIGNED8_INT`, `UNSIGNED32_INT`, `UNSIGNED16_INT`, `UNSIGNED8_INT`): `int64`
- **BYTE**: `byte` (1 byte)
- **FLOAT**: `float64` (8 bytes)
- **BOOLEAN**: `bool` (1 byte)

#### Reference Types (Constant Pool Indices)

- **STRING**, **CHAR_STRING**, **DECIMAL**: `int32` (CP index to string entry)
- **NIL**: `int32` (always -1)

## Instructions

Each instruction is prefixed with its instruction kind (1 byte):

```
+------------------+
| Instruction Kind | uint8
+------------------+
| Instruction Data | Variable (depends on kind)
+------------------+
```

### Supported Instruction Kinds

- `MOVE`: Move operation
- `BINARY_OP`: Binary operations (ADD, SUB, MUL, etc.)
- `UNARY_OP`: Unary operations (NOT, NEGATE, etc.)
- `CONST_LOAD`: Load constant value
- `FIELD_ACCESS`: Field/map/array access
- `NEW_ARRAY`: Create new array

### Constant Load Instruction

```
+------------------+
| Instruction Kind | uint8 (CONST_LOAD)
+------------------+
| Type CP          | int32
+------------------+
| LHS Operand      | See Operand
+------------------+
| Is Wrapped       | bool
+------------------+
| Value Tag        | int8
+------------------+
| Value            | See Constant Value Format
+------------------+
```

### Operand Format

```
+------------------+
| Ignore Variable  | bool
+------------------+
| [If ignore:]      |
|   Type CP        | int32
+------------------+
| [If not ignore:]  |
|   Kind           | uint8
|   Scope          | uint8
|   Name CP        | int32
+------------------+
```

## Terminator Format

Terminators end basic blocks:

```
+------------------+
| Terminator Kind  | uint8 (0 = none)
+------------------+
| [If kind != 0:]   |
|   Terminator Data| Variable (depends on kind)
+------------------+
```

### Supported Terminator Kinds

- `GOTO`: Unconditional branch
- `BRANCH`: Conditional branch
- `CALL`: Function call
- `RETURN`: Return from function

## Position Format

```
+------------------+
| Source File CP   | int32 (CP index to string)
+------------------+
| Start Line       | int32
+------------------+
| Start Column      | int32
+------------------+
| End Line          | int32
+------------------+
| End Column        | int32
+------------------+
```