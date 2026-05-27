package odata

import (
	"reflect"
	"testing"
)

func TestToJSReportOne(t *testing.T) {
	in := map[string]any{
		"created_at":       "2020",
		"updated_at":       "2021",
		"template_shortid": "abc",
		"is_admin":         true,
		"read_perms":       []string{"g1"},
		"name":             "untouched", // unmapped passes through
	}
	got := toJSReportOne(in)
	want := map[string]any{
		"creationDate":     "2020",
		"modificationDate": "2021",
		"templateShortid":  "abc",
		"isAdmin":          true,
		"readPermissions":  []string{"g1"},
		"name":             "untouched",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toJSReportOne mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestFromJSReport(t *testing.T) {
	in := map[string]any{
		"creationDate":    "2020",
		"templateShortid": "abc",
		"isAdmin":         true,
		"folder":          "f1",
		"customField":     "x", // unmapped passes through
	}
	got := fromJSReport(in)
	want := map[string]any{
		"created_at":       "2020",
		"template_shortid": "abc",
		"is_admin":         true,
		"folder_id":        "f1",
		"customField":      "x",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fromJSReport mismatch\n got: %v\nwant: %v", got, want)
	}
}

func TestAliasRoundTrip(t *testing.T) {
	// For every alias, camel -> snake -> camel should be stable.
	for camel := range jsreportAliases {
		t.Run(camel, func(t *testing.T) {
			snake := fromJSReport(map[string]any{camel: "v"})
			back := toJSReportOne(snake)
			if _, ok := back[camel]; !ok {
				t.Errorf("round-trip lost key %q (snake=%v back=%v)", camel, snake, back)
			}
		})
	}
}

func TestToJSReportMany(t *testing.T) {
	items := []map[string]any{
		{"created_at": "a"},
		{"is_admin": false},
	}
	got := toJSReport(items)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0]["creationDate"] != "a" {
		t.Errorf("got[0] = %v", got[0])
	}
	if got[1]["isAdmin"] != false {
		t.Errorf("got[1] = %v", got[1])
	}
}

func TestIntersects(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"overlap", []string{"x", "y"}, []string{"y", "z"}, true},
		{"no overlap", []string{"a"}, []string{"b"}, false},
		{"empty a", nil, []string{"b"}, false},
		{"empty b", []string{"a"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intersects(tt.a, tt.b); got != tt.want {
				t.Errorf("intersects(%v,%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAllowed(t *testing.T) {
	uid := "user1"
	principals := []string{"user1", "groupA"}
	tests := []struct {
		name string
		row  map[string]any
		want bool
	}{
		{"owner matches", map[string]any{"owner_id": "user1"}, true},
		{"owner mismatch, no perms denies", map[string]any{"owner_id": "other"}, false},
		{"perms include user ([]string)", map[string]any{"owner_id": "other", "read_perms": []string{"user1"}}, true},
		{"perms include group ([]any)", map[string]any{"owner_id": "other", "read_perms": []any{"groupA"}}, true},
		{"perms exclude user", map[string]any{"owner_id": "other", "read_perms": []string{"someoneElse"}}, false},
		{"empty perms open to authenticated", map[string]any{"owner_id": "other", "read_perms": []string{}}, true},
		{"no owner no perms is public", map[string]any{"name": "x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowed(tt.row, uid, principals, "read_perms"); got != tt.want {
				t.Errorf("allowed(%v) = %v, want %v", tt.row, got, tt.want)
			}
		})
	}
}

func TestScopeByPermissions(t *testing.T) {
	uid := "u"
	principals := []string{"u"}
	items := []map[string]any{
		{"owner_id": "u", "name": "mine"},
		{"owner_id": "other", "name": "theirs"},
		{"name": "public"},
	}
	got := scopeByPermissions(items, uid, principals, "read_perms")
	if len(got) != 2 {
		t.Fatalf("expected 2 visible rows, got %d: %v", len(got), got)
	}
}

func TestParseLiteral(t *testing.T) {
	tests := []struct {
		in   string
		want any
	}{
		{"'hello'", "hello"},
		{"'it''s'", "it's"},
		{"true", true},
		{"false", false},
		{"null", nil},
		{"42", int64(42)},
		{"3.14", float64(3.14)},
		{"bareword", "bareword"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseLiteral(tt.in)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLiteral(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{"simple", "name eq 'x'", []string{"name", "eq", "'x'"}, false},
		{"quoted with spaces", "name eq 'a b c'", []string{"name", "eq", "'a b c'"}, false},
		{"escaped quote", "name eq 'it''s'", []string{"name", "eq", "'it''s'"}, false},
		{"unterminated", "name eq 'x", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenize(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveCol(t *testing.T) {
	allowed := map[string]bool{"created_at": true, "name": true}
	alias := map[string]string{"creationDate": "created_at"}
	tests := []struct {
		in   string
		want string
	}{
		{"creationDate", "created_at"},
		{"name", "name"},
		{"created_at", "created_at"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := resolveCol(tt.in, allowed, alias); got != tt.want {
				t.Errorf("resolveCol(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseFilter(t *testing.T) {
	allowed := map[string]bool{"name": true, "is_admin": true}
	alias := map[string]string{"isAdmin": "is_admin"}

	t.Run("single clause", func(t *testing.T) {
		sql, args, err := parseFilter("name eq 'bob'", allowed, alias)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "name = $1" {
			t.Errorf("sql = %q", sql)
		}
		if len(args) != 1 || args[0] != "bob" {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("alias and conjunction", func(t *testing.T) {
		sql, args, err := parseFilter("isAdmin eq true and name ne 'x'", allowed, alias)
		if err != nil {
			t.Fatal(err)
		}
		if sql != "is_admin = $1 AND name <> $2" {
			t.Errorf("sql = %q", sql)
		}
		if len(args) != 2 || args[0] != true || args[1] != "x" {
			t.Errorf("args = %v", args)
		}
	})

	t.Run("unknown field errors", func(t *testing.T) {
		if _, _, err := parseFilter("bogus eq 'x'", allowed, alias); err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("unsupported op errors", func(t *testing.T) {
		if _, _, err := parseFilter("name like 'x'", allowed, alias); err == nil {
			t.Error("expected error for unsupported op")
		}
	})
}

func TestParse(t *testing.T) {
	cols := []string{"name", "created_at"}
	alias := map[string]string{"creationDate": "created_at"}

	t.Run("defaults top 100", func(t *testing.T) {
		q, err := Parse(map[string]string{}, cols, alias)
		if err != nil {
			t.Fatal(err)
		}
		if q.Top != 100 {
			t.Errorf("Top default = %d, want 100", q.Top)
		}
	})

	t.Run("top/skip/count", func(t *testing.T) {
		q, err := Parse(map[string]string{"$top": "5", "$skip": "10", "$count": "true"}, cols, alias)
		if err != nil {
			t.Fatal(err)
		}
		if q.Top != 5 || q.Skip != 10 || !q.Count {
			t.Errorf("got Top=%d Skip=%d Count=%v", q.Top, q.Skip, q.Count)
		}
	})

	t.Run("orderby with alias and desc", func(t *testing.T) {
		q, err := Parse(map[string]string{"$orderby": "creationDate desc, name asc"}, cols, alias)
		if err != nil {
			t.Fatal(err)
		}
		if q.OrderBy != "created_at DESC, name ASC" {
			t.Errorf("OrderBy = %q", q.OrderBy)
		}
	})

	t.Run("select resolves aliases", func(t *testing.T) {
		q, err := Parse(map[string]string{"$select": "creationDate, name, bogus"}, cols, alias)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"created_at", "name"}
		if !reflect.DeepEqual(q.Select, want) {
			t.Errorf("Select = %v, want %v", q.Select, want)
		}
	})

	t.Run("inlinecount allpages", func(t *testing.T) {
		q, err := Parse(map[string]string{"$inlinecount": "allpages"}, cols, alias)
		if err != nil || !q.InlineCount {
			t.Errorf("InlineCount = %v err=%v", q.InlineCount, err)
		}
	})

	t.Run("invalid top errors", func(t *testing.T) {
		if _, err := Parse(map[string]string{"$top": "-1"}, cols, alias); err == nil {
			t.Error("expected error for negative top")
		}
	})
}
