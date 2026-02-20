package ui

import (
	"runtime"
	"strings"
)

func lineBreak() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

func RenderTable(headers []string, rows [][]string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = visibleLen(h)
	}

	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if l := visibleLen(row[i]); l > widths[i] {
				widths[i] = l
			}
		}
	}

	var b strings.Builder
	b.WriteString(renderDivider(widths))
	b.WriteString(renderRow(headers, widths))
	b.WriteString(renderDivider(widths))
	for _, row := range rows {
		b.WriteString(renderRow(row, widths))
	}
	b.WriteString(renderDivider(widths))
	return b.String()
}

func renderDivider(widths []int) string {
	var b strings.Builder
	b.WriteString("+")
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2))
		b.WriteString("+")
	}
	b.WriteString(lineBreak())
	return b.String()
}

func renderRow(cols []string, widths []int) string {
	var b strings.Builder
	b.WriteString("|")
	for i, w := range widths {
		val := ""
		if i < len(cols) {
			val = cols[i]
		}
		pad := w - visibleLen(val)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(" ")
		b.WriteString(val)
		b.WriteString(strings.Repeat(" ", pad+1))
		b.WriteString("|")
	}
	b.WriteString(lineBreak())
	return b.String()
}

func visibleLen(s string) int {
	return len(stripANSI(s))
}

func stripANSI(s string) string {
	out := make([]rune, 0, len(s))
	inSeq := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == 0x1b {
			inSeq = true
			continue
		}
		if inSeq {
			if ch == 'm' {
				inSeq = false
			}
			continue
		}
		out = append(out, rune(ch))
	}
	return string(out)
}
