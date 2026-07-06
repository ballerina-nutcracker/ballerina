// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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

// Package decimal provides Ballerina's IEEE 754 decimal128 numeric type [https://ballerina.io/spec/lang/master/#section_5.2.4.2].
package decimal

import (
	"github.com/cockroachdb/apd/v3"
)

const (
	precision   = 34
	maxExponent = 6144
	minExponent = -6143
)

// context is the IEEE 754 decimal128 arithmetic context used for all Ballerina
// decimal operations: 34 significant digits, exponent range [-6143, 6144],
// round-half-to-even. Subnormal/Underflow conditions are handled explicitly
// (flushed to zero rather than trapped) via validate.
var context = func() *apd.Context {
	c := apd.BaseContext.WithPrecision(precision)
	c.MaxExponent = maxExponent
	c.MinExponent = minExponent
	c.Rounding = apd.RoundHalfEven
	c.Traps = apd.SystemOverflow | apd.SystemUnderflow | apd.Overflow |
		apd.DivisionUndefined | apd.DivisionByZero | apd.DivisionImpossible |
		apd.InvalidOperation
	return c
}()

// ErrorKind identifies the category of a decimal arithmetic failure.
type ErrorKind int

const (
	ErrInvalid ErrorKind = iota + 1
	ErrOverflow
	ErrUnderflow
	ErrDivisionByZero
	ErrSyntax
)

// Error is the typed error returned by every operation in this package that
// can fail because of decimal128 limits. The runtime maps each Kind onto the
// matching Ballerina runtime error message.
type Error struct {
	Kind ErrorKind
}

func (e *Error) Error() string {
	switch e.Kind {
	case ErrInvalid:
		return "not a valid decimal"
	case ErrOverflow:
		return "arithmetic overflow"
	case ErrUnderflow:
		return "arithmetic underflow"
	case ErrDivisionByZero:
		return "divide by zero"
	case ErrSyntax:
		return "invalid decimal syntax"
	}
	return "decimal error"
}

// Decimal is a Ballerina decimal value. The zero value represents +0.
type Decimal struct {
	v apd.Decimal
}

// FromInt64 returns the decimal representation of n.
func FromInt64(n int64) *Decimal {
	d := &Decimal{}
	d.v.SetInt64(n)
	return d
}

// FromString parses s as a decimal literal without applying decimal128
// rounding. It is the appropriate constructor when the source is already known
// to be a canonical decimal128 representation (BIR constants, type-pool
// entries).
func FromString(s string) (*Decimal, *Error) {
	r, _, err := apd.NewFromString(s)
	if err != nil {
		return nil, &Error{Kind: ErrSyntax}
	}
	return &Decimal{v: *r}, nil
}

// FromLiteral parses s as a Ballerina decimal literal, applying decimal128
// rounding. Out-of-range values surface as a typed *Error.
func FromLiteral(s string) (*Decimal, *Error) {
	r, cond, err := context.NewFromString(s)
	if err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	d := &Decimal{v: *r}
	if e := validate(d, cond); e != nil {
		return nil, e
	}
	return d, nil
}

// FromFloat64 converts an IEEE 754 float64 into a Ballerina decimal. NaN,
// infinities and subnormals are not representable, so they surface as a typed
// *Error.
func FromFloat64(f float64) (*Decimal, *Error) {
	d := &Decimal{}
	if _, err := d.v.SetFloat64(f); err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	if e := validate(d, 0); e != nil {
		return nil, e
	}
	return d, nil
}

// validate enforces Ballerina's decimal value invariants on the result of an
// apd operation. Ballerina forbids NaN, infinities, and subnormals; the
// matching apd Form/Condition becomes a typed *Error. A finite, normal value
// is canonicalised to Ballerina's representation: -0 is collapsed to +0.
func validate(d *Decimal, cond apd.Condition) *Error {
	switch d.v.Form {
	case apd.NaN, apd.NaNSignaling:
		return &Error{Kind: ErrInvalid}
	case apd.Infinite:
		return &Error{Kind: ErrOverflow}
	}
	if cond.InvalidOperation() {
		return &Error{Kind: ErrInvalid}
	}
	if cond.Overflow() {
		return &Error{Kind: ErrOverflow}
	}
	if cond.DivisionByZero() {
		return &Error{Kind: ErrDivisionByZero}
	}
	if cond.Subnormal() || cond.Underflow() {
		return &Error{Kind: ErrUnderflow}
	}
	if d.v.IsZero() && d.v.Negative {
		d.v.Negative = false
	}
	return nil
}

func arith(a, b *Decimal, op func(*apd.Decimal, *apd.Decimal, *apd.Decimal) (apd.Condition, error)) (*Decimal, *Error) {
	out := &Decimal{}
	cond, err := op(&out.v, &a.v, &b.v)
	if e := validate(out, cond); e != nil {
		return nil, e
	}
	if err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	normalizeUpperExponent(out)
	return out, nil
}

// Add returns a + b.
func (a *Decimal) Add(b *Decimal) (*Decimal, *Error) {
	return arith(a, b, context.Add)
}

// Sub returns a - b.
func (a *Decimal) Sub(b *Decimal) (*Decimal, *Error) {
	return arith(a, b, context.Sub)
}

// Mul returns a * b.
func (a *Decimal) Mul(b *Decimal) (*Decimal, *Error) {
	return arith(a, b, context.Mul)
}

// Quo returns a / b. Division by zero produces an *Error of kind
// ErrDivisionByZero. apd.Context.Quo always pads the result to the context's
// precision; for exact results IEEE 754 prescribes the natural (ideal)
// exponent, so we strip trailing zeros via Reduce when the operation is exact.
func (a *Decimal) Quo(b *Decimal) (*Decimal, *Error) {
	if b.v.IsZero() {
		return nil, &Error{Kind: ErrDivisionByZero}
	}
	out := &Decimal{}
	cond, err := context.Quo(&out.v, &a.v, &b.v)
	if err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	if e := validate(out, cond); e != nil {
		return nil, e
	}
	// IEEE 754 prescribes the natural (ideal) exponent for exact division
	// results; Reduce strips the trailing zeros that Context.Quo always pads
	// to the context precision. Run Reduce first so we can then enforce the
	// upper-exponent canonical form on the reduced representation.
	if !cond.Inexact() {
		out.v.Reduce(&out.v)
	}
	normalizeUpperExponent(out)
	return out, nil
}

// Rem returns the remainder of a / b. Division by zero produces an *Error of
// kind ErrDivisionByZero.
func (a *Decimal) Rem(b *Decimal) (*Decimal, *Error) {
	if b.v.IsZero() {
		return nil, &Error{Kind: ErrDivisionByZero}
	}
	return arith(a, b, context.Rem)
}

// Neg returns -a. Negation cannot overflow.
func (a *Decimal) Neg() *Decimal {
	out := &Decimal{}
	out.v.Neg(&a.v)
	return out
}

// Abs returns the absolute value of a.
func (a *Decimal) Abs() *Decimal {
	out := &Decimal{}
	out.v.Abs(&a.v)
	return out
}

// Round returns a rounded to fractionDigits digits after the decimal point,
// using the decimal128 round-to-nearest-ties-to-even mode.
func (a *Decimal) Round(fractionDigits int64) (*Decimal, *Error) {
	const (
		minFractionDigits = -2147483647
		maxFractionDigits = 2147483648
	)
	if fractionDigits < minFractionDigits || fractionDigits > maxFractionDigits {
		return nil, &Error{Kind: ErrInvalid}
	}
	return a.quantizeExp(int32(-fractionDigits), context.Rounding)
}

// Quantize returns a rounded to the exponent of b.
func (a *Decimal) Quantize(b *Decimal) (*Decimal, *Error) {
	return a.quantizeExp(b.v.Exponent, context.Rounding)
}

// Floor returns the largest integral decimal not greater than a.
func (a *Decimal) Floor() (*Decimal, *Error) {
	return a.roundToIntegral(apd.RoundFloor)
}

// Ceiling returns the smallest integral decimal not less than a.
func (a *Decimal) Ceiling() (*Decimal, *Error) {
	return a.roundToIntegral(apd.RoundCeiling)
}

func (a *Decimal) quantizeExp(exp int32, rounding apd.Rounder) (*Decimal, *Error) {
	ctx := *context
	ctx.Rounding = rounding
	out := &Decimal{}
	cond, err := ctx.Quantize(&out.v, &a.v, exp)
	if e := validate(out, cond); e != nil {
		return nil, e
	}
	if err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	normalizeUpperExponent(out)
	return out, nil
}

func (a *Decimal) roundToIntegral(rounding apd.Rounder) (*Decimal, *Error) {
	ctx := *context
	ctx.Rounding = rounding
	out := &Decimal{}
	cond, err := ctx.RoundToIntegralValue(&out.v, &a.v)
	if e := validate(out, cond); e != nil {
		return nil, e
	}
	if err != nil {
		return nil, &Error{Kind: ErrInvalid}
	}
	normalizeUpperExponent(out)
	return out, nil
}

// Cmp returns -1, 0, or +1 for a<b, a==b, a>b.
func (a *Decimal) Cmp(b *Decimal) int {
	return a.v.Cmp(&b.v)
}

// ExactEqual implements Ballerina's `===` for decimals: two decimals are
// exactly equal only when their numeric value AND their decimal128
// representation (coefficient, exponent) match. This distinguishes e.g.
// 1.0 and 1.00, which are `==` but not `===`. Sign is ignored for zero so
// that +0 and -0 are exactly equal.
func (a *Decimal) ExactEqual(b *Decimal) bool {
	if a.v.Form != b.v.Form || a.v.Exponent != b.v.Exponent || a.v.Coeff.Cmp(&b.v.Coeff) != 0 {
		return false
	}
	return a.v.IsZero() || a.v.Negative == b.v.Negative
}

// String renders the value using its canonical IEEE 754 decimal128 string
// form, preserving the original coefficient/exponent representation (e.g.
// "0.0", "1.00"). Use FormatBallerina for the runtime println rendering that
// collapses any zero to "0".
func (a *Decimal) String() string {
	return a.v.String()
}

// FormatBallerina renders the value as Ballerina's runtime println output:
// the canonical decimal128 form, except zero is always emitted as "0".
func (a *Decimal) FormatBallerina() string {
	if a.v.IsZero() {
		return "0"
	}
	return a.v.String()
}

// Int64 rounds the value to an integer using the context's rounding mode and
// returns the int64 representation. The boolean is false when the rounded
// value does not fit in an int64.
func (a *Decimal) Int64() (int64, bool, *Error) {
	rounded := &apd.Decimal{}
	if _, err := context.RoundToIntegralValue(rounded, &a.v); err != nil {
		return 0, false, &Error{Kind: ErrInvalid}
	}
	n, err := rounded.Int64()
	if err != nil {
		return 0, false, nil
	}
	return n, true, nil
}

// Float64 returns the closest IEEE 754 float64 representation of the value.
func (a *Decimal) Float64() float64 {
	f, _ := a.v.Float64()
	return f
}

// normalizeUpperExponent enforces the IEEE 754 decimal128 canonical form when
// the result exponent exceeds the highest representable raw exponent (Emax -
// precision + 1). The coefficient is padded with trailing zeros and the
// exponent is reduced accordingly, preserving the numeric value while using
// the spec-mandated representation (e.g. 1E+6144 becomes 1.000...E+6144).
func normalizeUpperExponent(d *Decimal) {
	if d.v.Form != apd.Finite || d.v.IsZero() {
		return
	}
	etop := context.MaxExponent - int32(context.Precision) + 1
	if d.v.Exponent <= etop {
		return
	}
	shift := int64(d.v.Exponent - etop)
	var mul apd.BigInt
	mul.Exp(apd.NewBigInt(10), apd.NewBigInt(shift), nil)
	d.v.Coeff.Mul(&d.v.Coeff, &mul)
	d.v.Exponent = etop
}
