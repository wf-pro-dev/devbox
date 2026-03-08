package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

const usage = `devbox-cli — manage files on your devbox server

Usage:
  devbox-cli [--server URL] <command> [args]

Commands:
  ls                     list all files
  ls --tag <tag>         filter by tag
  push <file> [flags]    upload a file
  pull <id|name>         download a file to the current directory
  deliver <id|name>      push a file to machines on your tailnet

Global flags:
  --server   override server URL (default: $DEVBOX_SERVER)

Push flags:
  --desc     description
  --tags     comma-separated tags (e.g. bash,deploy)
  --lang     language override (default: auto-detected from extension)

Deliver flags:
  --to       target hostname (repeatable)
  --all      deliver to all online machines
  --dest     destination directory on target (default: ~/devbox-received)

Setup (run once on each machine):
  echo 'export DEVBOX_SERVER=https://wwwill-1.your-tailnet.ts.net' >> ~/.bashrc

Examples:
  devbox-cli ls
  devbox-cli push deploy.sh --desc "main deploy" --tags bash,deploy
  devbox-cli pull deploy.sh
  devbox-cli deliver deploy.sh --to macbook --to linux-box
  devbox-cli deliver deploy.sh --all
`

type devFile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Size        int64    `json:"size"`
	UploadedBy  string   `json:"uploaded_by"`
	CreatedAt   string   `json:"created_at"`
	Tags        []string `json:"tags"`
}

func main() {
	serverFlag := flag.String("server", "", "devbox server URL (overrides $DEVBOX_SERVER)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	server, err := resolveServer(*serverFlag)
	if err != nil {
		fatalf("%v", err)
	}

	switch cmd := args[0]; cmd {
	case "ls":
		runLS(server, args[1:])
	case "push":
		runPush(server, args[1:])
	case "pull":
		runPull(server, args[1:])
	case "deliver":
		runDeliver(server, args[1:])
	default:
		fatalf("unknown command %q\n\n%s", cmd, usage)
	}
}

// resolveServer returns the base URL of the devbox server.
// Priority: --server flag > $DEVBOX_SERVER env var.
func resolveServer(flagVal string) (string, error) {
	if flagVal != "" {
		return strings.TrimRight(flagVal, "/"), nil
	}
	if env := os.Getenv("DEVBOX_SERVER"); env != "" {
		return strings.TrimRight(env, "/"), nil
	}
	return "", fmt.Errorf(
		"server not set\n\n" +
			"Add to your shell profile and reload:\n" +
			"  export DEVBOX_SERVER=https://wwwill-1.your-tailnet.ts.net\n\n" +
			"Or pass it directly:\n" +
			"  devbox-cli --server https://wwwill-1.your-tailnet.ts.net ls",
	)
}

// ── ls ─────────────────────────────────────────────────────────────────────

func runLS(server string, args []string) {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	tag := fs.String("tag", "", "filter by tag")
	fs.Parse(args)

	url := server + "/files"
	if *tag != "" {
		url += "?tag=" + *tag
	}

	var files []devFile
	getJSON(url, &files)

	if len(files) == 0 {
		fmt.Println("no files found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tLANG\tSIZE\tTAGS\tUPLOADED BY\tCREATED")
	fmt.Fprintln(w, "──────────\t────────────────────────────\t──────────\t───────\t────────────────\t────────────\t────────────────")
	for _, f := range files {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			f.ID[:8],
			truncate(f.Name, 28),
			f.Language,
			formatBytes(f.Size),
			truncate(strings.Join(f.Tags, ","), 16),
			truncate(f.UploadedBy, 12),
			formatDate(f.CreatedAt),
		)
	}
	w.Flush()
	fmt.Printf("\n%d file(s)\n", len(files))
}

// ── push ───────────────────────────────────────────────────────────────────

func runPush(server string, args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	desc := fs.String("desc", "", "description")
	tags := fs.String("tags", "", "comma-separated tags")
	lang := fs.String("lang", "", "language override")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fatalf("push requires a file path\n  usage: devbox-cli push <file> [--desc ...] [--tags ...]")
	}

	path := fs.Arg(0)
	f, err := os.Open(path)
	if err != nil {
		fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fatalf("stat %s: %v", path, err)
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()
		part, err := mw.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			pw.CloseWithError(err)
			return
		}
		mw.WriteField("description", *desc)
		mw.WriteField("tags", *tags)
		if *lang != "" {
			mw.WriteField("language", *lang)
		}
	}()

	req, err := http.NewRequest("POST", server+"/files", pr)
	if err != nil {
		fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient().Do(req)
	if err != nil {
		fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result devFile
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fatalf("decode response: %v", err)
	}

	fmt.Printf("✓ uploaded  %s  (%s)\n", result.Name, formatBytes(fi.Size()))
	fmt.Printf("  id        %s\n", result.ID)
	fmt.Printf("  language  %s\n", result.Language)
	if len(result.Tags) > 0 {
		fmt.Printf("  tags      %s\n", strings.Join(result.Tags, ", "))
	}
}

// ── pull ───────────────────────────────────────────────────────────────────

func runPull(server string, args []string) {
	if len(args) == 0 {
		fatalf("pull requires a file id or name\n  usage: devbox-cli pull <id|name>")
	}

	fileID, fileName, err := resolveFileID(server, args[0])
	if err != nil {
		fatalf("%v", err)
	}

	resp, err := httpClient().Get(server + "/files/" + fileID)
	if err != nil {
		fatalf("download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	out, err := os.Create(fileName)
	if err != nil {
		fatalf("create %s: %v", fileName, err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		fatalf("write: %v", err)
	}

	fmt.Printf("✓ saved  %s  (%s)\n", fileName, formatBytes(n))
}

func resolveFileID(server, query string) (id, name string, err error) {
	var files []devFile
	getJSON(server+"/files", &files)

	// Exact UUID → UUID prefix → exact name → name prefix
	for _, f := range files {
		if f.ID == query {
			return f.ID, f.Name, nil
		}
	}
	for _, f := range files {
		if strings.HasPrefix(f.ID, query) {
			return f.ID, f.Name, nil
		}
	}
	for _, f := range files {
		if f.Name == query {
			return f.ID, f.Name, nil
		}
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name, query) {
			return f.ID, f.Name, nil
		}
	}
	return "", "", fmt.Errorf("no file matching %q", query)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func httpClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func getJSON(url string, out interface{}) {
	resp, err := httpClient().Get(url)
	if err != nil {
		fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		fatalf("decode: %v", err)
	}
}

func formatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const k = 1024
	sizes := []string{"B", "KB", "MB", "GB"}
	i, v := 0, float64(b)
	for v >= k && i < len(sizes)-1 {
		v /= k
		i++
	}
	if v == float64(int(v)) {
		return fmt.Sprintf("%d %s", int(v), sizes[i])
	}
	return fmt.Sprintf("%.1f %s", v, sizes[i])
}

func formatDate(iso string) string {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, iso); err == nil {
			return t.Format("02 Jan 06 15:04")
		}
	}
	if len(iso) >= 10 {
		return iso[:10]
	}
	return iso
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// ── deliver ────────────────────────────────────────────────────────────────

func runDeliver(server string, args []string) {
	fs := flag.NewFlagSet("deliver", flag.ExitOnError)
	var targets multiFlag
	all := fs.Bool("all", false, "deliver to all online machines")
	dest := fs.String("dest", "", "destination directory (default: ~/devbox-received)")
	fs.Var(&targets, "to", "target hostname (repeatable: --to mac --to linux-box)")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fatalf("deliver requires a file id or name\n  usage: devbox-cli deliver <id|name> --to <hostname> [--to <hostname>...] [--all]")
	}
	if !*all && len(targets) == 0 {
		fatalf("specify at least one --to <hostname> or use --all")
	}

	fileID, fileName, err := resolveFileID(server, fs.Arg(0))
	if err != nil {
		fatalf("%v", err)
	}

	type deliverReq struct {
		Targets   []string `json:"targets"`
		Broadcast bool     `json:"broadcast"`
		DestDir   string   `json:"dest_dir"`
	}
	type deliverResult struct {
		Target  string `json:"target"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}
	type deliverResp struct {
		Results []deliverResult `json:"results"`
	}

	reqBody := deliverReq{
		Targets:   []string(targets),
		Broadcast: *all,
		DestDir:   *dest,
	}

	body, _ := json.Marshal(reqBody)
	resp, err := httpClient().Post(
		server+"/files/"+fileID+"/deliver",
		"application/json",
		strings.NewReader(string(body)),
	)
	if err != nil {
		fatalf("deliver request: %v", err)
	}
	defer resp.Body.Close()

	var result deliverResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fatalf("decode response: %v", err)
	}

	fmt.Printf("Delivering %s:\n\n", fileName)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, r := range result.Results {
		status := "✓"
		detail := ""
		if !r.Success {
			status = "✗"
			detail = r.Error
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\n", status, r.Target, detail)
	}
	w.Flush()
}

// multiFlag is a flag.Value that accumulates repeated --to flags.
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
