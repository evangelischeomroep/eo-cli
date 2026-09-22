package main

import (
	"strings"
	"testing"
)

func TestParseAskArgs(t *testing.T) {
	o, err := parseAskArgs([]string{"-m", "gpt", "-c", "--system=be brief", "-f", "a.pdf", "-k", "Handboek", "--web", "what", "is", "this?"})
	if err != nil {
		t.Fatal(err)
	}
	if o.model != "gpt" || !o.cont || o.system != "be brief" || !o.web {
		t.Errorf("flags: %+v", o)
	}
	if len(o.files) != 1 || len(o.knowledge) != 1 {
		t.Errorf("repeatables: %+v", o)
	}
	if o.prompt != "what is this?" {
		t.Errorf("prompt = %q", o.prompt)
	}

	o, err = parseAskArgs([]string{"--", "-not", "a", "flag"})
	if err != nil || o.prompt != "-not a flag" {
		t.Errorf("-- handling: %+v %v", o, err)
	}

	if _, err := parseAskArgs([]string{"--bogus"}); err == nil {
		t.Error("unknown flag should error")
	}
	if _, err := parseAskArgs([]string{"-m"}); err == nil {
		t.Error("missing value should error")
	}
}

func TestAskWantsHelp(t *testing.T) {
	if !askWantsHelp([]string{"--help"}) || !askWantsHelp([]string{"help"}) {
		t.Error("help flags not detected")
	}
	if askWantsHelp([]string{"help", "me", "with", "go"}) || askWantsHelp([]string{"how", "does", "--help", "work"}) || askWantsHelp(nil) {
		t.Error("questions mentioning help must not trigger help")
	}
}

func TestMDWriterStyled(t *testing.T) {
	old := useColor
	useColor = true
	defer func() { useColor = old }()

	var out strings.Builder
	w := newMDWriter(&out, true)
	w.write("# Tit")
	w.write("le\n- item with **bold** and `code`\n```go\nfmt.Println()\n```\ntrailing")
	w.flush()
	got := out.String()

	if !strings.Contains(got, ansiBold+"Title"+ansiReset) {
		t.Errorf("heading not bold: %q", got)
	}
	if !strings.Contains(got, "• item") || !strings.Contains(got, ansiBold+"bold"+ansiReset) || !strings.Contains(got, ansiCyan+"code"+ansiReset) {
		t.Errorf("inline styling missing: %q", got)
	}
	if !strings.Contains(got, ansiCyan+"fmt.Println()"+ansiReset) {
		t.Errorf("code block not styled: %q", got)
	}
	if !strings.HasSuffix(got, "trailing") {
		t.Errorf("trailing partial line lost: %q", got)
	}
}

func TestMDWriterRawPassthrough(t *testing.T) {
	var out strings.Builder
	w := newMDWriter(&out, false)
	in := "# Title\n- **x** `y`\n```\ncode\n```\nend"
	w.write(in)
	w.flush()
	if out.String() != in {
		t.Errorf("raw mode altered output: %q", out.String())
	}
}
