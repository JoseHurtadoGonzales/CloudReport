// Package odata implements a minimal OData v4 query parser sufficient for
// jsreport-style clients. Supported options:
//
//   $filter, $select, $top, $skip, $orderby, $count, $inlinecount
//
// $filter grammar: <field> <op> <literal> [ (and|or) <field> <op> <literal> ]*
// where op ∈ {eq, ne, gt, ge, lt, le} and literals are 'strings', numbers,
// true/false, or null. No parentheses, no functions.
package odata

import (
	"fmt"
	"strconv"
	"strings"
)

type Query struct {
	Filter      string         // SQL fragment (already $-numbered)
	Args        []any
	Select      []string
	OrderBy     string
	Skip        int
	Top         int
	Count       bool
	InlineCount bool
}

// Parse takes the raw URL query (as map) and a column whitelist for the entity.
func Parse(values map[string]string, allowedColumns []string, columnAlias map[string]string) (*Query, error) {
	q := &Query{Top: 100}
	allowed := map[string]bool{}
	for _, c := range allowedColumns {
		allowed[c] = true
	}
	if columnAlias == nil {
		columnAlias = map[string]string{}
	}

	if s := values["$top"]; s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("$top: %v", err)
		}
		q.Top = n
	}
	if s := values["$skip"]; s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("$skip: %v", err)
		}
		q.Skip = n
	}
	if s := values["$count"]; s == "true" {
		q.Count = true
	}
	if s := values["$inlinecount"]; s == "allpages" {
		q.InlineCount = true
	}
	if sel := values["$select"]; sel != "" {
		parts := strings.Split(sel, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			col := resolveCol(p, allowed, columnAlias)
			if col != "" {
				q.Select = append(q.Select, col)
			}
		}
	}
	if ob := values["$orderby"]; ob != "" {
		segs := strings.Split(ob, ",")
		var parts []string
		for _, s := range segs {
			s = strings.TrimSpace(s)
			dir := "ASC"
			if strings.HasSuffix(strings.ToLower(s), " desc") {
				dir = "DESC"
				s = strings.TrimSpace(s[:len(s)-5])
			} else if strings.HasSuffix(strings.ToLower(s), " asc") {
				s = strings.TrimSpace(s[:len(s)-4])
			}
			col := resolveCol(s, allowed, columnAlias)
			if col == "" {
				continue
			}
			parts = append(parts, col+" "+dir)
		}
		q.OrderBy = strings.Join(parts, ", ")
	}
	if filter := values["$filter"]; filter != "" {
		sql, args, err := parseFilter(filter, allowed, columnAlias)
		if err != nil {
			return nil, err
		}
		q.Filter = sql
		q.Args = args
	}
	return q, nil
}

func resolveCol(name string, allowed map[string]bool, alias map[string]string) string {
	if a, ok := alias[name]; ok {
		name = a
	}
	if allowed[name] {
		return name
	}
	return ""
}

func parseFilter(f string, allowed map[string]bool, alias map[string]string) (string, []any, error) {
	// Tokenize by whitespace, respecting single-quoted strings.
	tokens, err := tokenize(f)
	if err != nil {
		return "", nil, err
	}
	var parts []string
	var args []any
	i := 0
	argIdx := 1
	for i < len(tokens) {
		if i+3 > len(tokens) {
			return "", nil, fmt.Errorf("filter syntax: incomplete clause at %d", i)
		}
		field := tokens[i]
		op := strings.ToLower(tokens[i+1])
		lit := tokens[i+2]
		col := resolveCol(field, allowed, alias)
		if col == "" {
			return "", nil, fmt.Errorf("filter: unknown field %q", field)
		}
		sqlOp, ok := odataOps[op]
		if !ok {
			return "", nil, fmt.Errorf("filter: unsupported op %q", op)
		}
		v, err := parseLiteral(lit)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, fmt.Sprintf("%s %s $%d", col, sqlOp, argIdx))
		args = append(args, v)
		argIdx++
		i += 3
		if i < len(tokens) {
			conj := strings.ToLower(tokens[i])
			if conj != "and" && conj != "or" {
				return "", nil, fmt.Errorf("filter: expected and/or, got %q", conj)
			}
			parts = append(parts, strings.ToUpper(conj))
			i++
		}
	}
	return strings.Join(parts, " "), args, nil
}

var odataOps = map[string]string{
	"eq": "=", "ne": "<>", "gt": ">", "ge": ">=", "lt": "<", "le": "<=",
}

func tokenize(s string) ([]string, error) {
	var toks []string
	var cur strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\'' && !inStr:
			inStr = true
			cur.WriteByte(ch)
		case ch == '\'' && inStr:
			cur.WriteByte(ch)
			// handle escaped '' inside string
			if i+1 < len(s) && s[i+1] == '\'' {
				cur.WriteByte('\'')
				i++
				continue
			}
			inStr = false
		case ch == ' ' && !inStr:
			if cur.Len() > 0 {
				toks = append(toks, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(ch)
		}
	}
	if inStr {
		return nil, fmt.Errorf("unterminated string in filter")
	}
	if cur.Len() > 0 {
		toks = append(toks, cur.String())
	}
	return toks, nil
}

func parseLiteral(lit string) (any, error) {
	if strings.HasPrefix(lit, "'") && strings.HasSuffix(lit, "'") {
		return strings.ReplaceAll(lit[1:len(lit)-1], "''", "'"), nil
	}
	switch strings.ToLower(lit) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	if n, err := strconv.ParseInt(lit, 10, 64); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(lit, 64); err == nil {
		return f, nil
	}
	return lit, nil
}
