package main

import (
	"fmt"
	"os"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiPurple = "\033[38;5;93m"
)

var useColor = shouldColor()

func shouldColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string   { return paint(ansiBold, s) }
func dim(s string) string    { return paint(ansiDim, s) }
func red(s string) string    { return paint(ansiRed, s) }
func green(s string) string  { return paint(ansiGreen, s) }
func yellow(s string) string { return paint(ansiYellow, s) }
func cyan(s string) string   { return paint(ansiCyan, s) }
func purple(s string) string { return paint(ansiPurple, s) }

// pad right-pads s with spaces to width w.
func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

const banner = `
    ______ ____     ______ __     ____
   / ____// __ \   / ____// /    /  _/
  / __/  / / / /  / /    / /     / /
 / /___ / /_/ /  / /___ / /___ _/ /
/_____/ \____/   \____//_____//___/
`

func printBanner() {
	if !useColor {
		return
	}
	fmt.Println(purple(banner))
	fmt.Println(dim("  Evangelische Omroep — developer CLI"))
	fmt.Println()
}

// helpDoc builds a help page as a single string and prints it at once.
type helpDoc struct{ strings.Builder }

func (h *helpDoc) line(s string) { h.WriteString(s); h.WriteByte('\n') }
func (h *helpDoc) blank()        { h.WriteByte('\n') }
func (h *helpDoc) section(title string) {
	h.blank()
	h.line(bold(title))
}
func (h *helpDoc) cmd(name, desc string) {
	h.WriteString(fmt.Sprintf("  %s  %s\n", cyan(pad(name, 16)), desc))
}
func (h *helpDoc) flag(name, desc string) {
	h.WriteString(fmt.Sprintf("  %s  %s\n", pad(name, 16), desc))
}
func (h *helpDoc) example(comment, cmd string) {
	if comment != "" {
		h.line(dim("  # " + comment))
	}
	h.line("  " + cmd)
	h.blank()
}
func (h *helpDoc) print() { fmt.Print(h.String()) }
