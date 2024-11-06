package patternParser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Pattern struct {
	regex       *regexp.Regexp
	assignments []string
}

func (p *Pattern) Parse(str string) (map[string]string, error) {
	matches := p.regex.FindStringSubmatch(str)

	if len(matches) == 0 {
		return nil, fmt.Errorf("filename doesnt match to pattern: %s", str)
	}

	// prune full match
	matches = matches[1:]

	// check if assignments are filled
	for i, e := range matches {
		if e == "" {
			return nil, fmt.Errorf("filename '%s' has empty match for assignment '%s'", str, p.assignments[i])
		}
	}

	res := make(map[string]string)
	for i, e := range p.assignments {
		res[e] = matches[i]
	}

	return res, nil
}

func Parse(patternStr string) (pattern Pattern, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("unexpected expected end of pattern")
		}
	}()
	pattern, err = parse(patternStr)
	fmt.Println("pattern:", pattern.regex.String(), "tags:", pattern.assignments)
	return pattern, err
}

type parser func(int, string) (int, bool)

func parse(patternStr string) (Pattern, error) {
	str := patternStr
	pos := 0

	// the final regexStr pattern used to parse the filenames
	regexStr := strings.Builder{}

	// list the assignments in order of appearance
	assignments := []string{}

	// add the read runes as is into the regex pattern
	runeParser := parseRune(func(s string) { regexStr.WriteString(s) })

	// replace all relevant tags with a matchgroup to extract them from the filename
	tagParser := parseTag(func(s string) {
		regexStr.WriteString(`(.*)`)
		assignments = append(assignments, s)
	})

	// loop until end of string
	for pos < len(str) {

		// check if next is a tag
		newPos, success := tagParser(pos, str)

		// else check if next is a rune
		if !success {
			newPos, success = runeParser(pos, str)
		}

		// error (can happen on a loose escape character "\" at the end)
		if !success {
			return Pattern{}, errors.New("parsing failed")
		}

		pos = newPos
	}

	regex, err := regexp.Compile(regexStr.String())
	if err != nil {
		return Pattern{}, fmt.Errorf("failed compiling regex: %w", err)
	}

	parsedPattern := Pattern{
		regex:       regex,
		assignments: assignments,
	}

	return parsedPattern, nil
}

func parseTag(action func(string)) parser {
	return func(pos int, str string) (int, bool) {
		// check terminals
		if str[pos] != '%' {
			return 0, false
		}
		pos++

		if str[pos] != '(' {
			return 0, false
		}
		pos++

		// build parser for tagname
		tag := strings.Builder{}
		runeParser := parseRune(func(s string) {
			tag.WriteString(s)
		})

		// parse until ')'
		for {

			// check if tag has ended and return
			if str[pos] == ')' {
				pos++
				action(tag.String())
				return pos, true
			}

			// else just add the runes into the tag name
			newPos, _ := runeParser(pos, str)
			pos = newPos
		}
	}
}

func parseRune(action func(string)) parser {
	return func(pos int, str string) (int, bool) {

		r := str[pos]
		pos++
		a := func(r byte) { action(regexp.QuoteMeta(string(r))) }

		// if token is a escape character just call the action with the next token
		if r == '\\' {
			r = str[pos]
			pos++

			a(r)
			return pos, true
		}

		a(r)
		return pos, true
	}

}
