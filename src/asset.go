package main

import (
	"fmt"
	"sort"
)

type AssetKind string

const (
	AssetStable     AssetKind = "stable"
	AssetCollateral AssetKind = "collateral"
	AssetSynthetic  AssetKind = "synthetic"
)

type Asset struct {
	ID             string    `json:"id"`
	Symbol         string    `json:"symbol"`
	Decimals       int       `json:"decimals"`
	Kind           AssetKind `json:"kind"`
	RiskWeightBps  int64     `json:"risk_weight_bps"`
	SettlementBps  int64     `json:"settlement_bps"`
	LiquidationBps int64     `json:"liquidation_bps"`
}

func NewAsset(id, symbol string, decimals int, kind AssetKind) Asset {
	return Asset{
		ID:             id,
		Symbol:         symbol,
		Decimals:       decimals,
		Kind:           kind,
		RiskWeightBps:  10_000,
		SettlementBps:  4,
		LiquidationBps: 75,
	}
}

func (a Asset) Fee(amount Amount) Amount {
	return amount.MulBps(a.SettlementBps)
}

func (a Asset) LiquidationPenalty(amount Amount) Amount {
	return amount.MulBps(a.LiquidationBps)
}

func (a Asset) Weighted(amount Amount) Amount {
	return amount.MulBps(a.RiskWeightBps)
}

func (a Asset) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("asset id missing")
	}
	if a.Symbol == "" {
		return fmt.Errorf("asset symbol missing")
	}
	if a.Decimals < 0 || a.Decimals > 18 {
		return fmt.Errorf("asset %s decimals out of range", a.ID)
	}
	if a.RiskWeightBps <= 0 {
		return fmt.Errorf("asset %s risk weight must be positive", a.ID)
	}
	return nil
}

type AssetRegistry struct {
	assets map[string]Asset
	order  []string
}

func NewAssetRegistry() *AssetRegistry {
	return &AssetRegistry{assets: map[string]Asset{}, order: []string{}}
}

func (r *AssetRegistry) Add(asset Asset) error {
	if err := asset.Validate(); err != nil {
		return err
	}
	if _, ok := r.assets[asset.ID]; !ok {
		r.order = append(r.order, asset.ID)
		sort.Strings(r.order)
	}
	r.assets[asset.ID] = asset
	return nil
}

func (r *AssetRegistry) MustAdd(asset Asset) {
	if err := r.Add(asset); err != nil {
		panic(err)
	}
}

func (r *AssetRegistry) Get(id string) (Asset, bool) {
	asset, ok := r.assets[id]
	return asset, ok
}

func (r *AssetRegistry) MustGet(id string) Asset {
	asset, ok := r.Get(id)
	if !ok {
		panic("missing asset " + id)
	}
	return asset
}

func (r *AssetRegistry) List() []Asset {
	out := make([]Asset, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.assets[id])
	}
	return out
}

func (r *AssetRegistry) IDs() []string {
	out := append([]string{}, r.order...)
	return out
}

func (r *AssetRegistry) Convert(amount Amount, source string, target string) Amount {
	if source == target {
		return amount
	}
	src := r.MustGet(source)
	dst := r.MustGet(target)
	shift := dst.Decimals - src.Decimals
	value := int64(amount)
	if shift > 0 {
		for i := 0; i < shift; i++ {
			value *= 10
		}
	}
	if shift < 0 {
		for i := 0; i < -shift; i++ {
			value /= 10
		}
	}
	return Amount(value)
}
