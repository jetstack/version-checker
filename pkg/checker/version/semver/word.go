package semver

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

type word interface {
	isInt() bool
	addRune(s rune)
	lessThan(word) bool
	equal(word) bool
}

type stringWord struct {
	ss string
}

type intWord struct {
	ii string
}

func newWord(s rune) word {
	if unicode.IsNumber(s) {
		return newIntWord(s)
	}
	return newStringWord(s)
}

func newStringWord(s rune) word {
	return &stringWord{string(s)}
}

func (s *stringWord) addRune(r rune) {
	s.ss += string(r)
}

func (s *stringWord) isInt() bool {
	return false
}

func (s *stringWord) lessThan(w word) bool {
	ws, ok := w.(*stringWord)
	if !ok {
		return true
	}

	return s.ss < ws.ss
}

func (s *stringWord) equal(w word) bool {
	ws, ok := w.(*stringWord)
	if !ok {
		return true
	}

	return s.ss == ws.ss
}

func newIntWord(s rune) word {
	return &intWord{string(s)}
}

func (i *intWord) addRune(r rune) {
	i.ii += string(r)
}

func (i *intWord) isInt() bool {
	return true
}

func (i *intWord) lessThan(w word) bool {
	wi, ok := w.(*intWord)
	if !ok {
		return false
	}

	iii, _ := strconv.ParseInt(i.ii, 10, 64)
	wiii, _ := strconv.ParseInt(wi.ii, 10, 64)
	return iii < wiii
}

func (i *intWord) equal(w word) bool {
	wi, ok := w.(*intWord)
	if !ok {
		return true
	}

	return i.ii == wi.ii
}

func parseStringToWords(ss string) []word {
	if len(ss) == 0 {
		return nil
	}

	firstRune, _ := utf8.DecodeRuneInString(ss)
	words := []word{newWord(firstRune)}

	if len(ss) == 1 {
		return words
	}

	for _, s := range ss[1:] {
		if words[len(words)-1].isInt() == unicode.IsNumber(s) {
			words[len(words)-1].addRune(s)
		} else {
			words = append(words, newWord(s))
		}
	}

	return words
}
