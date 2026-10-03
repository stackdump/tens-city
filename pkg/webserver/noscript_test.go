package webserver

import (
	"strings"
	"testing"
)

func TestRenderNoscriptPosts(t *testing.T) {
	index := []byte(`{"itemListElement":[
		{"item":{"headline":"First <Post>","url":"https://example.com/posts/first","description":"About & more","datePublished":"2026-09-30T00:00:00Z"}},
		{"item":{"headline":"Second","url":"https://example.com/posts/second","datePublished":"2026-09-01T00:00:00Z"}},
		{"item":{"headline":"Third","url":"https://example.com/posts/third"}}
	]}`)

	out := renderNoscriptPosts(index, 2)
	for _, want := range []string{
		"<noscript>",
		`<a href="/posts/first">First &lt;Post&gt;</a>`,
		`<time datetime="2026-09-30T00:00:00Z">September 30, 2026</time>`,
		"About &amp; more",
		`<a href="/posts/second">Second</a>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Third") {
		t.Errorf("limit not applied:\n%s", out)
	}

	if got := renderNoscriptPosts([]byte(`{"itemListElement":[]}`), 10); got != "" {
		t.Errorf("empty index should render nothing, got %q", got)
	}
	if got := renderNoscriptPosts([]byte(`not json`), 10); got != "" {
		t.Errorf("bad index should render nothing, got %q", got)
	}
}
