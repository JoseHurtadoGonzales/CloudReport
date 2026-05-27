package render

import (
	"strings"
	"testing"
)

func TestParseAssetParam(t *testing.T) {
	tests := []struct {
		name         string
		in           string
		wantName     string
		wantEncoding string
		wantErr      bool
	}{
		{"name only defaults utf8", "logo.png", "logo.png", "utf8", false},
		{"name with spaces trimmed", "  styles.css  ", "styles.css", "utf8", false},
		{"explicit base64", "img.png @encoding=base64", "img.png", "base64", false},
		{"explicit utf8", "x.css @encoding=utf8", "x.css", "utf8", false},
		{"explicit string", "x.css @encoding=string", "x.css", "string", false},
		{"explicit dataURI", "x.png @encoding=dataURI", "x.png", "dataURI", false},
		{"explicit link", "x.png @encoding=link", "x.png", "link", false},
		{"whitespace around params", "x.png @encoding = base64 ", "x.png", "base64", false},
		{"bad param no equals", "x.png @encoding", "x.png", "utf8", true},
		{"unsupported param key", "x.png @charset=base64", "x.png", "utf8", true},
		{"unsupported encoding", "x.png @encoding=hex", "x.png", "hex", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, encoding, err := parseAssetParam(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseAssetParam(%q) err=%v, wantErr=%v", tt.in, err, tt.wantErr)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			// On bad-param errors the encoding may be the default or the bad
			// value; only assert encoding when no error was expected.
			if !tt.wantErr && encoding != tt.wantEncoding {
				t.Errorf("encoding = %q, want %q", encoding, tt.wantEncoding)
			}
		})
	}
}

func TestEncodeAsset(t *testing.T) {
	body := []byte("hello")
	tests := []struct {
		name     string
		body     []byte
		mime     string
		asset    string
		encoding string
		want     string
	}{
		{"utf8 returns raw", body, "text/plain", "a.txt", "utf8", "hello"},
		{"string returns raw", body, "text/plain", "a.txt", "string", "hello"},
		{"base64", body, "text/plain", "a.txt", "base64", "aGVsbG8="},
		{"dataURI with text mime adds charset", body, "text/css", "a.css", "dataURI", "data:text/css; charset=UTF-8;base64,aGVsbG8="},
		{"dataURI binary mime no charset", body, "image/png", "a.png", "dataURI", "data:image/png;base64,aGVsbG8="},
		{"dataURI guesses mime from name", body, "", "a.png", "dataURI", "data:image/png;base64,aGVsbG8="},
		{"dataURI fallback octet-stream", body, "", "unknownfile", "dataURI", "data:application/octet-stream;base64,aGVsbG8="},
		{"link uses by-name URL", body, "", "logo.png", "link", "/assets/by-name/logo.png"},
		{"unknown encoding falls back to string", body, "", "a.txt", "weird", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeAsset(tt.body, tt.mime, tt.asset, tt.encoding)
			if got != tt.want {
				t.Errorf("encodeAsset(...) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGuessMimeFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"styles.css", "text/css"},
		{"app.js", "application/javascript"},
		{"data.json", "application/json"},
		{"page.html", "text/html"},
		{"page.htm", "text/html"},
		{"icon.svg", "image/svg+xml"},
		{"img.png", "image/png"},
		{"img.jpg", "image/jpeg"},
		{"img.jpeg", "image/jpeg"},
		{"img.gif", "image/gif"},
		{"img.webp", "image/webp"},
		{"f.woff", "font/woff"},
		{"f.woff2", "font/woff2"},
		{"f.ttf", "font/ttf"},
		{"f.otf", "font/otf"},
		{"upper case STYLES.CSS", "text/css"},
		{"no extension", ""},
		{"unknown.xyz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessMimeFromName(tt.name)
			if got != tt.want {
				t.Errorf("guessMimeFromName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsHTMLOutput(t *testing.T) {
	tests := []struct {
		name     string
		recipe   string
		rendered string
		want     bool
	}{
		{"html recipe", "html", "anything", true},
		{"weasyprint recipe", "weasyprint", "x", true},
		{"chrome-pdf recipe", "chrome-pdf", "x", true},
		{"html-to-xlsx recipe", "html-to-xlsx", "x", true},
		{"xlsx not html", "xlsx", "{\"a\":1}", false},
		{"text not html", "text", "plain", false},
		{"sniff full html doc", "text", "<!DOCTYPE html><html><body>x</body></html>", true},
		{"sniff html tag", "text", "  <html>x</html>", true},
		{"sniff body tag", "text", "<body>x", true},
		{"sniff fragment not full", "text", "<div>x</div>", false},
		{"plain text no angle", "text", "hello world", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHTMLOutput(tt.recipe, tt.rendered)
			if got != tt.want {
				t.Errorf("isHTMLOutput(%q, %q) = %v, want %v", tt.recipe, tt.rendered, got, tt.want)
			}
		})
	}
}

func TestInjectStyles(t *testing.T) {
	t.Run("injects style before closing head", func(t *testing.T) {
		html := "<html><head><title>t</title></head><body>hi</body></html>"
		out := injectStyles(html, "p{color:red}", "Letter", "portrait", "2cm")
		if !strings.Contains(out, "<style>") {
			t.Fatalf("expected <style> in output: %s", out)
		}
		// style must land inside head, i.e. before </head>
		styleIdx := strings.Index(out, "<style>")
		headEndIdx := strings.Index(out, "</head>")
		if styleIdx == -1 || headEndIdx == -1 || styleIdx > headEndIdx {
			t.Errorf("style block not inside head: %s", out)
		}
		if !strings.Contains(out, "@page { size: Letter; margin: 2cm; }") {
			t.Errorf("expected page rule with given size/margin: %s", out)
		}
		if !strings.Contains(out, "p{color:red}") {
			t.Errorf("expected css inlined: %s", out)
		}
	})

	t.Run("wraps content when no head", func(t *testing.T) {
		out := injectStyles("<p>just a fragment</p>", "", "", "", "")
		if !strings.HasPrefix(out, "<!DOCTYPE html>") {
			t.Errorf("expected wrapped html envelope: %s", out)
		}
		if !strings.Contains(out, "<head>") || !strings.Contains(out, "<style>") {
			t.Errorf("expected head with style: %s", out)
		}
		if !strings.Contains(out, "<body><p>just a fragment</p></body>") {
			t.Errorf("expected original content in body: %s", out)
		}
		// Defaults: A4 + 1cm
		if !strings.Contains(out, "@page { size: A4; margin: 1cm; }") {
			t.Errorf("expected default page rule: %s", out)
		}
	})

	t.Run("landscape orientation", func(t *testing.T) {
		out := injectStyles("<head></head>", "", "A4", "landscape", "1cm")
		if !strings.Contains(out, "size: A4 landscape;") {
			t.Errorf("expected landscape size: %s", out)
		}
	})
}

func TestComponentRegex(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		match     bool
		wantInner string
	}{
		{"simple partial", "{{> header}}", true, "header"},
		{"no space", "{{>header}}", true, "header"},
		{"extra spaces", "{{>   foo_bar-1   }}", true, "foo_bar-1"},
		{"not a partial", "{{ name }}", false, ""},
		{"plain text", "hello", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := componentRegex.MatchString(tt.in); got != tt.match {
				t.Fatalf("MatchString(%q) = %v, want %v", tt.in, got, tt.match)
			}
			if tt.match {
				sub := componentRegex.FindStringSubmatch(tt.in)
				if len(sub) < 2 || sub[1] != tt.wantInner {
					t.Errorf("inner capture = %v, want %q", sub, tt.wantInner)
				}
			}
		})
	}
}

func TestAssetRegex(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		match     bool
		wantInner string
	}{
		{"basic asset", "{#asset logo.png}", true, "logo.png"},
		{"asset with encoding", "{#asset img.png @encoding=base64}", true, "img.png @encoding=base64"},
		{"not an asset", "{{ asset }}", false, ""},
		{"plain text", "no tokens here", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := assetRegex.MatchString(tt.in); got != tt.match {
				t.Fatalf("MatchString(%q) = %v, want %v", tt.in, got, tt.match)
			}
			if tt.match {
				sub := assetRegex.FindStringSubmatch(tt.in)
				if len(sub) < 2 || sub[1] != tt.wantInner {
					t.Errorf("inner capture = %v, want %q", sub, tt.wantInner)
				}
			}
		})
	}
}

func TestStripHTMLEnvelope(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full doc", "<html><body>content</body></html>", "content"},
		{"body with attrs", "<html><body class=\"x\">inner</body></html>", "inner"},
		{"no body returns as-is", "<div>fragment</div>", "<div>fragment</div>"},
		{"uppercase tags", "<HTML><BODY>up</BODY></HTML>", "up"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripHTMLEnvelope(tt.in); got != tt.want {
				t.Errorf("stripHTMLEnvelope(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDefaultStr(t *testing.T) {
	if got := defaultStr("", "def"); got != "def" {
		t.Errorf("defaultStr empty = %q, want def", got)
	}
	if got := defaultStr("val", "def"); got != "val" {
		t.Errorf("defaultStr non-empty = %q, want val", got)
	}
}

func TestDefaultJSON(t *testing.T) {
	if got := string(defaultJSON(nil, "[]")); got != "[]" {
		t.Errorf("defaultJSON nil = %q, want []", got)
	}
	if got := string(defaultJSON([]byte(`{"a":1}`), "[]")); got != `{"a":1}` {
		t.Errorf("defaultJSON value = %q", got)
	}
}
