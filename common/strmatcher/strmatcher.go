package strmatcher

import (
	"regexp"
)

// Matcher is the interface to determine a string matches a pattern.
type Matcher interface {
	// Match returns true if the given string matches a predefined pattern.
	Match(string) bool
	String() string
}

// Type is the type of the matcher.
type Type byte

const (
	// Full is the type of matcher that the input string must exactly equal to the pattern.
	Full Type = iota
	// Substr is the type of matcher that the input string must contain the pattern as a sub-string.
	Substr
	// Domain is the type of matcher that the input string must be a sub-domain or itself of the pattern.
	Domain
	// Regex is the type of matcher that the input string must matches the regular-expression pattern.
	Regex
)

// New creates a new Matcher based on the given pattern.
func (t Type) New(pattern string) (Matcher, error) {
	switch t {
	case Full:
		return fullMatcher(pattern), nil
	case Substr:
		return substrMatcher(pattern), nil
	case Domain:
		return domainMatcher(pattern), nil
	case Regex:
		r, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return &regexMatcher{
			pattern: r,
		}, nil
	default:
		panic("Unknown type")
	}
}

// IndexMatcher is the interface for matching with a group of matchers.
type IndexMatcher interface {
	// Match returns the index of a matcher that matches the input. It returns empty array if no such matcher exists.
	Match(input string) []uint32
}

type matcherEntry struct {
	m  Matcher
	id uint32
}

type ACAutomatonMatcherGroup struct {
	count         uint32
	ac            *ACAutomaton
	otherMatchers []matcherEntry
}

func NewACAutomatonMatcherGroup() *ACAutomatonMatcherGroup {
	var g = new(ACAutomatonMatcherGroup)
	g.count = 1
	g.ac = NewACAutomaton()
	return g
}

func (g *ACAutomatonMatcherGroup) AddPattern(pattern string, t Type) (uint32, error) {
	switch t {
	case Full, Substr, Domain:
		g.ac.Add(pattern, t)
	case Regex:
		g.count++
		r, err := regexp.Compile(pattern)
		if err != nil {
			return 0, err
		}
		g.otherMatchers = append(g.otherMatchers, matcherEntry{
			m:  &regexMatcher{pattern: r},
			id: g.count,
		})
	default:
		panic("Unknown type")
	}
	return g.count, nil
}

func (g *ACAutomatonMatcherGroup) Build() {
	g.ac.Build()
}

// Match implements IndexMatcher.Match.
func (g *ACAutomatonMatcherGroup) Match(pattern string) []uint32 {
	result := []uint32{}
	if g.ac.Match(pattern) {
		result = append(result, 1)
	}
	for _, e := range g.otherMatchers {
		if e.m.Match(pattern) {
			result = append(result, e.id)
		}
	}
	return result
}

// MatcherGroup is an implementation of IndexMatcher.
// Empty initialization works.
type MatcherGroup struct {
	count         uint32
	fullMatcher   FullMatcherGroup
	domainMatcher DomainMatcherGroup
	otherMatchers []matcherEntry
}

// Add adds a new Matcher into the MatcherGroup, and returns its index. The index will never be 0.
func (g *MatcherGroup) Add(m Matcher) uint32 {
	g.count++
	c := g.count

	switch tm := m.(type) {
	case fullMatcher:
		g.fullMatcher.addMatcher(tm, c)
	case domainMatcher:
		g.domainMatcher.addMatcher(tm, c)
	default:
		g.otherMatchers = append(g.otherMatchers, matcherEntry{
			m:  m,
			id: c,
		})
	}

	return c
}

// Match implements IndexMatcher.Match.
func (g *MatcherGroup) Match(pattern string) []uint32 {
	result := []uint32{}
	result = append(result, g.fullMatcher.Match(pattern)...)
	result = append(result, g.domainMatcher.Match(pattern)...)
	for _, e := range g.otherMatchers {
		if e.m.Match(pattern) {
			result = append(result, e.id)
		}
	}
	return result
}

// Size returns the number of matchers in the MatcherGroup.
func (g *MatcherGroup) Size() uint32 {
	return g.count
}
