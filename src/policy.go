package main

import "fmt"

type ProtocolPolicy struct {
	MaxRouteShareBps         int64  `json:"max_route_share_bps"`
	MinVaultBufferBps        int64  `json:"min_vault_buffer_bps"`
	ReservationTTL           int64  `json:"reservation_ttl"`
	ForecastConfidenceBps    int64  `json:"forecast_confidence_bps"`
	ProjectedCapacityEnabled bool   `json:"projected_capacity_enabled"`
	LiquidationThresholdBps  int64  `json:"liquidation_threshold_bps"`
	RecoveryPenaltyBps       int64  `json:"recovery_penalty_bps"`
	MaxSettlementLatency     int64  `json:"max_settlement_latency"`
	ProtocolAccount          string `json:"protocol_account"`
}

func DefaultPolicy() ProtocolPolicy {
	return ProtocolPolicy{
		MaxRouteShareBps:         6_500,
		MinVaultBufferBps:        700,
		ReservationTTL:           6,
		ForecastConfidenceBps:    9_000,
		ProjectedCapacityEnabled: true,
		LiquidationThresholdBps:  9_250,
		RecoveryPenaltyBps:       250,
		MaxSettlementLatency:     8,
		ProtocolAccount:          "protocol",
	}
}

func (p ProtocolPolicy) Validate() error {
	if p.MaxRouteShareBps <= 0 || p.MaxRouteShareBps > 10_000 {
		return fmt.Errorf("max route share out of range")
	}
	if p.MinVaultBufferBps < 0 || p.MinVaultBufferBps > 5_000 {
		return fmt.Errorf("min vault buffer out of range")
	}
	if p.ReservationTTL <= 0 {
		return fmt.Errorf("reservation ttl must be positive")
	}
	if p.ForecastConfidenceBps < 0 || p.ForecastConfidenceBps > 10_000 {
		return fmt.Errorf("forecast confidence out of range")
	}
	if p.LiquidationThresholdBps <= 0 || p.LiquidationThresholdBps > 15_000 {
		return fmt.Errorf("liquidation threshold out of range")
	}
	if p.MaxSettlementLatency <= 0 {
		return fmt.Errorf("settlement latency must be positive")
	}
	return nil
}

func (p ProtocolPolicy) RouteCap(vault *Vault, route *Route) Amount {
	if route.MaxAllocation > 0 {
		byRoute := route.MaxAllocation - route.Allocated
		byVault := vault.Reserve.MulBps(p.MaxRouteShareBps)
		return byRoute.Min(byVault).NonNegative()
	}
	return vault.Reserve.MulBps(p.MaxRouteShareBps)
}

func (p ProtocolPolicy) MinBuffer(vault *Vault) Amount {
	return vault.Reserve.MulBps(p.MinVaultBufferBps)
}

func (p ProtocolPolicy) AdmitsForecast(confidence int64) bool {
	return p.ProjectedCapacityEnabled && confidence >= p.ForecastConfidenceBps
}

func (p ProtocolPolicy) SettlementLatency(route *Route) int64 {
	if route.ExpectedLatency <= 0 {
		return 1
	}
	if route.ExpectedLatency > p.MaxSettlementLatency {
		return p.MaxSettlementLatency
	}
	return route.ExpectedLatency
}

func (p ProtocolPolicy) Liquidatable(vault *Vault) bool {
	return vault.CoverageBps() < p.LiquidationThresholdBps
}

func (p ProtocolPolicy) RecoveryPenalty(amount Amount) Amount {
	return amount.MulBps(p.RecoveryPenaltyBps)
}

type CapacityMode string

const (
	CapacityObserved  CapacityMode = "observed"
	CapacityProjected CapacityMode = "projected"
)

type CapacityProfile struct {
	VaultID       string       `json:"vault_id"`
	Asset         string       `json:"asset"`
	Mode          CapacityMode `json:"mode"`
	Free          Amount       `json:"free"`
	Reserved      Amount       `json:"reserved"`
	InFlight      Amount       `json:"in_flight"`
	PendingIn     Amount       `json:"pending_in"`
	PendingOut    Amount       `json:"pending_out"`
	Forecast      Amount       `json:"forecast"`
	Assignable    Amount       `json:"assignable"`
	MinBuffer     Amount       `json:"min_buffer"`
	ConfidenceBps int64        `json:"confidence_bps"`
}

func (p CapacityProfile) Has(amount Amount) bool {
	return p.Assignable >= amount
}

func (p CapacityProfile) UsesProjection() bool {
	return p.Mode == CapacityProjected && p.Forecast > 0
}

func (p CapacityProfile) HeadroomAfter(amount Amount) Amount {
	if amount > p.Assignable {
		return 0
	}
	return p.Assignable - amount
}

func (p CapacityProfile) Weight() int64 {
	if p.Assignable <= 0 {
		return 0
	}
	if p.ConfidenceBps <= 0 {
		return int64(p.Assignable)
	}
	return int64(p.Assignable) * p.ConfidenceBps / 10_000
}
