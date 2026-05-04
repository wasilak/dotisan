package resource

import "testing"

func TestParseResourceID_Valid(t *testing.T) {
	cases := []struct {
		in        string
		wantKind  string
		wantGroup string
		wantItem  string
	}{
		{"Kind", "Kind", "", ""},
		{"Kind/Group", "Kind", "Group", ""},
		{"Kind/Group[item]", "Kind", "Group", "item"},
		{"Kind/Group[user/repo]", "Kind", "Group", "user/repo"},
		{"ns/Kind/Group[item]", "Kind", "Group", "item"},
		{"ns/Kind/Group[user/repo]", "Kind", "Group", "user/repo"},
		{"ns/Kind/Group", "Kind", "Group", ""},
	}
	for _, c := range cases {
		rid, err := ParseResourceID(c.in)
		if err != nil {
			t.Fatalf("input %q: unexpected err: %v", c.in, err)
		}
		if rid.Kind != c.wantKind || rid.Group != c.wantGroup || rid.Item != c.wantItem {
			t.Fatalf("input %q: got %#v want kind=%s group=%s item=%s", c.in, rid, c.wantKind, c.wantGroup, c.wantItem)
		}
	}
}

func TestParseResourceID_Invalid(t *testing.T) {
	cases := []string{"", "[foo]", "Kind/Group[unclosed", "Kind//Group"}
	for _, in := range cases {
		if _, err := ParseResourceID(in); err == nil {
			t.Fatalf("expected error for input %q", in)
		}
	}
}
