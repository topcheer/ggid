package scim

import (
	"encoding/json"
	"testing"
)

// --- Filter Parser Tests ---

func TestParseFilter_SimpleEq(t *testing.T) {
	expr, err := ParseFilter(`userName eq "john"`)
	if err != nil {
		t.Fatal(err)
	}
	ae, ok := expr.(*AttrExpression)
	if !ok {
		t.Fatalf("expected *AttrExpression, got %T", expr)
	}
	if ae.AttrPath != "userName" {
		t.Errorf("attrPath = %q, want %q", ae.AttrPath, "userName")
	}
	if ae.Op != OpEq {
		t.Errorf("op = %q, want eq", ae.Op)
	}
	if ae.Value != "john" {
		t.Errorf("value = %v, want john", ae.Value)
	}
}

func TestParseFilter_Empty(t *testing.T) {
	expr, err := ParseFilter("")
	if err != nil {
		t.Fatal(err)
	}
	if expr != nil {
		t.Errorf("expected nil for empty filter, got %v", expr)
	}
}

func TestParseFilter_AllOperators(t *testing.T) {
	cases := []string{
		`x eq "v"`,
		`x ne "v"`,
		`x co "v"`,
		`x sw "v"`,
		`x ew "v"`,
		`x pr`,
		`x gt "v"`,
		`x ge "v"`,
		`x lt "v"`,
		`x le "v"`,
	}
	for _, c := range cases {
		_, err := ParseFilter(c)
		if err != nil {
			t.Errorf("ParseFilter(%q) error: %v", c, err)
		}
	}
}

func TestParseFilter_AndOr(t *testing.T) {
	cases := []string{
		`a eq "1" and b eq "2"`,
		`a eq "1" or b eq "2"`,
		`a eq "1" and b eq "2" and c eq "3"`,
		`a eq "1" or b eq "2" or c eq "3"`,
		`(a eq "1" or b eq "2") and c eq "3"`,
	}
	for _, c := range cases {
		_, err := ParseFilter(c)
		if err != nil {
			t.Errorf("ParseFilter(%q) error: %v", c, err)
		}
	}
}

func TestParseFilter_Not(t *testing.T) {
	expr, err := ParseFilter(`not (userName eq "john")`)
	if err != nil {
		t.Fatal(err)
	}
	ne, ok := expr.(*NotExpr)
	if !ok {
		t.Fatalf("expected *NotExpr, got %T", expr)
	}
	inner, ok := ne.Inner.(*AttrExpression)
	if !ok {
		t.Fatalf("expected inner *AttrExpression, got %T", ne.Inner)
	}
	if inner.AttrPath != "userName" {
		t.Errorf("inner attrPath = %q", inner.AttrPath)
	}
}

func TestParseFilter_BooleanLiteral(t *testing.T) {
	expr, err := ParseFilter(`active eq true`)
	if err != nil {
		t.Fatal(err)
	}
	ae := expr.(*AttrExpression)
	if ae.Value != true {
		t.Errorf("value = %v, want true", ae.Value)
	}

	expr2, _ := ParseFilter(`active eq false`)
	ae2 := expr2.(*AttrExpression)
	if ae2.Value != false {
		t.Errorf("value = %v, want false", ae2.Value)
	}
}

func TestParseFilter_MultivaluedPath(t *testing.T) {
	expr, err := ParseFilter(`emails[type eq "work"].value eq "john@example.com"`)
	if err != nil {
		t.Fatal(err)
	}
	ae := expr.(*AttrExpression)
	if ae.AttrPath == "" {
		t.Error("expected non-empty attrPath")
	}
}

func TestParseFilter_DottedPath(t *testing.T) {
	expr, err := ParseFilter(`name.familyName eq "Smith"`)
	if err != nil {
		t.Fatal(err)
	}
	ae := expr.(*AttrExpression)
	if ae.AttrPath != "name.familyName" {
		t.Errorf("attrPath = %q, want name.familyName", ae.AttrPath)
	}
}

func TestParseFilter_NumberValue(t *testing.T) {
	expr, err := ParseFilter(`age gt 42`)
	if err != nil {
		t.Fatal(err)
	}
	ae := expr.(*AttrExpression)
	if ae.Value != "42" {
		t.Errorf("value = %v, want 42", ae.Value)
	}
}

func TestParseFilter_Precedence(t *testing.T) {
	// "and" binds tighter than "or"
	expr, err := ParseFilter(`a eq "1" or b eq "2" and c eq "3"`)
	if err != nil {
		t.Fatal(err)
	}
	or, ok := expr.(*OrExpr)
	if !ok {
		t.Fatalf("expected top-level *OrExpr, got %T", expr)
	}
	// Right side should be an AndExpr
	if _, ok := or.Right.(*AndExpr); !ok {
		t.Fatalf("expected right side *AndExpr, got %T", or.Right)
	}
}

func TestParseFilter_NestedParens(t *testing.T) {
	expr, err := ParseFilter(`((a eq "1"))`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := expr.(*AttrExpression); !ok {
		t.Fatalf("expected *AttrExpression, got %T", expr)
	}
}

func TestParseFilter_InvalidInput(t *testing.T) {
	cases := []string{
		`eq "value"`,      // starts with operator
		`userName eq`,      // missing value
		`userName pr "v"`,  // pr with value
		`userName bogus "v"`, // invalid operator
		`userName eq `,     // missing value
	}
	for _, c := range cases {
		_, err := ParseFilter(c)
		if err == nil {
			t.Errorf("ParseFilter(%q) expected error", c)
		}
	}
}

func TestParseFilter_UnterminatedString(t *testing.T) {
	_, err := ParseFilter(`userName eq "john`)
	if err == nil {
		t.Error("expected error for unterminated string")
	}
}

func TestParseFilter_EscapedQuote(t *testing.T) {
	expr, err := ParseFilter(`userName eq "jo\"hn"`)
	if err != nil {
		t.Fatal(err)
	}
	ae := expr.(*AttrExpression)
	if ae.Value != "jo\"hn" {
		t.Errorf("value = %q, want jo\"hn", ae.Value)
	}
}

// --- Filter Evaluation Tests ---

func TestEvaluate_SimpleEq(t *testing.T) {
	attrs := map[string]any{
		"userName": "john",
		"active":   true,
	}
	tests := []struct {
		filter string
		want   bool
	}{
		{`userName eq "john"`, true},
		{`userName eq "jane"`, false},
		{`userName eq "JOHN"`, true},  // case-insensitive
		{`active eq true`, true},
		{`active eq false`, false},
		{`userName pr`, true},
		{`missingAttr pr`, false},
	}
	for _, tt := range tests {
		expr, err := ParseFilter(tt.filter)
		if err != nil {
			t.Errorf("ParseFilter(%q) error: %v", tt.filter, err)
			continue
		}
		got := expr.Evaluate(attrs)
		if got != tt.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestEvaluate_Contains(t *testing.T) {
	attrs := map[string]any{"userName": "johnny"}
	tests := []struct {
		filter string
		want   bool
	}{
		{`userName co "john"`, true},
		{`userName co "xyz"`, false},
		{`userName sw "john"`, true},
		{`userName sw "xyz"`, false},
		{`userName ew "nny"`, true},
		{`userName ew "xyz"`, false},
	}
	for _, tt := range tests {
		expr, _ := ParseFilter(tt.filter)
		got := expr.Evaluate(attrs)
		if got != tt.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestEvaluate_AndOr(t *testing.T) {
	attrs := map[string]any{
		"userName": "john",
		"active":   true,
		"age":      "25",
	}
	tests := []struct {
		filter string
		want   bool
	}{
		{`userName eq "john" and active eq true`, true},
		{`userName eq "john" and active eq false`, false},
		{`userName eq "john" or active eq false`, true},
		{`userName eq "jane" or active eq false`, false},
		{`age gt "20"`, true},
		{`age lt "30"`, true},
	}
	for _, tt := range tests {
		expr, err := ParseFilter(tt.filter)
		if err != nil {
			t.Errorf("ParseFilter(%q) error: %v", tt.filter, err)
			continue
		}
		got := expr.Evaluate(attrs)
		if got != tt.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestEvaluate_Not(t *testing.T) {
	attrs := map[string]any{"userName": "john"}
	expr, err := ParseFilter(`not (userName eq "jane")`)
	if err != nil {
		t.Fatal(err)
	}
	if !expr.Evaluate(attrs) {
		t.Error("not (userName eq \"jane\") should be true when userName is john")
	}
}

func TestEvaluate_MultivaluedFilter(t *testing.T) {
	attrs := map[string]any{
		"emails": []any{
			map[string]any{"value": "work@example.com", "type": "work", "primary": true},
			map[string]any{"value": "home@example.com", "type": "home", "primary": false},
		},
	}
	tests := []struct {
		filter string
		want   bool
	}{
		{`emails[type eq "work"].value eq "work@example.com"`, true},
		{`emails[type eq "work"].value eq "home@example.com"`, false},
		{`emails[type eq "home"].value eq "home@example.com"`, true},
		{`emails[primary eq true].value eq "work@example.com"`, true},
	}
	for _, tt := range tests {
		expr, err := ParseFilter(tt.filter)
		if err != nil {
			t.Errorf("ParseFilter(%q) error: %v", tt.filter, err)
			continue
		}
		got := expr.Evaluate(attrs)
		if got != tt.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestEvaluate_DottedAttr(t *testing.T) {
	attrs := map[string]any{
		"name": map[string]any{
			"givenName":  "John",
			"familyName": "Smith",
		},
	}
	expr, err := ParseFilter(`name.familyName eq "Smith"`)
	if err != nil {
		t.Fatal(err)
	}
	if !expr.Evaluate(attrs) {
		t.Error("name.familyName eq Smith should be true")
	}
}

func TestEvaluate_CaseInsensitiveAttr(t *testing.T) {
	attrs := map[string]any{"DisplayName": "John"}
	expr, err := ParseFilter(`displayName eq "john"`)
	if err != nil {
		t.Fatal(err)
	}
	if !expr.Evaluate(attrs) {
		t.Error("should match case-insensitively")
	}
}

// --- PATCH Engine Tests ---

func TestApplyPatch_AddSimple(t *testing.T) {
	attrs := map[string]any{"userName": "john"}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "add", Path: "displayName", Value: json.RawMessage(`"John Doe"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["displayName"] != "John Doe" {
		t.Errorf("displayName = %v, want John Doe", result["displayName"])
	}
}

func TestApplyPatch_ReplaceSimple(t *testing.T) {
	attrs := map[string]any{"userName": "john", "displayName": "Old"}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "replace", Path: "displayName", Value: json.RawMessage(`"New"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["displayName"] != "New" {
		t.Errorf("displayName = %v, want New", result["displayName"])
	}
}

func TestApplyPatch_RemoveSimple(t *testing.T) {
	attrs := map[string]any{"userName": "john", "displayName": "John"}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "remove", Path: "displayName"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := result["displayName"]; exists {
		t.Error("displayName should be removed")
	}
}

func TestApplyPatch_AddNestedAttr(t *testing.T) {
	attrs := map[string]any{}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "add", Path: "name.familyName", Value: json.RawMessage(`"Smith"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	name, ok := result["name"].(map[string]any)
	if !ok {
		t.Fatalf("name is not a map: %T", result["name"])
	}
	if name["familyName"] != "Smith" {
		t.Errorf("name.familyName = %v, want Smith", name["familyName"])
	}
}

func TestApplyPatch_AddToArray(t *testing.T) {
	attrs := map[string]any{
		"emails": []any{
			map[string]any{"value": "old@example.com", "type": "work"},
		},
	}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "add", Path: "emails", Value: json.RawMessage(`{"value":"new@example.com","type":"home"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	emails := result["emails"].([]any)
	if len(emails) != 2 {
		t.Fatalf("emails len = %d, want 2", len(emails))
	}
}

func TestApplyPatch_ReplaceInArrayWithFilter(t *testing.T) {
	attrs := map[string]any{
		"emails": []any{
			map[string]any{"value": "old@example.com", "type": "work"},
			map[string]any{"value": "home@example.com", "type": "home"},
		},
	}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{
			Op:    "replace",
			Path:  `emails[type eq "work"].value`,
			Value: json.RawMessage(`"new@example.com"`),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	emails := result["emails"].([]any)
	workEmail := emails[0].(map[string]any)
	if workEmail["value"] != "new@example.com" {
		t.Errorf("work email = %v, want new@example.com", workEmail["value"])
	}
	// Home email should be unchanged
	homeEmail := emails[1].(map[string]any)
	if homeEmail["value"] != "home@example.com" {
		t.Errorf("home email = %v, should be unchanged", homeEmail["value"])
	}
}

func TestApplyPatch_RemoveFromArrayWithFilter(t *testing.T) {
	attrs := map[string]any{
		"emails": []any{
			map[string]any{"value": "work@example.com", "type": "work"},
			map[string]any{"value": "home@example.com", "type": "home"},
		},
	}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "remove", Path: `emails[type eq "home"]`},
	})
	if err != nil {
		t.Fatal(err)
	}
	emails := result["emails"].([]any)
	if len(emails) != 1 {
		t.Fatalf("emails len = %d, want 1", len(emails))
	}
	remaining := emails[0].(map[string]any)
	if remaining["type"] != "work" {
		t.Errorf("remaining email type = %v, want work", remaining["type"])
	}
}

func TestApplyPatch_MultipleOps(t *testing.T) {
	attrs := map[string]any{
		"userName": "john",
		"active":   true,
	}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "replace", Path: "userName", Value: json.RawMessage(`"johnny"`)},
		{Op: "add", Path: "displayName", Value: json.RawMessage(`"John"`)},
		{Op: "replace", Path: "active", Value: json.RawMessage(`false`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["userName"] != "johnny" {
		t.Errorf("userName = %v", result["userName"])
	}
	if result["displayName"] != "John" {
		t.Errorf("displayName = %v", result["displayName"])
	}
	if result["active"] != false {
		t.Errorf("active = %v", result["active"])
	}
}

func TestApplyPatch_InvalidOp(t *testing.T) {
	_, err := ApplyPatch(map[string]any{}, []PatchOperation{
		{Op: "bogus", Path: "x", Value: json.RawMessage(`"y"`)},
	})
	if err == nil {
		t.Error("expected error for invalid op")
	}
}

func TestApplyPatch_AddWithoutPath(t *testing.T) {
	attrs := map[string]any{}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "add", Value: json.RawMessage(`{"userName":"john","displayName":"John"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["userName"] != "john" {
		t.Errorf("userName = %v", result["userName"])
	}
	if result["displayName"] != "John" {
		t.Errorf("displayName = %v", result["displayName"])
	}
}

func TestApplyPatch_RemoveRequiresPath(t *testing.T) {
	_, err := ApplyPatch(map[string]any{}, []PatchOperation{
		{Op: "remove"},
	})
	if err == nil {
		t.Error("expected error for remove without path")
	}
}

func TestApplyPatch_RemoveNestedAttr(t *testing.T) {
	attrs := map[string]any{
		"name": map[string]any{
			"givenName":  "John",
			"familyName": "Smith",
		},
	}
	result, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "remove", Path: "name.familyName"},
	})
	if err != nil {
		t.Fatal(err)
	}
	name := result["name"].(map[string]any)
	if _, exists := name["familyName"]; exists {
		t.Error("name.familyName should be removed")
	}
	if name["givenName"] != "John" {
		t.Error("name.givenName should be preserved")
	}
}

func TestApplyPatch_DoesNotMutateOriginal(t *testing.T) {
	attrs := map[string]any{"userName": "john"}
	_, err := ApplyPatch(attrs, []PatchOperation{
		{Op: "replace", Path: "userName", Value: json.RawMessage(`"changed"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attrs["userName"] != "john" {
		t.Error("original map should not be mutated")
	}
}

func TestParsePatchPath_Simple(t *testing.T) {
	name, sub, filter := parsePatchPath("displayName")
	if name != "displayName" || sub != "" || filter != "" {
		t.Errorf("got name=%q sub=%q filter=%q", name, sub, filter)
	}
}

func TestParsePatchPath_Dotted(t *testing.T) {
	name, sub, filter := parsePatchPath("name.familyName")
	if name != "name" || sub != "familyName" || filter != "" {
		t.Errorf("got name=%q sub=%q filter=%q", name, sub, filter)
	}
}

func TestParsePatchPath_WithFilter(t *testing.T) {
	name, sub, filter := parsePatchPath(`emails[type eq "work"].value`)
	if name != "emails" {
		t.Errorf("name = %q", name)
	}
	if sub != "value" {
		t.Errorf("sub = %q", sub)
	}
	if filter != `type eq "work"` {
		t.Errorf("filter = %q", filter)
	}
}

func TestParsePatchPath_FilterNoSubPath(t *testing.T) {
	name, sub, filter := parsePatchPath(`emails[type eq "work"]`)
	if name != "emails" || sub != "" {
		t.Errorf("name=%q sub=%q", name, sub)
	}
	if filter == "" {
		t.Error("expected non-empty filter")
	}
}

func TestPatchedAttrsToSCIMUser(t *testing.T) {
	attrs := map[string]any{
		"id":          "abc-123",
		"userName":    "john",
		"displayName": "John Doe",
		"active":      true,
		"emails": []any{
			map[string]any{"value": "john@example.com", "type": "work", "primary": true},
		},
		"name": map[string]any{
			"givenName":  "John",
			"familyName": "Doe",
		},
	}
	u := PatchedAttrsToSCIMUser(attrs)
	if u.ID != "abc-123" {
		t.Errorf("ID = %q", u.ID)
	}
	if u.UserName != "john" {
		t.Errorf("UserName = %q", u.UserName)
	}
	if u.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q", u.DisplayName)
	}
	if !u.Active {
		t.Error("Active should be true")
	}
	if len(u.Emails) != 1 {
		t.Fatalf("Emails len = %d", len(u.Emails))
	}
	if u.Emails[0].Value != "john@example.com" {
		t.Errorf("email = %v", u.Emails[0])
	}
	if u.Name.GivenName != "John" {
		t.Errorf("GivenName = %q", u.Name.GivenName)
	}
	if u.Name.FamilyName != "Doe" {
		t.Errorf("FamilyName = %q", u.Name.FamilyName)
	}
}
