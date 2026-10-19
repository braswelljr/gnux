package factor

import (
	"math/big"
	"math/bits"
	"slices"
)

var one = big.NewInt(1)

// smallPrimes are the primes below 1000, divided out by trial before any
// probabilistic work starts.
var smallPrimes = func() []uint64 {
	var ps []uint64
	sieve := make([]bool, 1000)
	for i := 2; i < len(sieve); i++ {
		if sieve[i] {
			continue
		}
		ps = append(ps, uint64(i))
		for j := i * i; j < len(sieve); j += i {
			sieve[j] = true
		}
	}
	return ps
}()

// factor64 returns the prime factors of n > 1 in ascending order.
func factor64(n uint64) []uint64 {
	var fs []uint64
	for _, p := range smallPrimes {
		if p*p > n {
			break
		}
		for n%p == 0 {
			fs = append(fs, p)
			n /= p
		}
	}
	if n > 1 {
		fs = splitInto(n, fs)
	}
	slices.Sort(fs)
	return fs
}

// splitInto appends the prime factors of n > 1, which has no factor below
// 1000, to fs.
func splitInto(n uint64, fs []uint64) []uint64 {
	last := smallPrimes[len(smallPrimes)-1]
	if n < last*last || isPrime64(n) {
		return append(fs, n)
	}
	d := rho64(n)
	fs = splitInto(d, fs)
	return splitInto(n/d, fs)
}

// mulmod returns a*b mod m without overflow.
func mulmod(a, b, m uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	return bits.Rem64(hi, lo, m)
}

func powmod(b, e, m uint64) uint64 {
	r := uint64(1)
	b %= m
	for ; e > 0; e >>= 1 {
		if e&1 == 1 {
			r = mulmod(r, b, m)
		}
		b = mulmod(b, b, m)
	}
	return r
}

// isPrime64 is Miller-Rabin with the first twelve primes as bases, which is
// deterministic for every n below 2^64.
func isPrime64(n uint64) bool {
	if n < 2 {
		return false
	}
	for _, p := range smallPrimes[:12] {
		if n%p == 0 {
			return n == p
		}
	}
	d, s := n-1, 0
	for d&1 == 0 {
		d >>= 1
		s++
	}
	for _, a := range smallPrimes[:12] {
		x := powmod(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}
		composite := true
		for range s - 1 {
			x = mulmod(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// rho64 finds a nontrivial factor of the odd composite n with Brent's
// variant of Pollard's rho, moving to another polynomial when one fails.
func rho64(n uint64) uint64 {
	for c := uint64(1); ; c++ {
		f := func(x uint64) uint64 { return (mulmod(x, x, n) + c) % n }
		y, r, q := uint64(2), uint64(1), uint64(1)
		var x, ys, g uint64
		const m = 128
		for g = 1; g == 1; r <<= 1 {
			x = y
			for range r {
				y = f(y)
			}
			for k := uint64(0); k < r && g == 1; k += m {
				ys = y
				for range min(m, r-k) {
					y = f(y)
					diff := x - y
					if x < y {
						diff = y - x
					}
					q = mulmod(q, diff, n)
				}
				g = gcd(q, n)
			}
		}
		if g == n {
			// The batch overshot: step back one value at a time.
			for g = 1; g == 1; {
				ys = f(ys)
				diff := x - ys
				if x < ys {
					diff = ys - x
				}
				g = gcd(diff, n)
			}
		}
		if g != n {
			return g
		}
	}
}

// factorBig appends the prime factors of n > 1 to fs, in no particular
// order. A cofactor that fits in 64 bits goes to the fast path.
func factorBig(n *big.Int, fs []*big.Int) []*big.Int {
	var r big.Int
	for _, p := range smallPrimes {
		bp := new(big.Int).SetUint64(p)
		for {
			q, m := new(big.Int).QuoRem(n, bp, &r)
			if m.Sign() != 0 {
				break
			}
			fs = append(fs, bp)
			n = q
		}
	}
	return splitBig(n, fs)
}

func splitBig(n *big.Int, fs []*big.Int) []*big.Int {
	if n.Cmp(one) == 0 {
		return fs
	}
	if n.IsUint64() {
		for _, p := range factor64(n.Uint64()) {
			fs = append(fs, new(big.Int).SetUint64(p))
		}
		return fs
	}
	if n.ProbablyPrime(25) {
		return append(fs, n)
	}
	d := rhoBig(n)
	fs = splitBig(d, fs)
	return splitBig(new(big.Int).Quo(n, d), fs)
}

// rhoBig is rho64 on big integers.
func rhoBig(n *big.Int) *big.Int {
	x, y, ys, q, g, diff := new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	for c := int64(1); ; c++ {
		bc := big.NewInt(c)
		f := func(v *big.Int) {
			v.Mul(v, v)
			v.Add(v, bc)
			v.Mod(v, n)
		}
		y.SetInt64(2)
		q.SetInt64(1)
		g.SetInt64(1)
		const m = 128
		for r := 1; g.Cmp(one) == 0; r <<= 1 {
			x.Set(y)
			for range r {
				f(y)
			}
			for k := 0; k < r && g.Cmp(one) == 0; k += m {
				ys.Set(y)
				for range min(m, r-k) {
					f(y)
					diff.Sub(x, y)
					diff.Abs(diff)
					q.Mul(q, diff)
					q.Mod(q, n)
				}
				g.GCD(nil, nil, q, n)
			}
		}
		if g.Cmp(n) == 0 {
			for g.SetInt64(1); g.Cmp(one) == 0; {
				f(ys)
				diff.Sub(x, ys)
				diff.Abs(diff)
				g.GCD(nil, nil, diff, n)
			}
		}
		if g.Cmp(n) != 0 {
			return new(big.Int).Set(g)
		}
	}
}
