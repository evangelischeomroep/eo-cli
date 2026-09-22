package main

import (
	"io"
	"regexp"
	"strings"
)

var (
	mdBold       = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	mdInlineCode = regexp.MustCompile("`([^`\n]+)`")
	mdHeading    = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	mdBullet     = regexp.MustCompile(`^(\s*)[-*]\s+`)
)

// mdWriter styles streamed Markdown line by line: fenced code blocks, headings,
// bullets, **bold** and `inline code`. Text is buffered until a newline so a
// line is styled in one go; call flush at the end for a trailing partial line.
// With styling disabled every byte passes through untouched, which keeps
// piped output clean.
type mdWriter struct {
	out    io.Writer
	styled bool
	buf    strings.Builder
	inCode bool
}

func newMDWriter(out io.Writer, styled bool) *mdWriter {
	return &mdWriter{out: out, styled: styled}
}

func (m *mdWriter) write(s string) {
	if !m.styled {
		io.WriteString(m.out, s)
		return
	}
	m.buf.WriteString(s)
	for {
		text := m.buf.String()
		i := strings.IndexByte(text, '\n')
		if i < 0 {
			return
		}
		line := text[:i]
		m.buf.Reset()
		m.buf.WriteString(text[i+1:])
		io.WriteString(m.out, m.renderLine(line)+"\n")
	}
}

func (m *mdWriter) flush() {
	if m.buf.Len() == 0 {
		return
	}
	io.WriteString(m.out, m.renderLine(m.buf.String()))
	m.buf.Reset()
}

func (m *mdWriter) renderLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") {
		m.inCode = !m.inCode
		return dim(line)
	}
	if m.inCode {
		return cyan(line)
	}
	if h := mdHeading.FindStringSubmatch(line); h != nil {
		return bold(h[2])
	}
	line = mdBullet.ReplaceAllString(line, "${1}• ")
	line = mdBold.ReplaceAllStringFunc(line, func(s string) string {
		return bold(mdBold.FindStringSubmatch(s)[1])
	})
	line = mdInlineCode.ReplaceAllStringFunc(line, func(s string) string {
		return cyan(mdInlineCode.FindStringSubmatch(s)[1])
	})
	return line
}
