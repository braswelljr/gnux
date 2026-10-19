package factor_test

import (
	"bytes"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/braswelljr/gnux/core"
	"github.com/braswelljr/gnux/core/factor"
)

func run(opts factor.Options, stdin string, args ...string) (stdout, stderr string, err error) {
	var out, errb bytes.Buffer
	err = factor.New(opts, core.Streams{In: strings.NewReader(stdin), Out: &out, Err: &errb}).Run(args)
	return out.String(), errb.String(), err
}

func TestOperands(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"0", "0:\n"},
		{"1", "1:\n"},
		{"2", "2: 2\n"},
		{"12", "12: 2 2 3\n"},
		{"97", "97: 97\n"},
		{"1000000007", "1000000007: 1000000007\n"},
		{"+0010", "10: 2 5\n"},
		{"  42", "42: 2 3 7\n"},
		{"4611686014132420609", "4611686014132420609: 2147483647 2147483647\n"},
		{"18446744073709551615", "18446744073709551615: 3 5 17 257 641 65537 6700417\n"},
		{"18446744073709551557", "18446744073709551557: 18446744073709551557\n"},
		{"18446744073709551616", "18446744073709551616:" + strings.Repeat(" 2", 64) + "\n"},
		{"1234567890123456789012345678901234567890",
			"1234567890123456789012345678901234567890: 2 3 3 5 73 101 137 3541 3607 3803 27961 1676321 5964848081\n"},
		{"170141183460469231731687303715884105727", "170141183460469231731687303715884105727: 170141183460469231731687303715884105727\n"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			out, errOut, err := run(factor.Options{}, "", tt.in)
			if err != nil || out != tt.want || errOut != "" {
				t.Fatalf("out = %q, stderr = %q, err = %v; want %q", out, errOut, err, tt.want)
			}
		})
	}
}

func TestExponents(t *testing.T) {
	out, _, err := run(factor.Options{Exponents: true}, "", "1024", "360", "7", "1")
	if want := "1024: 2^10\n360: 2^3 3^2 5\n7: 7\n1:\n"; err != nil || out != want {
		t.Fatalf("out = %q, err = %v; want %q", out, err, want)
	}
}

func TestStdin(t *testing.T) {
	out, errOut, err := run(factor.Options{}, "12 abc\n9\t+0010\n\n")
	if want := "12: 2 2 3\n9: 3 3\n10: 2 5\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	if want := "factorx: 'abc' is not a valid positive integer\n"; errOut != want {
		t.Fatalf("stderr = %q; want %q", errOut, want)
	}
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestInvalidOperandsDoNotStopTheRest(t *testing.T) {
	out, errOut, err := run(factor.Options{}, "", "", "4", "-3", "1.5", "0x10", "5 ", "6")
	if want := "4: 2 2\n6: 2 3\n"; out != want {
		t.Fatalf("out = %q; want %q", out, want)
	}
	var want strings.Builder
	for _, bad := range []string{"", "-3", "1.5", "0x10", "5 "} {
		want.WriteString("factorx: '" + bad + "' is not a valid positive integer\n")
	}
	if errOut != want.String() {
		t.Fatalf("stderr = %q; want %q", errOut, want.String())
	}
	if !errors.Is(err, core.ErrFailed) {
		t.Fatalf("err = %v", err)
	}
}

// TestOfMultipliesBack checks factorizations against the number itself over
// a range that crosses the trial-division limit and needs rho.
func TestOfMultipliesBack(t *testing.T) {
	check := func(n *big.Int) {
		t.Helper()
		product := big.NewInt(1)
		prev := big.NewInt(0)
		for _, p := range factor.Of(n) {
			if !p.ProbablyPrime(20) {
				t.Fatalf("%v: factor %v is not prime", n, p)
			}
			if p.Cmp(prev) < 0 {
				t.Fatalf("%v: factors out of order", n)
			}
			prev = p
			product.Mul(product, p)
		}
		if product.Cmp(n) != 0 {
			t.Fatalf("%v: factors multiply to %v", n, product)
		}
	}
	for i := int64(2); i < 5000; i++ {
		check(big.NewInt(i))
	}
	// Semiprimes and powers of primes just above the trial limit.
	for _, s := range []string{"1018081", "999985999949", "1000036000099", "9223372036854775783",
		"9223372036854775807", "18446744030759878681", "36893488147419103232", "1000000016000000063"} {
		n, _ := new(big.Int).SetString(s, 10)
		check(n)
	}
}
