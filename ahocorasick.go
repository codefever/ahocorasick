package ahocorasick

import (
	"log"
	"sort"
)

const blockSize = 256

// Builder is an interface to create AC.
type Builder struct {
	// input
	words      []string
	wordValues []interface{}

	// tries
	base       []int // reused to store value index when represented '\0'
	check      []int
	suffixLink []int
	values     []interface{}

	entries   []*entryState
	headEntry *entryState
}

// Searcher is an interface to search over AC.
type Searcher struct {
	base       []int
	check      []int
	suffixLink []int
	values     []interface{}
}

type entryState struct {
	used  bool
	index int
	next  *entryState
	prev  *entryState
}

func newEntryState() *entryState {
	var es entryState
	es.init()
	return &es
}

func (es *entryState) init() {
	es.used = false
	es.index = -1
	es.next = es
	es.prev = es
}

func (es *entryState) unlink() {
	es.next.prev = es.prev
	es.prev.next = es.next
}

func (es *entryState) linkAsNext(other *entryState) {
	other.next = es.next
	other.prev = es
	es.next.prev = other
	es.next = other
}

func (es *entryState) linkAsPrev(other *entryState) {
	other.prev = es.prev
	other.next = es
	es.prev.next = other
	es.prev = other
}

type wordSorter struct {
	words  []string
	values []interface{}
}

func (ws *wordSorter) Len() int {
	return sort.StringSlice(ws.words).Len()
}

func (ws *wordSorter) Less(i, j int) bool {
	return sort.StringSlice(ws.words).Less(i, j)
}

func (ws *wordSorter) Swap(i, j int) {
	sort.StringSlice(ws.words).Swap(i, j)
	ws.values[i], ws.values[j] = ws.values[j], ws.values[i]
}

// NewBuilder creates a new AC builder
func NewBuilder() *Builder {
	return &Builder{headEntry: newEntryState()}
}

// Add inserts candidate words
func (b *Builder) Add(word string, value interface{}) *Builder {
	if len(word) == 0 {
		panic("Add empty word.")
	}
	b.words = append(b.words, word)
	b.wordValues = append(b.wordValues, value)
	return b
}

// Build create a new searcher from the builder
func (b *Builder) Build() *Searcher {
	sort.StringSlice(b.words).Sort()
	b.values = make([]interface{}, 1) // 1-st not used
	b.extendBlocks()
	b.buildLevel(0, len(b.words), 0, 0)
	b.buildSuffixLinks()
	return &Searcher{b.base, b.check, b.suffixLink, b.values}
}

func (b *Builder) extendBlocks() {
	start := len(b.base)
	for i := 0; i < blockSize; i++ {
		b.base = append(b.base, 0)
		b.check = append(b.check, -1)
		b.suffixLink = append(b.suffixLink, 0)

		es := newEntryState()
		es.index = start + i
		b.entries = append(b.entries, es)
		b.headEntry.linkAsPrev(es)
	}
}

func (b *Builder) buildLevel(begin, end, depth, state int) {
	var labels []byte
	var bs []int
	for i := begin; i < end; i++ {
		c := b.getCharacter(i, depth)
		if len(labels) == 0 || labels[len(labels)-1] != c {
			if len(labels) > 0 && labels[len(labels)-1] > c {
				panic("Words not sorted?")
			}
			labels = append(labels, c)
			bs = append(bs, i)
		}
	}
	bs = append(bs, end)

	// Lock states
	next := b.findNextPosition(labels)
	b.base[state] = next
	for _, l := range labels {
		nc := next + int(l)
		b.check[nc] = state
	}

	// Go depth
	for i, l := range labels {
		nc := next + int(l)
		if l == 0 {
			// save value
			b.base[nc] = len(b.values)
			b.values = append(b.values, b.wordValues[bs[i]])
			if bs[i+1]-bs[i] > 1 {
				log.Printf("skip duplicated value for word: %v", b.words[bs[i]])
			}
			continue
		}
		b.buildLevel(bs[i], bs[i+1], depth+1, nc)
	}
}

type suffixLink struct {
	state int
	begin int
	end   int
}

func (b *Builder) buildSuffixLinks() {
	var depth int
	q := make([]suffixLink, 0)
	q = append(q, suffixLink{0, 0, len(b.words)})
	for len(q) > 0 {
		nextQ := make([]suffixLink, 0)
		for _, sl := range q {
			var labels []byte
			var bs []int
			for i := sl.begin; i < sl.end; i++ {
				c := b.getCharacter(i, depth)
				if len(labels) == 0 || labels[len(labels)-1] != c {
					if len(labels) > 0 && labels[len(labels)-1] > c {
						panic("Words not sorted?")
					}
					labels = append(labels, c)
					bs = append(bs, i)
				}
			}
			bs = append(bs, sl.end)

			// create links and go next depth
			next := b.base[sl.state]
			for i, l := range labels {
				nc := next + int(l)
				if sl.state != 0 {
					b.createSuffixLink(sl.state, nc, l)
				}
				if l == 0 {
					continue
				}
				nextQ = append(nextQ, suffixLink{nc, bs[i], bs[i+1]})
			}
		}
		depth++
		q = nextQ
	}
}

func (b *Builder) createSuffixLink(state, childState int, c byte) {
	suffix := b.suffixLink[state]
	// do while?
	for {
		tmp := b.base[suffix] + int(c)
		if b.check[tmp] == suffix {
			b.suffixLink[childState] = tmp
			break
		}
		if suffix == 0 {
			break
		}
		suffix = b.suffixLink[suffix]
	}
}

func (b *Builder) getCharacter(i, j int) byte {
	if j < len(b.words[i]) {
		c := b.words[i][j]
		if c == 0 {
			panic("Word contains '\\0'")
		}
		return c
	}
	return 0
}

func (b *Builder) findNextPosition(labels []byte) int {
	impl := func(startEntry, endEntry *entryState) int {
		for es := startEntry; es != endEntry; es = es.next {
			if es.used || es.index < 0 {
				panic("invalid entry but in links")
			}
			i := es.index
			// check length
			if i+int(labels[len(labels)-1]) >= len(b.entries) {
				break
			}
			// match all the labels
			ok := true
			for _, l := range labels {
				nc := i + int(l)
				if b.entries[nc].used {
					ok = false
					break
				}
			}
			if ok {
				return i
			}
		}
		return -1
	}

	p := -1
	startEntry := b.headEntry.next
	lastEntry := b.headEntry.prev
	for i := 0; ; i++ {
		p = impl(startEntry, b.headEntry)
		if p >= 0 {
			break
		}
		if p < 0 && i >= 1 {
			panic("cannot find next pos?")
		}

		atLeastIndex := len(b.base) - int(labels[len(labels)-1])
		b.extendBlocks()
		for lastEntry != b.headEntry && lastEntry.index >= atLeastIndex {
			lastEntry = lastEntry.prev
		}
		startEntry = lastEntry.next
	}

	for _, l := range labels {
		nc := p + int(l)
		b.entries[nc].used = true
		b.entries[nc].unlink()
	}
	b.entries[p].used = true
	b.entries[p].unlink()
	return p
}

func (s *Searcher) prefixSearch(word string) (int, bool) {
	state := 0
	bytes := []byte(word)
	for _, c := range bytes {
		nextState := s.base[state] + int(c)
		if nextState >= len(s.check) || s.check[nextState] != state {
			return -1, false
		}
		state = nextState
	}
	return state, true
}

// Search returns true if there's a exactly match.
func (s *Searcher) Search(word string) (bool, interface{}) {
	state, ok := s.prefixSearch(word)
	if !ok {
		return false, false
	}
	nextState := s.base[state]
	if nextState < len(s.check) && s.check[nextState] == state {
		return true, s.values[s.base[nextState]]
	}
	return false, nil
}

// PrefixSearch returns true if some words which are prefix for the given `word`.
func (s *Searcher) PrefixSearch(word string) bool {
	_, ok := s.prefixSearch(word)
	return ok
}

// Cover returns all the values of words which are covered by thte given `text`.
func (s *Searcher) Cover(text string) []interface{} {
	ret := make([]interface{}, 0)
	state := 0
	seen := make(map[int]struct{})
	bytes := []byte(text)
	for _, c := range bytes {
		for {
			nextState := s.base[state] + int(c)
			if nextState < len(s.check) && s.check[nextState] == state {
				state = nextState
				break
			}
			if state == 0 {
				break
			}
			state = s.suffixLink[state]
		}

		checkState := state
		for {
			if _, ok := seen[checkState]; ok {
				break
			}
			seen[checkState] = struct{}{}
			endState := s.base[checkState] + 0
			if s.check[endState] == checkState {
				if val := s.values[s.base[endState]]; val != nil {
					ret = append(ret, val)
				}
			}
			checkState = s.suffixLink[checkState]
		}
	}
	return ret
}
