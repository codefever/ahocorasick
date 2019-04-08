package ahocorasick

import (
	"sort"
	"testing"
)

func TestSearch(t *testing.T) {
	builder := NewBuilder()
	words := []string{"hello", "world"}
	for _, word := range words {
		builder.Add(word, word)
	}
	searcher := builder.Build()

	// Must match
	for _, word := range words {
		ok, value := searcher.Search(word)
		if !ok {
			t.Errorf("Fail to match '%v'", word)
		}
		if value != word {
			t.Errorf("Value mismatched by '%v'", word)
		}
	}
	prefixWords := []string{"hell", "w"}
	for _, word := range prefixWords {
		if !searcher.PrefixSearch(word) {
			t.Errorf("Fail to prefix match '%v'", word)
		}
	}

	// Fail match
	failWords := []string{"helm", "wa"}
	for _, word := range failWords {
		if searcher.PrefixSearch(word) {
			t.Errorf("Unexpected prefix match '%v'", word)
		}
	}
}

func TestSearchCN(t *testing.T) {
	builder := NewBuilder()
	words := []string{"犹豫就会败北"}
	for _, word := range words {
		builder.Add(word, word)
	}
	searcher := builder.Build()

	// Must match
	for _, word := range words {
		ok, value := searcher.Search(word)
		if !ok {
			t.Errorf("Fail to match '%v'", word)
		}
		if value != word {
			t.Errorf("Value mismatched by '%v'", word)
		}
	}
	prefixWords := []string{"犹豫"}
	for _, word := range prefixWords {
		if !searcher.PrefixSearch(word) {
			t.Errorf("Fail to prefix match '%v'", word)
		}
	}
}

func TestCover(t *testing.T) {
	builder := NewBuilder()
	words := []string{
		"abash", "abashed", "unabashed",
		"atomical", "atomically", "anatomical", "anatomically"}
	for _, word := range words {
		builder.Add(word, word)
	}
	searcher := builder.Build()
	ret := searcher.Cover("unabashed x anatomically")
	if len(ret) != len(words) {
		t.Fatal("Fail to cover enough words:", ret)
	}
	var values []string
	for _, v := range ret {
		values = append(values, v.(string))
	}
	sort.StringSlice(values).Sort()
}

func TestCoverCN(t *testing.T) {
	builder := NewBuilder()
	words := []string{"床前", "月光", "明月", "地上", "霜", "是"}
	for _, word := range words {
		builder.Add(word, word)
	}
	searcher := builder.Build()
	ret := searcher.Cover("床前明月光x，a疑是地上霜")
	if len(ret) != len(words) {
		t.Fatal("Fail to cover enough words:", ret)
	}
	var values []string
	for _, v := range ret {
		values = append(values, v.(string))
	}
	sort.StringSlice(values).Sort()
}
