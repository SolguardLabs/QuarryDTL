package main

import "fmt"

type Balance struct {
	Asset     string `json:"asset"`
	Available Amount `json:"available"`
	Reserved  Amount `json:"reserved"`
	Locked    Amount `json:"locked"`
}

func (b Balance) Total() Amount {
	return b.Available + b.Reserved + b.Locked
}

func (b Balance) NonNegative() bool {
	return b.Available >= 0 && b.Reserved >= 0 && b.Locked >= 0
}

type Account struct {
	ID       string             `json:"id"`
	Role     string             `json:"role"`
	Balances map[string]Balance `json:"balances"`
	Flags    LabelSet           `json:"flags,omitempty"`
}

func NewAccount(id string, role string) *Account {
	return &Account{
		ID:       id,
		Role:     role,
		Balances: map[string]Balance{},
		Flags:    LabelSet{},
	}
}

func (a *Account) ensure(asset string) Balance {
	balance, ok := a.Balances[asset]
	if !ok {
		balance = Balance{Asset: asset}
	}
	return balance
}

func (a *Account) set(balance Balance) {
	a.Balances[balance.Asset] = balance
}

func (a *Account) Deposit(asset string, amount Amount) error {
	if err := MustPositiveAmount("deposit", amount); err != nil {
		return err
	}
	balance := a.ensure(asset)
	balance.Available += amount
	a.set(balance)
	return nil
}

func (a *Account) Credit(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("credit cannot be negative")
	}
	balance := a.ensure(asset)
	balance.Available += amount
	a.set(balance)
	return nil
}

func (a *Account) Debit(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("debit cannot be negative")
	}
	balance := a.ensure(asset)
	if balance.Available < amount {
		return fmt.Errorf("insufficient available balance in account %s", a.ID)
	}
	balance.Available -= amount
	a.set(balance)
	return nil
}

func (a *Account) Reserve(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("reserve cannot be negative")
	}
	balance := a.ensure(asset)
	if balance.Available < amount {
		return fmt.Errorf("insufficient account liquidity in %s", a.ID)
	}
	balance.Available -= amount
	balance.Reserved += amount
	a.set(balance)
	return nil
}

func (a *Account) Release(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("release cannot be negative")
	}
	balance := a.ensure(asset)
	if balance.Reserved < amount {
		return fmt.Errorf("reserved balance too small in account %s", a.ID)
	}
	balance.Reserved -= amount
	balance.Available += amount
	a.set(balance)
	return nil
}

func (a *Account) Lock(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("lock cannot be negative")
	}
	balance := a.ensure(asset)
	if balance.Available < amount {
		return fmt.Errorf("insufficient unlocked balance in account %s", a.ID)
	}
	balance.Available -= amount
	balance.Locked += amount
	a.set(balance)
	return nil
}

func (a *Account) Unlock(asset string, amount Amount) error {
	if amount < 0 {
		return fmt.Errorf("unlock cannot be negative")
	}
	balance := a.ensure(asset)
	if balance.Locked < amount {
		return fmt.Errorf("locked balance too small in account %s", a.ID)
	}
	balance.Locked -= amount
	balance.Available += amount
	a.set(balance)
	return nil
}

func (a *Account) Balance(asset string) Balance {
	return a.ensure(asset)
}

func (a *Account) Snapshot() Account {
	copy := NewAccount(a.ID, a.Role)
	for asset, balance := range a.Balances {
		copy.Balances[asset] = balance
	}
	copy.Flags = a.Flags.Clone()
	return *copy
}

type AccountBook struct {
	accounts map[string]*Account
	order    []string
}

func NewAccountBook() *AccountBook {
	return &AccountBook{accounts: map[string]*Account{}, order: []string{}}
}

func (b *AccountBook) Add(account *Account) error {
	if account == nil || account.ID == "" {
		return fmt.Errorf("account missing")
	}
	if _, ok := b.accounts[account.ID]; !ok {
		b.order = append(b.order, account.ID)
	}
	b.accounts[account.ID] = account
	return nil
}

func (b *AccountBook) MustAdd(account *Account) {
	if err := b.Add(account); err != nil {
		panic(err)
	}
}

func (b *AccountBook) Get(id string) (*Account, bool) {
	account, ok := b.accounts[id]
	return account, ok
}

func (b *AccountBook) MustGet(id string) *Account {
	account, ok := b.Get(id)
	if !ok {
		panic("missing account " + id)
	}
	return account
}

func (b *AccountBook) List() []Account {
	out := make([]Account, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, b.accounts[id].Snapshot())
	}
	return out
}

func (b *AccountBook) Total(asset string) Amount {
	total := Amount(0)
	for _, account := range b.accounts {
		total += account.Balance(asset).Total()
	}
	return total
}

func (b *AccountBook) AllNonNegative() bool {
	for _, account := range b.accounts {
		for _, balance := range account.Balances {
			if !balance.NonNegative() {
				return false
			}
		}
	}
	return true
}
