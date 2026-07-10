package main

import (
	"fmt"
	"math"
	"strconv"
)

type Amount int64

const (
	FixedScale int64 = 1_000_000
	BasisPoint int64 = 10_000
)

func NewAmount(value int64) Amount {
	return Amount(value)
}

func ZeroAmount() Amount {
	return Amount(0)
}

func AmountFromUnits(units int64, decimals int) Amount {
	if decimals < 0 {
		decimals = 0
	}
	if decimals > 9 {
		decimals = 9
	}
	multiplier := int64(1)
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	return Amount(units * multiplier)
}

func ParseAmount(value string) (Amount, error) {
	if value == "" {
		return 0, fmt.Errorf("empty amount")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", value, err)
	}
	return Amount(parsed), nil
}

func (a Amount) Int64() int64 {
	return int64(a)
}

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) IsZero() bool {
	return a == 0
}

func (a Amount) Positive() bool {
	return a > 0
}

func (a Amount) Negative() bool {
	return a < 0
}

func (a Amount) Abs() Amount {
	if a < 0 {
		return -a
	}
	return a
}

func (a Amount) Add(b Amount) Amount {
	return a + b
}

func (a Amount) Sub(b Amount) Amount {
	return a - b
}

func (a Amount) CheckedAdd(b Amount) (Amount, error) {
	if b > 0 && a > Amount(math.MaxInt64)-b {
		return 0, fmt.Errorf("amount addition overflow")
	}
	if b < 0 && a < Amount(math.MinInt64)-b {
		return 0, fmt.Errorf("amount addition underflow")
	}
	return a + b, nil
}

func (a Amount) CheckedSub(b Amount) (Amount, error) {
	return a.CheckedAdd(-b)
}

func (a Amount) MulRatio(numerator, denominator int64) Amount {
	if denominator == 0 {
		return 0
	}
	return Amount((int64(a) * numerator) / denominator)
}

func (a Amount) MulBps(bps int64) Amount {
	return a.MulRatio(bps, BasisPoint)
}

func (a Amount) AddBps(bps int64) Amount {
	return a + a.MulBps(bps)
}

func (a Amount) SubBps(bps int64) Amount {
	return a - a.MulBps(bps)
}

func (a Amount) Clamp(min Amount, max Amount) Amount {
	if a < min {
		return min
	}
	if a > max {
		return max
	}
	return a
}

func (a Amount) Max(b Amount) Amount {
	if a > b {
		return a
	}
	return b
}

func (a Amount) Min(b Amount) Amount {
	if a < b {
		return a
	}
	return b
}

func (a Amount) NonNegative() Amount {
	if a < 0 {
		return 0
	}
	return a
}

func (a Amount) Format(decimals int) string {
	if decimals <= 0 {
		return a.String()
	}
	unit := int64(1)
	for i := 0; i < decimals; i++ {
		unit *= 10
	}
	whole := int64(a) / unit
	frac := int64(a) % unit
	if frac < 0 {
		frac = -frac
	}
	return fmt.Sprintf("%d.%0*d", whole, decimals, frac)
}

func SumAmounts(values []Amount) Amount {
	total := Amount(0)
	for _, value := range values {
		total += value
	}
	return total
}

func MinAmount(values ...Amount) Amount {
	if len(values) == 0 {
		return 0
	}
	best := values[0]
	for _, value := range values[1:] {
		if value < best {
			best = value
		}
	}
	return best
}

func MaxAmount(values ...Amount) Amount {
	if len(values) == 0 {
		return 0
	}
	best := values[0]
	for _, value := range values[1:] {
		if value > best {
			best = value
		}
	}
	return best
}

func MustPositiveAmount(label string, value Amount) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", label)
	}
	return nil
}

func RequireNonNegative(label string, value Amount) error {
	if value < 0 {
		return fmt.Errorf("%s cannot be negative", label)
	}
	return nil
}

func WeightedAverageAmount(total Amount, weight int64, divisor int64) Amount {
	if divisor == 0 {
		return 0
	}
	return Amount((int64(total) * weight) / divisor)
}

type AmountBucket struct {
	Asset  string `json:"asset"`
	Amount Amount `json:"amount"`
}

func Bucket(asset string, amount Amount) AmountBucket {
	return AmountBucket{Asset: asset, Amount: amount}
}

func MergeBuckets(entries []AmountBucket) []AmountBucket {
	index := map[string]Amount{}
	order := make([]string, 0)
	for _, entry := range entries {
		if _, ok := index[entry.Asset]; !ok {
			order = append(order, entry.Asset)
		}
		index[entry.Asset] += entry.Amount
	}
	out := make([]AmountBucket, 0, len(order))
	for _, asset := range order {
		out = append(out, AmountBucket{Asset: asset, Amount: index[asset]})
	}
	return out
}

func AmountMap(entries []AmountBucket) map[string]Amount {
	out := map[string]Amount{}
	for _, entry := range entries {
		out[entry.Asset] += entry.Amount
	}
	return out
}
