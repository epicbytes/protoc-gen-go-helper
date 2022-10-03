package module

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var (
	slashSlash = []byte("//")
	moduleStr  = []byte("module")
)

var ProtoTypesMap = map[string]string{
	"TYPE_DOUBLE": "float64",
	"TYPE_FLOAT":  "float32",
	"TYPE_INT64":  "int64",
	"TYPE_UINT64": "uint64",
	"TYPE_INT32":  "int32",
	"TYPE_BOOL":   "bool",
	"TYPE_STRING": "string",
	"TYPE_BYTES":  "byte",
	"TYPE_UINT32": "uint32",
}

type MergedPickedFieldData struct {
	Name      string
	ProtoType string
	Type      string
	Repeated  bool
	Source    string
	Merged    bool
	Picked    bool
}

// ModulePath returns the module path from the gomod file text.
// If it cannot find a module path, it returns an empty string.
// It is tolerant of unrelated problems in the go.mod file.
func ModulePath(mod []byte) string {
	for len(mod) > 0 {
		line := mod
		mod = nil
		if i := bytes.IndexByte(line, '\n'); i >= 0 {
			line, mod = line[:i], line[i+1:]
		}
		if i := bytes.Index(line, slashSlash); i >= 0 {
			line = line[:i]
		}
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, moduleStr) {
			continue
		}
		line = line[len(moduleStr):]
		n := len(line)
		line = bytes.TrimSpace(line)
		if len(line) == n || len(line) == 0 {
			continue
		}

		if line[0] == '"' || line[0] == '`' {
			p, err := strconv.Unquote(string(line))
			if err != nil {
				return "" // malformed quoted string or multiline module path
			}
			return p
		}

		return string(line)
	}
	return "" // missing module path
}

const (
	INITIAL_STATE                 = iota
	EXPECT_FOLLOWING_SMALL_LETTER = iota
	IN_CONSECUTIVE_CAPITALS       = iota
	IN_WORD                       = iota
	SEEK_FOR_NEXT_WORD            = iota
)

type CaseTranslator struct {
	FirstLetter       func(rune) rune
	LetterInWord      func(rune) rune
	FirstLetterOfWord func(rune) rune
	Separator         rune
}

type processor struct {
	state        int
	buffer       *bytes.Buffer
	tr           *CaseTranslator
	bufferedRune rune
}

func NewProcessor(t *CaseTranslator) *processor {
	p := new(processor)
	p.state = INITIAL_STATE
	p.buffer = bytes.NewBuffer(nil)
	p.tr = t
	return p
}

func (p *processor) flushRuneBuffer() {
	if p.bufferedRune != 0 {
		p.writeRune(p.bufferedRune)
	}
}
func (p *processor) putCharInRuneBuffer(r rune) {
	if p.bufferedRune != 0 {
		p.charInWord(p.bufferedRune)
	}
	p.bufferedRune = r
}
func (p *processor) writeRune(r rune) {
	p.buffer.WriteRune(r)
}
func (p *processor) firstLetter(r rune) {
	p.writeRune(p.tr.FirstLetter(r))
	if unicode.IsUpper(r) {
		p.state = EXPECT_FOLLOWING_SMALL_LETTER
	} else {
		p.state = IN_WORD
	}
}
func (p *processor) foundNewWord(r rune) {
	if p.tr.Separator != 0 {
		p.writeRune(p.tr.Separator)
	}
	p.writeRune(p.tr.FirstLetterOfWord(r))

	if unicode.IsUpper(r) {
		p.state = EXPECT_FOLLOWING_SMALL_LETTER
	} else {
		p.state = IN_WORD
	}
}

func (p *processor) charInWord(r rune) {
	r = p.tr.LetterInWord(r)
	p.writeRune(r)
}
func (p *processor) firstLetterOfWord(r rune) {
	r = p.tr.FirstLetterOfWord(r)
	p.writeRune(r)
}

func (p *processor) convert(s string) string {
	p.buffer.Grow(len(s))
	for _, r := range s {
		isNumber := unicode.Is(unicode.Number, r)
		isWord := unicode.Is(unicode.Letter, r) || isNumber

		switch p.state {
		case INITIAL_STATE:
			if isWord {
				p.firstLetter(r)
			}
		case EXPECT_FOLLOWING_SMALL_LETTER:
			if isWord {
				if unicode.IsUpper(r) {
					p.putCharInRuneBuffer(r)
					p.state = IN_CONSECUTIVE_CAPITALS
				} else {
					p.flushRuneBuffer()
					p.charInWord(r)
					p.state = IN_WORD
				}
			} else {
				p.putCharInRuneBuffer(0)
				p.state = SEEK_FOR_NEXT_WORD
			}
		case IN_CONSECUTIVE_CAPITALS:
			if isWord {
				if unicode.IsUpper(r) || isNumber {
					p.putCharInRuneBuffer(r)
				} else {
					p.foundNewWord(p.bufferedRune)
					p.bufferedRune = 0
					p.charInWord(r)
					p.state = IN_WORD
				}
			} else {
				p.putCharInRuneBuffer(0)
				p.state = SEEK_FOR_NEXT_WORD
			}
		case IN_WORD:
			if isWord {
				if unicode.IsUpper(r) {
					p.foundNewWord(r)
				} else {
					p.charInWord(r)
				}
			} else {
				p.state = SEEK_FOR_NEXT_WORD
			}
		case SEEK_FOR_NEXT_WORD:
			if isWord {
				p.foundNewWord(r)
			}
		}
	}
	if p.bufferedRune != 0 {
		p.charInWord(p.bufferedRune)
	}
	return p.buffer.String()
}

func (p *processor) Convert(s string) string {
	return p.convert(s)
}

func NewLowerProcessor(separator rune) *processor {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToLower,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToLower,
		Separator:         separator,
	})
}

func Camel(s string) string {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToLower,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToUpper,
	}).Convert(s)
}

func Pascal(s string) string {
	return NewProcessor(&CaseTranslator{
		FirstLetter:       unicode.ToUpper,
		LetterInWord:      unicode.ToLower,
		FirstLetterOfWord: unicode.ToUpper,
	}).Convert(s)
}

func Snake(s string) string {
	return NewLowerProcessor('_').Convert(s)
}

func GetErrWithLinesNumber(err error) error {
	if err == nil {
		return nil
	}
	sb := bytes.NewBufferString("")

	reader := strings.NewReader(err.Error())
	sc := bufio.NewScanner(reader)
	var num int
	for sc.Scan() {
		if num > 0 {
			sb.WriteString(strconv.Itoa(num))
			sb.WriteString(". ")
		}
		sb.WriteString(sc.Text())
		sb.WriteString("\n")
		num++
	}

	return errors.New(fmt.Sprintf("%v\n", sb.String()))
}
