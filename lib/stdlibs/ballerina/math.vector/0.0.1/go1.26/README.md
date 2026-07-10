# Ballerina Math Vector Library

## Overview

This module provides vector math operations for float vectors, primarily for use in AI and machine learning workflows. It supports norm calculation, dot product, cosine similarity, and distance metrics (Euclidean and Manhattan).

## Key Functionalities

- Calculate the L1 or L2 norm of a float vector.
- Compute the dot product of two float vectors.
- Compute the cosine similarity between two float vectors.
- Compute the Euclidean distance between two float vectors.
- Compute the Manhattan distance between two float vectors.

## Examples

```ballerina
import ballerina/io;
import ballerina/math.vector;

public function main() {
    float[] v1 = [1.0, 2.0, 3.0];
    float[] v2 = [4.0, 5.0, 6.0];

    float l2Norm = vector:vectorNorm(v1, vector:L2);
    io:println("L2 norm: ", l2Norm);

    float dot = vector:dotProduct(v1, v2);
    io:println("Dot product: ", dot);

    float similarity = vector:cosineSimilarity(v1, v2);
    io:println("Cosine similarity: ", similarity);

    float euclidean = vector:euclideanDistance(v1, v2);
    io:println("Euclidean distance: ", euclidean);

    float manhattan = vector:manhattanDistance(v1, v2);
    io:println("Manhattan distance: ", manhattan);
}
```

## Ballerina Nutcracker Support Status

This library is currently being migrated to Go to support Ballerina Nutcracker. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| L1 and L2 norm calculation | Supported | Fully implemented in pure Ballerina; no Go externs required. |
| Dot product of two vectors | Supported | Panics if vectors differ in length, matching jBallerina behaviour. |
| Cosine similarity between two vectors | Supported | Panics on zero vectors, matching jBallerina behaviour. |
| Euclidean distance between two vectors | Supported | Panics if vectors differ in length, matching jBallerina behaviour. |
| Manhattan distance between two vectors | Supported | Panics if vectors differ in length, matching jBallerina behaviour. |

### Notable Behavioural Changes

There are **no** notable behavioural changes in the Go-native version compared to the original jBallerina implementation for the currently supported features.
