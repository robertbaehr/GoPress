package admin

import (
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// serveImport is the Tools → Import screen with a WordPress WXR upload stub (P7.54 / P7.91).
func (h *Handler) serveImport(w http.ResponseWriter, r *http.Request, c *wp.Context) {
	// P7.91: import capability gate (mirrors import.php wp_die).
	if c == nil || c.CurrentUser == nil || !c.UserCan(c.CurrentUser, "import") {
		http.Error(w, "Sorry, you are not allowed to import content into this site.\n", http.StatusForbidden)
		return
	}
	note := ""
	noteClass := "notice notice-info"
	if r.Method == http.MethodPost {
		note, noteClass = h.handleWXRUpload(r)
	}
	blogname := optionString(c, "blogname", "GoPress")
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<title>Import &lsaquo; ")
	b.WriteString(html.EscapeString(blogname))
	b.WriteString(" &#8212; WordPress</title>\n</head>\n")
	// P7.91: import-php body class like get_body_class for this screen.
	b.WriteString("<body class=\"wp-admin wp-core-ui import-php\">\n")
	b.WriteString(adminNoticesHTML(c))
	b.WriteString("<div class=\"wrap\">\n")
	b.WriteString("<h1>Import</h1>\n")
	if inv := strings.TrimSpace(r.URL.Query().Get("invalid")); inv != "" {
		b.WriteString("<div class=\"notice notice-error\"><p><strong>Error:</strong> The ")
		b.WriteString("<strong>")
		b.WriteString(html.EscapeString(inv))
		b.WriteString("</strong> importer is invalid or is not installed.</p></div>\n")
	}
	b.WriteString("<p>If you have posts or comments in another system, WordPress can import those into this site. To get started, choose a system to import from below:</p>\n")
	if note != "" {
		b.WriteString("<div class=\"")
		b.WriteString(html.EscapeString(noteClass))
		b.WriteString("\"><p>")
		b.WriteString(note) // trusted markup from handleWXRUpload
		b.WriteString("</p></div>\n")
	}
	b.WriteString("<table class=\"widefat importers striped\">\n")
	b.WriteString("<thead><tr><th scope=\"col\">Importer</th><th scope=\"col\">Description</th></tr></thead>\n")
	b.WriteString("<tbody>\n")
	b.WriteString("<tr class=\"importer-item\">\n")
	b.WriteString("<td class=\"import-system row-title\"><a href=\"#wordpress-importer\" aria-label=\"Run WordPress\"><strong>WordPress</strong></a> ")
	b.WriteString("<span class=\"importer-action\"><a href=\"#wordpress-importer\">Run Importer</a></span></td>\n")
	b.WriteString("<td class=\"desc\"><span class=\"importer-desc\">Import posts, pages, comments, custom fields, categories, and tags from a WordPress export file.</span></td>\n")
	b.WriteString("</tr>\n")
	b.WriteString("</tbody>\n</table>\n")
	b.WriteString("<div class=\"wp-upload-form\" id=\"wordpress-importer\">\n")
	b.WriteString("<h2>Import WordPress</h2>\n")
	b.WriteString("<p class=\"description\">Howdy! Upload your WordPress eXtended RSS (WXR) file and we&#8217;ll import the posts, pages, comments, custom fields, categories, and tags into this site.</p>\n")
	b.WriteString("<p class=\"description\">Choose a WXR (.xml) file to upload, then click Upload file and import. GoPress validates the document and reports item counts; full import lands later.</p>\n")
	b.WriteString("<form enctype=\"multipart/form-data\" method=\"post\" action=\"/wp-admin/import.php\">\n")
	b.WriteString("<input type=\"hidden\" name=\"import\" value=\"wordpress\">\n")
	b.WriteString("<p><label for=\"wxr-upload\">Choose a WXR file from your computer:</label><br>\n")
	b.WriteString("<input type=\"file\" id=\"wxr-upload\" name=\"import_file\" accept=\".xml,text/xml,application/xml\"></p>\n")
	b.WriteString("<p class=\"submit\"><input type=\"submit\" name=\"submit\" id=\"submit\" class=\"button button-primary\" value=\"Upload file and import\"></p>\n")
	b.WriteString("</form>\n</div>\n")
	b.WriteString("</div>\n</body>\n</html>\n")
	writeHTML(w, http.StatusOK, b.String())
}

func (h *Handler) handleWXRUpload(r *http.Request) (note, noteClass string) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return "Upload failed: could not parse multipart form.", "notice notice-error"
	}
	file, hdr, err := r.FormFile("import_file")
	if err != nil {
		return "Please choose a WXR (.xml) file to upload.", "notice notice-error"
	}
	defer file.Close()
	name := ""
	if hdr != nil {
		name = hdr.Filename
	}
	data, err := io.ReadAll(io.LimitReader(file, 16<<20))
	if err != nil {
		return "Upload failed: could not read file.", "notice notice-error"
	}
	result, err := parseMinimalWXR(data)
	if err != nil {
		return "Import rejected: " + html.EscapeString(err.Error()) +
			" (" + html.EscapeString(name) + ")", "notice notice-error"
	}
	msg := "Parsed WXR " + html.EscapeString(name) +
		": version " + html.EscapeString(result.Version) +
		", " + strconv.Itoa(result.Items) + " item(s)"
	if strings.TrimSpace(result.Title) != "" {
		msg += ", site “" + html.EscapeString(strings.TrimSpace(result.Title)) + "”"
	}
	msg += ". Full import is not enabled in this stub."
	return msg, "notice notice-success"
}

type wxrParseResult struct {
	Version string
	Items   int
	Title   string
}

// parseMinimalWXR validates a WordPress eXtended RSS document and counts items.
func parseMinimalWXR(data []byte) (*wxrParseResult, error) {
	raw := strings.TrimSpace(string(data))
	if raw == "" {
		return nil, errWXR("empty file")
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "<rss") && !strings.Contains(lower, "wordpress.org/export") {
		return nil, errWXR("not a WXR/RSS export document")
	}
	if !strings.Contains(lower, "wxr_version") && !strings.Contains(lower, "wordpress.org/export") {
		return nil, errWXR("missing WXR markers (wp:wxr_version)")
	}

	type wxrChannel struct {
		Title      string `xml:"title"`
		WXRVersion string `xml:"http://wordpress.org/export/1.2/ wxr_version"`
		Items      []struct {
			Title string `xml:"title"`
		} `xml:"item"`
	}
	type wxrRSS struct {
		Channel wxrChannel `xml:"channel"`
	}
	var doc wxrRSS
	dec := xml.NewDecoder(strings.NewReader(raw))
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		// Fall back to lightweight scanning when XML namespaces confuse the decoder.
		ver := extractBetween(raw, "<wp:wxr_version>", "</wp:wxr_version>")
		if ver == "" {
			return nil, errWXR("invalid XML: " + err.Error())
		}
		items := strings.Count(lower, "<item>")
		return &wxrParseResult{Version: strings.TrimSpace(ver), Items: items, Title: extractBetween(raw, "<title>", "</title>")}, nil
	}
	ver := strings.TrimSpace(doc.Channel.WXRVersion)
	if ver == "" {
		ver = extractBetween(raw, "<wp:wxr_version>", "</wp:wxr_version>")
	}
	if ver == "" {
		return nil, errWXR("missing wp:wxr_version")
	}
	return &wxrParseResult{
		Version: ver,
		Items:   len(doc.Channel.Items),
		Title:   strings.TrimSpace(doc.Channel.Title),
	}, nil
}

type wxrError string

func (e wxrError) Error() string { return string(e) }

func errWXR(msg string) error { return wxrError(msg) }

func extractBetween(s, open, close string) string {
	i := strings.Index(s, open)
	if i < 0 {
		return ""
	}
	i += len(open)
	j := strings.Index(s[i:], close)
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}
