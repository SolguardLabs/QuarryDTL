package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)

type ID string

func NewID(value string) (ID, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if !idPattern.MatchString(value) {
		return "", fmt.Errorf("invalid id %q", value)
	}
	return ID(value), nil
}

func MustID(value string) ID {
	id, err := NewID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ID) String() string {
	return string(id)
}

func (id ID) Empty() bool {
	return string(id) == ""
}

func (id ID) WithSuffix(suffix string) ID {
	raw := strings.TrimSuffix(id.String(), "-") + "-" + strings.Trim(strings.ToLower(suffix), "-")
	out, err := NewID(raw)
	if err != nil {
		return id
	}
	return out
}

type Sequencer struct {
	prefix   string
	counters map[string]int
}

func NewSequencer(prefix string) *Sequencer {
	return &Sequencer{
		prefix:   strings.Trim(strings.ToLower(prefix), "-"),
		counters: map[string]int{},
	}
}

func (s *Sequencer) Next(kind string) string {
	kind = strings.Trim(strings.ToLower(kind), "-")
	s.counters[kind]++
	if s.prefix == "" {
		return fmt.Sprintf("%s-%04d", kind, s.counters[kind])
	}
	return fmt.Sprintf("%s-%s-%04d", s.prefix, kind, s.counters[kind])
}

func (s *Sequencer) Snapshot() map[string]int {
	out := make(map[string]int, len(s.counters))
	for key, value := range s.counters {
		out[key] = value
	}
	return out
}

func StablePair(left string, right string) string {
	if left < right {
		return left + ":" + right
	}
	return right + ":" + left
}

func StableJoin(values ...string) string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		clean = append(clean, strings.TrimSpace(value))
	}
	return strings.Join(clean, "|")
}

func SortedKeys[T any](input map[string]T) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type LabelSet map[string]string

func (l LabelSet) Clone() LabelSet {
	out := LabelSet{}
	for key, value := range l {
		out[key] = value
	}
	return out
}

func (l LabelSet) Canonical() string {
	if len(l) == 0 {
		return ""
	}
	keys := SortedKeys(l)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+l[key])
	}
	return strings.Join(parts, ";")
}

func (l LabelSet) Get(key string) string {
	if l == nil {
		return ""
	}
	return l[key]
}

func (l LabelSet) Set(key string, value string) {
	if l == nil {
		return
	}
	l[key] = value
}

type NamedRef struct {
	ID     string   `json:"id"`
	Labels LabelSet `json:"labels,omitempty"`
}

func Ref(id string, labels LabelSet) NamedRef {
	return NamedRef{ID: id, Labels: labels.Clone()}
}
