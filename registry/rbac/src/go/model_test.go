package rbac

import "testing"

func TestModelPermissionParsingAndMatching(t *testing.T) {
	perm := ParsePermission("posts:write")
	if perm.Resource != "posts" || perm.Action != "write" || perm.String() != "posts:write" {
		t.Fatalf("unexpected permission: %+v", perm)
	}
	if !matches(ParsePermission("posts:*"), perm) {
		t.Fatal("resource wildcard should match")
	}
	if !matches(ParsePermission("*:write"), perm) {
		t.Fatal("action wildcard should match")
	}
	if validateName("") || validateName("bad\r\n") {
		t.Fatal("unsafe names should be rejected")
	}
}
