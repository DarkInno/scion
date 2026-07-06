package problem

import (
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeProblemDefaultsStatusAndTitle(t *testing.T) {
	p := sanitizeProblem(New(200, "", ""), Defaults())
	if p.Status != http.StatusInternalServerError || p.Title != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("unexpected problem: %+v", p)
	}
}

func TestValidationProblemSanitizesErrors(t *testing.T) {
	p := Validation([]InvalidParam{
		{Detail: "required", Pointer: "#/email"},
		{Detail: "bad\r\nheader", Pointer: "#/name"},
		{Detail: "bad pointer", Pointer: "name"},
	})
	p = sanitizeProblem(p, Options{MaxErrors: 2})
	if p.Type != "/validation-error" || len(p.Errors) != 1 {
		t.Fatalf("unexpected validation problem: %+v", p)
	}
	if p.Errors[0].Pointer != "#/email" {
		t.Fatalf("unexpected pointer: %+v", p.Errors[0])
	}
}

func TestTypeBaseApplied(t *testing.T) {
	p := sanitizeProblem(Problem{Type: "quota", Status: http.StatusTooManyRequests, Title: "Too many"}, Options{
		TypeBase: "https://api.example.com/problems",
	})
	if p.Type != "https://api.example.com/problems/quota" {
		t.Fatalf("type = %q", p.Type)
	}
}

func TestSafeFieldTruncates(t *testing.T) {
	got := safeField(strings.Repeat("a", 10), 4)
	if got != "aaaa" {
		t.Fatalf("got %q", got)
	}
}
