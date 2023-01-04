package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"unicode"
)

func parseTemplate(path string, msgHolder *MsgHolder) error {

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(b)
	var lineNo int

	for pos := 0; pos < len(content); pos++ {
		if content[pos] == '\n' {
			lineNo++
		}

		if strings.HasPrefix(content[pos:], "{%") {
			pos += 2
			for unicode.IsSpace(rune(content[pos])) {
				pos++
			}

			if strings.HasPrefix(content[pos:], "trans ") {
				newpos, err := handleTransTag(msgHolder, path, lineNo, content[pos:])
				if err != nil {
					return err
				}
				pos += newpos
			} else if strings.HasPrefix(content[pos:], "blocktrans") {
				newpos, err := handleBlockTransTag(msgHolder, path, lineNo, content[pos:])
				if err != nil {
					return err
				}
				pos += newpos
			}
		}
	}
	return nil
}

func handleTransTag(msgHolder *MsgHolder, path string, line int, content string) (int, error) {
	var context string

	tagEndPos := strings.Index(content, "%}")
	if tagEndPos == -1 {
		return 0, fmt.Errorf("could not find end of tag in template %s:%d", path, line)
	}
	tagEndPos += 2

	pos := 5 // Skip "trans"
	for unicode.IsSpace(rune(content[pos])) {
		pos++
	}

	delimitier := content[pos : pos+1]
	// we're only interested in strings
	if delimitier != "\"" && delimitier != "'" {
		return tagEndPos, nil
	}
	pos++

	strEndPos := strings.Index(content[pos:], delimitier)
	if strEndPos == -1 {
		return 0, fmt.Errorf("could not find end of string in template %s", path)
	}
	strEndPos += pos

	if ctxidx := strings.Index(content[strEndPos:tagEndPos], "context "); ctxidx != -1 {
		ctxidx += strEndPos
		ctxidx += 8

		for unicode.IsSpace(rune(content[ctxidx])) {
			ctxidx++
		}

		delimitier := content[ctxidx : ctxidx+1]
		// we're only interested in strings
		if delimitier == "\"" || delimitier == "'" {
			ctxidx++

			ctxEndPos := strings.Index(content[ctxidx:], delimitier)
			if ctxEndPos == -1 {
				return 0, fmt.Errorf("could not find end of context string in template %s", path)
			}
			ctxEndPos += ctxidx
			context = content[ctxidx:ctxEndPos]
		}

	}

	msgHolder.Add(TranslationString{
		Position: fmt.Sprintf("%s:%d", path, line),
		Singular: content[pos:strEndPos],
		Context:  context,
	})
	return tagEndPos, nil
}

func handleBlockTransTag(msgHolder *MsgHolder, path string, line int, content string) (int, error) {
	var context, singular, plural string

	tagEndPos := strings.Index(content, "%}")
	if tagEndPos == -1 {
		return 0, fmt.Errorf("could not find end of tag in template %s:%d", path, line)
	}
	tagEndPos += 2

	pos := 10 // Skip "blocktrans"
	for unicode.IsSpace(rune(content[pos])) {
		pos++
	}

	if ctxidx := strings.Index(content[pos:tagEndPos], "context "); ctxidx != -1 {
		ctxidx += pos
		ctxidx += 8

		for unicode.IsSpace(rune(content[ctxidx])) {
			ctxidx++
		}

		delimitier := content[ctxidx : ctxidx+1]
		// we're only interested in strings
		if delimitier == "\"" || delimitier == "'" {
			ctxidx++

			ctxEndPos := strings.Index(content[ctxidx:], delimitier)
			if ctxEndPos == -1 {
				return 0, fmt.Errorf("could not find end of context string in template %s:%d", path, line)
			}
			ctxEndPos += ctxidx
			context = content[ctxidx:ctxEndPos]
		}
	}

	hasPlural := false
	for pos = tagEndPos; pos < len(content); pos++ {
		if strings.HasPrefix(content[pos:], "{% plural ") {
			singular = content[tagEndPos:pos]
			hasPlural = true

			pos += 2
			tagEndPos = strings.Index(content[pos:], "%}")
			if tagEndPos == -1 {
				return 0, fmt.Errorf("could not find end of tag in template %s:%d", path, line)
			}
			tagEndPos += pos + 2
		} else if strings.HasPrefix(content[pos:], "{% endblocktrans") {
			if hasPlural {
				plural = content[tagEndPos:pos]
			} else {
				singular = content[tagEndPos:pos]
			}
			break
		}
	}

	replacer := strings.NewReplacer("{{ ", "{{", " }}", "}}")
	for {
		n := replacer.Replace(singular)
		if n == singular {
			break
		}
		singular = n
	}

	for {
		n := replacer.Replace(plural)
		if n == plural {
			break
		}
		plural = n
	}

	msgHolder.Add(TranslationString{
		Position: fmt.Sprintf("%s:%d", path, line),
		Singular: singular,
		Plural:   plural,
		Context:  context,
	})
	return tagEndPos, nil
}
