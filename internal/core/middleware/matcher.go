package middleware

import (
	"sort"
	"strings"
)

type Matcher interface {
	Use(ms ...Middleware)
	Add(selector string, ms ...Middleware)
	Match(operation string) []Middleware
}

func New() Matcher {
	return &matcher{
		matchs: make(map[string][]Middleware),
	}
}

type matcher struct {
	prefix   []string
	defaults []Middleware
	matchs   map[string][]Middleware
}

func (m *matcher) Use(ms ...Middleware) {
	m.defaults = ms
}

func (m *matcher) Add(selector string, ms ...Middleware) {
	if strings.HasSuffix(selector, "*") {
		selector = strings.TrimSuffix(selector, "*")
		m.prefix = append(m.prefix, selector)
		sort.Slice(m.prefix, func(i, j int) bool {
			return m.prefix[i] > m.prefix[j]
		})
	}
	m.matchs[selector] = ms
}

func (m *matcher) Match(operation string) []Middleware {
	ms := make([]Middleware, 0, len(m.defaults))
	if len(m.defaults) > 0 {
		ms = append(ms, m.defaults...)
	}
	if next, ok := m.matchs[operation]; ok {
		return append(ms, next...)
	}
	for _, prefix := range m.prefix {
		if strings.HasPrefix(operation, prefix) {
			return append(ms, m.matchs[prefix]...)
		}
	}
	return ms
}
