package main

import "fmt"

type VaultStatus string

const (
	VaultActive   VaultStatus = "active"
	VaultDraining VaultStatus = "draining"
	VaultPaused   VaultStatus = "paused"
)

type Vault struct {
	ID            string      `json:"id"`
	Asset         string      `json:"asset"`
	Reserve       Amount      `json:"reserve"`
	Idle          Amount      `json:"idle"`
	Reserved      Amount      `json:"reserved"`
	InFlight      Amount      `json:"in_flight"`
	Liability     Amount      `json:"liability"`
	PendingIn     Amount      `json:"pending_in"`
	PendingOut    Amount      `json:"pending_out"`
	ShareSupply   Amount      `json:"share_supply"`
	MinBuffer     Amount      `json:"min_buffer"`
	HealthBps     int64       `json:"health_bps"`
	Priority      int         `json:"priority"`
	Status        VaultStatus `json:"status"`
	Strategy      string      `json:"strategy"`
	LastRebalance int64       `json:"last_rebalance"`
}

func NewVault(id string, asset string, reserve Amount) *Vault {
	return &Vault{
		ID:          id,
		Asset:       asset,
		Reserve:     reserve,
		Idle:        reserve,
		ShareSupply: reserve,
		HealthBps:   10_000,
		Status:      VaultActive,
		Strategy:    "balanced",
	}
}

func (v *Vault) Validate() error {
	if v.ID == "" || v.Asset == "" {
		return fmt.Errorf("vault missing id or asset")
	}
	if v.Reserve < 0 || v.Idle < 0 || v.Reserved < 0 || v.InFlight < 0 {
		return fmt.Errorf("vault %s has negative reserve component", v.ID)
	}
	if v.Idle+v.Reserved+v.InFlight > v.Reserve+v.PendingIn {
		return fmt.Errorf("vault %s components exceed reserve view", v.ID)
	}
	return nil
}

func (v *Vault) Available() Amount {
	available := v.Idle - v.MinBuffer
	if available < 0 {
		return 0
	}
	return available
}

func (v *Vault) EconomicEquity() Amount {
	return v.Reserve + v.PendingIn - v.PendingOut - v.Liability
}

func (v *Vault) CoverageBps() int64 {
	if v.Liability <= 0 {
		return 10_000
	}
	return int64(v.Reserve) * 10_000 / int64(v.Liability)
}

func (v *Vault) Deposit(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("deposit cannot be negative")
	}
	v.Reserve += amount
	v.Idle += amount
	v.ShareSupply += amount
	return nil
}

func (v *Vault) ReserveFor(kind string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("reservation cannot be negative")
	}
	if v.Available() < amount {
		return fmt.Errorf("vault %s has insufficient idle liquidity for %s", v.ID, kind)
	}
	v.Idle -= amount
	v.Reserved += amount
	return nil
}

func (v *Vault) ReleaseReservation(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("release cannot be negative")
	}
	if v.Reserved < amount {
		return fmt.Errorf("vault %s reserved amount too small", v.ID)
	}
	v.Reserved -= amount
	v.Idle += amount
	return nil
}

func (v *Vault) OpenOutflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("outflow cannot be negative")
	}
	if v.Reserved < amount {
		return fmt.Errorf("vault %s reserved amount too small for outflow", v.ID)
	}
	v.Reserved -= amount
	v.InFlight += amount
	v.PendingOut += amount
	return nil
}

func (v *Vault) CompleteOutflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("complete outflow cannot be negative")
	}
	if v.InFlight < amount || v.PendingOut < amount {
		return fmt.Errorf("vault %s outflow view too small", v.ID)
	}
	if v.Reserve < amount {
		return fmt.Errorf("vault %s reserve too small", v.ID)
	}
	v.InFlight -= amount
	v.PendingOut -= amount
	v.Reserve -= amount
	return nil
}

func (v *Vault) RevertOutflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("revert outflow cannot be negative")
	}
	if v.InFlight < amount || v.PendingOut < amount {
		return fmt.Errorf("vault %s cannot revert outflow", v.ID)
	}
	v.InFlight -= amount
	v.PendingOut -= amount
	v.Idle += amount
	return nil
}

func (v *Vault) ExpectInflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("inflow cannot be negative")
	}
	v.PendingIn += amount
	return nil
}

func (v *Vault) CompleteInflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("complete inflow cannot be negative")
	}
	if v.PendingIn < amount {
		return fmt.Errorf("vault %s pending inflow too small", v.ID)
	}
	v.PendingIn -= amount
	v.Reserve += amount
	v.Idle += amount
	return nil
}

func (v *Vault) CancelInflow(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("cancel inflow cannot be negative")
	}
	if v.PendingIn < amount {
		return fmt.Errorf("vault %s pending inflow too small", v.ID)
	}
	v.PendingIn -= amount
	return nil
}

func (v *Vault) AddLiability(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("liability cannot be negative")
	}
	v.Liability += amount
	return nil
}

func (v *Vault) BurnLiability(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("burn liability cannot be negative")
	}
	if v.Liability < amount {
		return fmt.Errorf("vault %s liability too small", v.ID)
	}
	v.Liability -= amount
	return nil
}

func (v *Vault) ApplyPenalty(amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("penalty cannot be negative")
	}
	if v.Reserve < amount {
		amount = v.Reserve
	}
	if v.Idle >= amount {
		v.Idle -= amount
	} else {
		remain := amount - v.Idle
		v.Idle = 0
		if v.Reserved >= remain {
			v.Reserved -= remain
		} else {
			v.Reserved = 0
		}
	}
	v.Reserve -= amount
	return nil
}

func (v *Vault) Snapshot() Vault {
	return *v
}

type VaultBook struct {
	vaults map[string]*Vault
	order  []string
}

func NewVaultBook() *VaultBook {
	return &VaultBook{vaults: map[string]*Vault{}, order: []string{}}
}

func (b *VaultBook) Add(vault *Vault) error {
	if vault == nil {
		return fmt.Errorf("vault missing")
	}
	if err := vault.Validate(); err != nil {
		return err
	}
	if _, ok := b.vaults[vault.ID]; !ok {
		b.order = append(b.order, vault.ID)
	}
	b.vaults[vault.ID] = vault
	return nil
}

func (b *VaultBook) MustAdd(vault *Vault) {
	if err := b.Add(vault); err != nil {
		panic(err)
	}
}

func (b *VaultBook) Get(id string) (*Vault, bool) {
	vault, ok := b.vaults[id]
	return vault, ok
}

func (b *VaultBook) MustGet(id string) *Vault {
	vault, ok := b.Get(id)
	if !ok {
		panic("missing vault " + id)
	}
	return vault
}

func (b *VaultBook) List() []Vault {
	out := make([]Vault, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, b.vaults[id].Snapshot())
	}
	return out
}

func (b *VaultBook) TotalReserve(asset string) Amount {
	total := Amount(0)
	for _, vault := range b.vaults {
		if vault.Asset == asset {
			total += vault.Reserve
		}
	}
	return total
}

func (b *VaultBook) TotalLiability(asset string) Amount {
	total := Amount(0)
	for _, vault := range b.vaults {
		if vault.Asset == asset {
			total += vault.Liability
		}
	}
	return total
}

func (b *VaultBook) AllNonNegative() bool {
	for _, vault := range b.vaults {
		if vault.Reserve < 0 || vault.Idle < 0 || vault.Reserved < 0 || vault.InFlight < 0 || vault.PendingIn < 0 || vault.PendingOut < 0 {
			return false
		}
	}
	return true
}
