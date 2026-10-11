package wp

// Word count type constants (WP_Locale::get_word_count_type, P2.141).
const (
	WordCountTypeWords                     = "words"
	WordCountTypeCharactersExcludingSpaces = "characters_excluding_spaces"
	WordCountTypeCharactersIncludingSpaces = "characters_including_spaces"
)

// GetWordCountType mirrors WP_Locale::get_word_count_type().
// Valid values are words, characters_excluding_spaces, characters_including_spaces.
// Invalid or empty WordCountType falls back to words.
func (l *Locale) GetWordCountType() string {
	if l == nil {
		return WordCountTypeWords
	}
	switch l.WordCountType {
	case WordCountTypeCharactersExcludingSpaces, WordCountTypeCharactersIncludingSpaces:
		return l.WordCountType
	default:
		return WordCountTypeWords
	}
}

// WPGetWordCountType mirrors wp_get_word_count_type() (P2.141).
// Returns Locale.GetWordCountType when Locale is set; otherwise words.
func WPGetWordCountType(c *Context) string {
	if c == nil || c.Locale == nil {
		return WordCountTypeWords
	}
	return c.Locale.GetWordCountType()
}
