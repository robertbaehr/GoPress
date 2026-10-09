// Locale / i18n stub (P1.9) — minimal port of wp-includes/l10n.php.
//
// Translate and TranslateNoop return the msgid unchanged until real
// gettext catalogs land in a later task.
package wp

// Locale holds the active locale and translation lookups for a Context.
type Locale struct {
	// Code is the locale identifier (e.g. "en_US"). Empty means site default.
	Code string
}

// NewLocale returns a Locale stub. code may be empty.
func NewLocale(code string) *Locale {
	return &Locale{Code: code}
}

// Translate returns the translation of msgid in domain.
// The stub returns msgid unchanged (WordPress behavior when no catalog is loaded).
func (l *Locale) Translate(msgid, domain string) string {
	_ = l
	_ = domain
	return msgid
}

// TranslateNoop marks msgid for translation without looking it up.
// Used like WordPress noop / esc_attr__ registration helpers; returns msgid.
func (l *Locale) TranslateNoop(msgid, domain string) string {
	_ = l
	_ = domain
	return msgid
}

// Translate is a package-level helper that uses l when non-nil.
func Translate(l *Locale, msgid, domain string) string {
	if l == nil {
		return msgid
	}
	return l.Translate(msgid, domain)
}

// TranslateNoop is a package-level helper that uses l when non-nil.
func TranslateNoop(l *Locale, msgid, domain string) string {
	if l == nil {
		return msgid
	}
	return l.TranslateNoop(msgid, domain)
}
