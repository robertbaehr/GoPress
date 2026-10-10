package server

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveRSDXMLStub serves /rsd.xml (P3.32 / P3.100).
// Core serves RSD from xmlrpc.php?rsd; GoPress keeps that path 403 (XML-RPC
// gone) and answers legacy /rsd.xml probes with a discovery document that
// advertises the REST API (WP-API) instead of XML-RPC endpoints.
func serveRSDXMLStub(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	_ = r
	home := rsdHomeURL(c)
	apiRoot := strings.TrimRight(home, "/") + "/wp-json/"
	// P3.100: rest_output_link_header on RSD responses.
	if home != "" {
		w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="https://api.w.org/"`, apiRoot))
	}
	w.Header().Set("Content-Type", "text/xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	body := buildRSDXML(home, apiRoot)
	_, _ = w.Write([]byte(body))
}

func rsdHomeURL(c *wp.Context) string {
	if c == nil || c.Options == nil {
		return ""
	}
	siteURL, _ := c.Options.GetOption("siteurl", "").(string)
	home, _ := c.Options.GetOption("home", siteURL).(string)
	home = strings.TrimRight(strings.TrimSpace(home), "/")
	if home == "" {
		home = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	}
	return home
}

// buildRSDXML builds an RSD document with WP-API discovery (P3.100).
// XML-RPC api entries are omitted — XML-RPC remains out of scope.
func buildRSDXML(home, apiRoot string) string {
	if home == "" {
		home = "/"
	}
	if apiRoot == "" {
		apiRoot = strings.TrimRight(home, "/") + "/wp-json/"
		if home == "/" {
			apiRoot = "/wp-json/"
		}
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rsd version="1.0" xmlns="http://archipelago.phrasewise.com/rsd">` + "\n")
	b.WriteString("\t<service>\n")
	b.WriteString("\t\t<engineName>GoPress</engineName>\n")
	b.WriteString("\t\t<engineLink>https://github.com/robertbaehr/GoPress</engineLink>\n")
	b.WriteString("\t\t<homePageLink>")
	b.WriteString(html.EscapeString(home))
	b.WriteString("</homePageLink>\n")
	b.WriteString("\t\t<apis>\n")
	b.WriteString("\t\t\t<api name=\"WP-API\" blogID=\"1\" preferred=\"true\" apiLink=\"")
	b.WriteString(html.EscapeString(apiRoot))
	b.WriteString("\" />\n")
	b.WriteString("\t\t</apis>\n")
	b.WriteString("\t</service>\n")
	b.WriteString("</rsd>\n")
	return b.String()
}
