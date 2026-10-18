package pattern

import (
	"fmt"
	"strconv"
	"strings"
)

type token struct {
	value byte
	any   bool
}

type Pattern struct {
	tokens []token
	text   string
}

func MustParse(text string) Pattern {
	p, err := Parse(text)
	if err != nil {
		panic(err)
	}

	return p
}

func Parse(text string) (Pattern, error) {
	p := Pattern{text: text}

	for tok := range strings.FieldsSeq(text) {
		if tok == "??" {
			p.tokens = append(p.tokens, token{any: true})

			continue
		}

		b, err := strconv.ParseUint(tok, 16, 8)
		if err != nil {
			return Pattern{}, fmt.Errorf("pattern %q: bad token %q", text, tok)
		}

		p.tokens = append(p.tokens, token{value: byte(b)})
	}

	if len(p.tokens) == 0 {
		return Pattern{}, fmt.Errorf("pattern %q: empty", text)
	}

	return p, nil
}

func (p Pattern) Index(data []byte) int {
	for i := 0; i+len(p.tokens) <= len(data); i++ {
		if p.matchAt(data[i:]) {
			return i
		}
	}

	return -1
}

func (p Pattern) Contains(data []byte) bool {
	return p.Index(data) >= 0
}

func (p Pattern) String() string {
	return p.text
}

func (p Pattern) matchAt(data []byte) bool {
	for i, tok := range p.tokens {
		if !tok.any && data[i] != tok.value {
			return false
		}
	}

	return true
}
