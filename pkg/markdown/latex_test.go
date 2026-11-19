package markdown

import (
	"strings"
	"testing"
	"time"
)

const latexFront = `---
title: Math
description: dollars and math
datePublished: 2026-11-19T00:00:00Z
author:
  name: Test Author
  type: Person
lang: en
slug: math
---

`

// render parses a post body, failing (instead of hanging the test run) if rendering does not finish.
func render(t *testing.T, body string) string {
	t.Helper()
	type res struct {
		html string
		err  error
	}
	done := make(chan res, 1)
	go func() {
		doc, err := ParseDocumentFromBytes([]byte(latexFront+body), "math.md")
		if err != nil {
			done <- res{err: err}
			return
		}
		done <- res{html: doc.HTML}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("ParseDocumentFromBytes: %v", r.err)
		}
		return r.html
	case <-time.After(5 * time.Second):
		t.Fatalf("rendering %q did not finish: the inline math parser is looping", body)
		return ""
	}
}

// A lone dollar sign used to hang goldmark-latex's inline parser forever (blog.stackdump.com, 2026-11-19).
func TestALoneDollarSignIsText(t *testing.T) {
	for _, body := range []string{
		"The composes were about $0.015 each. That is a sample of four.\n",
		"Ends with a dollar $\n",
		"A price $5 on one line\nand more text on the next\n",
		"$\n",
		"Two paragraphs, the first $ open.\n\nThe second has none.\n",
	} {
		html := render(t, body)
		if !strings.Contains(html, "$") {
			t.Errorf("%q: the dollar sign was dropped: %s", body, html)
		}
		if strings.Contains(html, `class="math inline"`) {
			t.Errorf("%q: a lone $ became math: %s", body, html)
		}
	}
}

func TestInlineAndBlockMathStillRender(t *testing.T) {
	if html := render(t, "Energy is $E = mc^2$ here.\n"); !strings.Contains(html, `class="math inline"`) || !strings.Contains(html, "E = mc^2") {
		t.Errorf("inline math not rendered: %s", html)
	}
	if html := render(t, "Across $a +\nb$ two lines.\n"); !strings.Contains(html, `class="math inline"`) {
		t.Errorf("inline math across a line break not rendered: %s", html)
	}
	if html := render(t, "$$\nx^2 + y^2\n$$\n"); !strings.Contains(html, `class="math block"`) {
		t.Errorf("block math not rendered: %s", html)
	}
	// an unclosed block must not hang either
	render(t, "$$\nx^2 never closed\n")
}
