package wp

import (
	"regexp"
	"strings"
)

// Word-count regexes mirror wp-admin/js/word-count.js / block_core_post_time_to_read_word_count (P2.142).
var (
	reWordCountHTML     = regexp.MustCompile(`(?i)</?[a-z][^>]*?>`)
	reWordCountComment  = regexp.MustCompile(`<!--[\s\S]*?-->`)
	reWordCountNBSP     = regexp.MustCompile(`(?i)&nbsp;|&#160;`)
	reWordCountEntity   = regexp.MustCompile(`&\S+?;`)
	reWordCountConnect  = regexp.MustCompile(`--|\x{2014}`)
	reWordCountRemove   = regexp.MustCompile(`[\x{0021}-\x{0040}\x{005B}-\x{0060}\x{007B}-\x{007E}\x{0080}-\x{00BF}\x{00D7}\x{00F7}\x{2000}-\x{2BFF}\x{2E00}-\x{2E7F}]`)
	reWordCountAstral   = regexp.MustCompile(`[\x{010000}-\x{10FFFF}]`)
	reWordCountWords    = regexp.MustCompile(`\S\s+`)
	reWordCountCharEx   = regexp.MustCompile(`\S`)
	reWordCountCharIncl = regexp.MustCompile(`[^\f\n\r\t\v\x{00AD}\x{2028}\x{2029}]`)
)

// WordCount counts words or characters in text (wp.utils.WordCounter.prototype.count, P2.142).
// type is words | characters_excluding_spaces | characters_including_spaces; invalid → words.
func WordCount(text, countType string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	switch countType {
	case WordCountTypeCharactersExcludingSpaces, WordCountTypeCharactersIncludingSpaces:
		// keep
	default:
		countType = WordCountTypeWords
	}

	text = text + "\n"
	text = reWordCountHTML.ReplaceAllString(text, "\n")
	text = reWordCountComment.ReplaceAllString(text, "")
	text = reWordCountNBSP.ReplaceAllString(text, " ")

	if countType == WordCountTypeWords {
		text = reWordCountEntity.ReplaceAllString(text, "")
		text = reWordCountConnect.ReplaceAllString(text, " ")
		text = reWordCountRemove.ReplaceAllString(text, "")
	} else {
		text = reWordCountEntity.ReplaceAllString(text, "a")
		text = reWordCountAstral.ReplaceAllString(text, "a")
	}

	var re *regexp.Regexp
	switch countType {
	case WordCountTypeCharactersExcludingSpaces:
		re = reWordCountCharEx
	case WordCountTypeCharactersIncludingSpaces:
		re = reWordCountCharIncl
	default:
		re = reWordCountWords
	}
	return len(re.FindAllString(text, -1))
}

// WPWordCount counts text using the context locale word-count type when type is empty (P2.142).
func WPWordCount(c *Context, text, countType string) int {
	if strings.TrimSpace(countType) == "" {
		countType = WPGetWordCountType(c)
	}
	return WordCount(text, countType)
}
