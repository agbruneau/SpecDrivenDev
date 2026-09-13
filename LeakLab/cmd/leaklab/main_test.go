package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestUC003_CommandeCtxvet(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"ctxvet", "../../lab/corpus"}, &out, &errOut); code != 1 {
		t.Fatalf("code %d, 1 attendu (diagnostics) ; %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "io_without_context_leak.go") || !strings.Contains(out.String(), "net.Dial ignore le contexte") {
		t.Fatalf("sortie sans le diagnostic attendu :\n%s", out.String())
	}
	if code := run(context.Background(), []string{"ctxvet", "../../internal/results"}, &out, &errOut); code != 0 {
		t.Fatalf("paquet sain : code %d", code)
	}
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"inconnue"}, {"verdict"}, {"ctxvet"}} {
		var out, errOut bytes.Buffer
		if code := run(context.Background(), args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage") {
			t.Errorf("%v : code %d, sortie %q", args, code, errOut.String())
		}
	}
}
