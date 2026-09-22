// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

# An enum containing the supported norm types.
# L1 - L1 norm which is the sum of the absolute values of the vector elements
# L2 - L2 norm which is the square root of the sum of the squared vector elements.
public enum NormType {
    L1,
    L2
}

# Calculates the L1 and L2 norm of a float vector.
#
# + v - float vector to operate on
# + norm - norm type to calculate out of [L1, L2]
# + return - the L1 or L2 norm of the given float vector
public isolated function vectorNorm(float[] v, NormType norm) returns float {
    if norm == L2 {
        float magnitudeSquared = 0.0;
        foreach int i in 0 ..< v.length() {
            magnitudeSquared += v[i].pow(2.0);
        }
        return magnitudeSquared.sqrt();
    } else {
        float absoluteMagnitude = 0.0;
        foreach int i in 0 ..< v.length() {
            absoluteMagnitude += v[i].abs();
        }
        return absoluteMagnitude;
    }
}

# Calculates the dot product of two float vectors.
#
# + v1 - float vector 1
# + v2 - float vector 2
# + return - the dot product of the given two vectors
public isolated function dotProduct(float[] v1, float[] v2) returns float {
    if v1.length() != v2.length() {
        panic error("The dot product can only be computed for two vectors of equal length.");
    } else {
        float dotProduct = 0.0;
        foreach int i in 0 ..< v1.length() {
            dotProduct += v1[i] * v2[i];
        }
        return dotProduct;
    }
}

# Calculates the cosine similarity between two float vectors.
#
# + v1 - float vector 1
# + v2 - float vector 2
# + return - the cosine similarity between the given two vectors
public isolated function cosineSimilarity(float[] v1, float[] v2) returns float {
    float vector1Norm = vectorNorm(v1, L2);
    float vector2Norm = vectorNorm(v2, L2);

    if vector1Norm == 0.0 || vector2Norm == 0.0 {
        panic error("Cosine similarity is undefined for zero vectors.");
    } else {
        return dotProduct(v1, v2) / (vector1Norm * vector2Norm);
    }
}

# Calculates the Euclidean distance between two float vectors.
#
# + v1 - float vector 1
# + v2 - float vector 2
# + return - the Euclidean distance between the given two vectors
public isolated function euclideanDistance(float[] v1, float[] v2) returns float {
    if v1.length() != v2.length() {
        panic error("The Euclidean distance can only be computed for two vectors of equal length.");
    } else {
        float differenceSquared = 0.0;
        foreach int i in 0 ..< v1.length() {
            differenceSquared += (v1[i] - v2[i]).pow(2.0);
        }
        return differenceSquared.sqrt();
    }
}

# Calculates the Manhattan distance between two float vectors.
#
# + v1 - float vector 1
# + v2 - float vector 2
# + return - the Manhattan distance between the given two vectors
public isolated function manhattanDistance(float[] v1, float[] v2) returns float {
    if v1.length() != v2.length() {
        panic error("The Manhattan distance can only be computed for two vectors of equal length.");
    } else {
        float absoluteDifference = 0.0;
        foreach int i in 0 ..< v1.length() {
            absoluteDifference += (v1[i] - v2[i]).abs();
        }
        return absoluteDifference;
    }
}
