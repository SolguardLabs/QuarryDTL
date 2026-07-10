package main

type Event struct {
	Index   int               `json:"index"`
	Clock   int64             `json:"clock"`
	Kind    string            `json:"kind"`
	Ref     string            `json:"ref,omitempty"`
	Vault   string            `json:"vault,omitempty"`
	Route   string            `json:"route,omitempty"`
	Asset   string            `json:"asset,omitempty"`
	Amount  Amount            `json:"amount,omitempty"`
	Message string            `json:"message,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

type Journal struct {
	events []Event
}

func NewJournal() *Journal {
	return &Journal{events: []Event{}}
}

func (j *Journal) Record(clock int64, kind string, ref string, vault string, route string, asset string, amount Amount, message string, data map[string]string) Event {
	event := Event{
		Index:   len(j.events) + 1,
		Clock:   clock,
		Kind:    kind,
		Ref:     ref,
		Vault:   vault,
		Route:   route,
		Asset:   asset,
		Amount:  amount,
		Message: message,
		Data:    data,
	}
	j.events = append(j.events, event)
	return event
}

func (j *Journal) List() []Event {
	out := make([]Event, len(j.events))
	copy(out, j.events)
	return out
}

func (j *Journal) Count(kind string) int {
	count := 0
	for _, event := range j.events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

func (j *Journal) Amount(kind string) Amount {
	total := Amount(0)
	for _, event := range j.events {
		if event.Kind == kind {
			total += event.Amount
		}
	}
	return total
}

func (j *Journal) Since(index int) []Event {
	if index < 0 {
		index = 0
	}
	if index >= len(j.events) {
		return []Event{}
	}
	out := make([]Event, len(j.events[index:]))
	copy(out, j.events[index:])
	return out
}
