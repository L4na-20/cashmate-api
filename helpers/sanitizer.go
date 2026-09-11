package helpers

import (
	"html"
	"regexp"
	"strings"
)

var scriptTagRegex = regexp.MustCompile(`(?i)<\s*/?\s*(script|style|iframe|object|embed|link|meta|form)\b[^>]*>`)
var eventAttrRegex = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
var javascriptURIRegEx = regexp.MustCompile(`(?i)\s*(href|src|action)\s*=\s*["']\s*javascript:`)

// Sanitize membersihkan string dari potensi konten berbahaya (XSS).
// Menghapus tag <script>/<style>, onClick dan sejenisnya, serta
// URL javascript: pada href/src, lalu HTML-escape karakter berbahaya.
func Sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	// Hapus seluruh tag berbahaya (dengan isinya untuk script).
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	s = reScript.ReplaceAllString(s, "")
	s = scriptTagRegex.ReplaceAllString(s, "")
	s = eventAttrRegex.ReplaceAllString(s, "")
	s = javascriptURIRegEx.ReplaceAllString(s, "")

	// Escape HTML agar karakter seperti <, >, &, ', " tidak bisa
	// dieksekusi sebagai markup.
	s = html.EscapeString(s)

	return s
}
