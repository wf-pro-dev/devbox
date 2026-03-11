package internal

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// tw returns a tabwriter that flushes to stdout.
func Tw() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// shortID returns the first 8 chars of a UUID.
func ShortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// shortSHA returns the first 12 chars of a sha256 hex string.
func ShortSHA(sha string) string {
	if len(sha) >= 12 {
		return sha[:12]
	}
	return sha
}

// fmtTags formats a slice of tag names for display.
func FmtTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ", ")
}

// fmtSize formats bytes as a human-readable string.
func FmtSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%dB", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1fM", float64(n)/(1024*1024))
	}
}

// fmtDate trims the T and Z off an ISO timestamp for compact display.
func FmtDate(ts string) string {
	ts = strings.Replace(ts, "T", " ", 1)
	ts = strings.TrimSuffix(ts, "Z")
	if len(ts) > 16 {
		return ts[:16]
	}
	return ts
}

// die prints an error and exits 1.
func Die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// warn prints a warning without exiting.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
