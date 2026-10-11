package wp

import (
	"html"
	"net/url"
	"strconv"
	"strings"
)

// PaginateLinksArgs mirrors paginate_links() arguments (P3.87 / P3.102 subset).
// PrevNext defaults to true (WP); set NoPrevNext to disable.
type PaginateLinksArgs struct {
	Base             string
	Format           string
	Total            int
	Current          int
	AriaCurrent      string
	ShowAll          bool
	NoPrevNext       bool
	PrevText         string
	NextText         string
	EndSize          int
	MidSize          int
	Type             string // plain | list | array
	AddFragment      string
	BeforePageNumber string
	AfterPageNumber  string
	// AddArgs are query args appended to each page link (P3.102; WP add_args).
	AddArgs map[string]string
	// Hooks optionally applies the paginate_links filter per URL (P3.102).
	Hooks *Hooks
}

// PaginateLinks builds archive pagination markup (paginate_links subset, P3.87 / P3.102).
// Returns "" when Total < 2.
func PaginateLinks(args PaginateLinksArgs) string {
	total := args.Total
	if total < 2 {
		return ""
	}
	current := args.Current
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	endSize := args.EndSize
	if endSize < 1 {
		endSize = 1
	}
	midSize := args.MidSize
	if midSize < 0 {
		midSize = 2
	}
	ariaCurrent := args.AriaCurrent
	if ariaCurrent == "" {
		ariaCurrent = "page"
	}
	prevText := args.PrevText
	if prevText == "" {
		prevText = "&laquo; Previous"
	}
	nextText := args.NextText
	if nextText == "" {
		nextText = "Next &raquo;"
	}
	base := args.Base
	if base == "" {
		base = "%_%"
	}
	format := args.Format
	if format == "" {
		format = "?paged=%#%"
	}
	prevNext := !args.NoPrevNext

	var pageLinks []string
	if prevNext && current > 1 {
		link := finalizePageLink(pageLinkURL(base, format, current-1), args)
		pageLinks = append(pageLinks, `<a class="prev page-numbers" href="`+html.EscapeString(link)+`">`+prevText+`</a>`)
	}

	dots := false
	for n := 1; n <= total; n++ {
		if n == current {
			pageLinks = append(pageLinks, `<span aria-current="`+html.EscapeString(ariaCurrent)+`" class="page-numbers current">`+
				args.BeforePageNumber+strconv.Itoa(n)+args.AfterPageNumber+`</span>`)
			dots = true
			continue
		}
		inWindow := args.ShowAll ||
			n <= endSize ||
			n > total-endSize ||
			(n >= current-midSize && n <= current+midSize)
		if inWindow {
			link := finalizePageLink(pageLinkURL(base, format, n), args)
			pageLinks = append(pageLinks, `<a class="page-numbers" href="`+html.EscapeString(link)+`">`+
				args.BeforePageNumber+strconv.Itoa(n)+args.AfterPageNumber+`</a>`)
			dots = true
		} else if dots {
			pageLinks = append(pageLinks, `<span class="page-numbers dots">&hellip;</span>`)
			dots = false
		}
	}

	if prevNext && current < total {
		link := finalizePageLink(pageLinkURL(base, format, current+1), args)
		pageLinks = append(pageLinks, `<a class="next page-numbers" href="`+html.EscapeString(link)+`">`+nextText+`</a>`)
	}

	switch strings.ToLower(strings.TrimSpace(args.Type)) {
	case "list":
		return "<ul class='page-numbers'>\n\t<li>" + strings.Join(pageLinks, "</li>\n\t<li>") + "</li>\n</ul>\n"
	default: // plain + array
		return strings.Join(pageLinks, "\n")
	}
}

func finalizePageLink(link string, args PaginateLinksArgs) string {
	link = addQueryArgsToLink(link, args.AddArgs)
	if args.Hooks != nil {
		if v := args.Hooks.ApplyFilters("paginate_links", link); v != nil {
			if s, ok := v.(string); ok {
				link = s
			}
		}
	}
	return link + args.AddFragment
}

func addQueryArgsToLink(link string, args map[string]string) string {
	if len(args) == 0 {
		return link
	}
	vals := url.Values{}
	for k, v := range args {
		if strings.TrimSpace(k) == "" {
			continue
		}
		vals.Set(k, v)
	}
	enc := vals.Encode()
	if enc == "" {
		return link
	}
	if strings.Contains(link, "?") {
		return link + "&" + enc
	}
	return link + "?" + enc
}

func pageLinkURL(base, format string, page int) string {
	repl := ""
	if page != 1 {
		repl = strings.ReplaceAll(format, "%#%", strconv.Itoa(page))
	}
	link := strings.ReplaceAll(base, "%_%", repl)
	link = strings.ReplaceAll(link, "%#%", strconv.Itoa(page))
	return link
}

// PaginateLinksPlain is a convenience for plain type with common archive defaults.
func PaginateLinksPlain(archiveBase string, current, total int) string {
	base := strings.TrimRight(strings.TrimSpace(archiveBase), "/") + "/%_%"
	return PaginateLinks(PaginateLinksArgs{
		Base:    base,
		Format:  "page/%#%/",
		Total:   total,
		Current: current,
		EndSize: 1,
		MidSize: 2,
		Type:    "plain",
	})
}
