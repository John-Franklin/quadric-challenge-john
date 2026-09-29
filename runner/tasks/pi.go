package tasks

import (
	"context"
	"math/big"
)

const (
	defaultPiDigits = 1000
	piGuardDigits   = 10
	piLineWidth     = 100
)

// CalculatePi computes pi to in.Digits decimal places (default defaultPiDigits) with Machin's
// formula: pi = 16*arctan(1/5) - 4*arctan(1/239), in fixed-point big-integer arithmetic.
func CalculatePi(ctx context.Context, in Input, logf Logf) error {
	n := in.Digits
	if n <= 0 {
		n = defaultPiDigits
	}
	logf("Starting PI calculation (%d digits, Machin's formula)...", n)

	digits, err := computePi(ctx, n, logf)
	if err != nil {
		return err
	}

	logf("PI = %s...", digits[:2+min(50, n)])
	for i := 2; i < len(digits); i += piLineWidth {
		logf("  %s", digits[i:min(i+piLineWidth, len(digits))])
	}
	logf("PI calculation completed")
	return nil
}

// computePi returns pi as "3.<n decimals>", truncated (not rounded).
func computePi(ctx context.Context, n int, logf Logf) (string, error) {
	unity := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n+piGuardDigits)), nil)

	a5, err := arctanInv(ctx, 5, unity, logf)
	if err != nil {
		return "", err
	}
	a239, err := arctanInv(ctx, 239, unity, logf)
	if err != nil {
		return "", err
	}

	pi := new(big.Int).Mul(a5, big.NewInt(16))
	pi.Sub(pi, a239.Mul(a239, big.NewInt(4)))
	pi.Div(pi, new(big.Int).Exp(big.NewInt(10), big.NewInt(piGuardDigits), nil))

	s := pi.String()
	return s[:1] + "." + s[1:], nil
}

// arctanInv returns arctan(1/x) * unity using the Taylor series
// 1/x - 1/(3x^3) + 1/(5x^5) - ...
func arctanInv(ctx context.Context, x int64, unity *big.Int, logf Logf) (*big.Int, error) {
	bx := big.NewInt(x)
	x2 := big.NewInt(x * x)

	power := new(big.Int).Div(unity, bx) // unity / x^(2k+1)
	sum := new(big.Int).Set(power)
	term := new(big.Int)

	k := int64(1)
	for ; ; k++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		power.Div(power, x2)
		if power.Sign() == 0 {
			break
		}
		term.Div(power, big.NewInt(2*k+1))
		if k%2 == 1 {
			sum.Sub(sum, term)
		} else {
			sum.Add(sum, term)
		}
	}

	logf("arctan(1/%d) converged after %d terms", x, k)
	return sum, nil
}
