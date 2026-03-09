package main

import (
	"bytes"
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
  devbox-cli <command> [flags] [args]

Commands:
  ls                     list all files
  ls --tag <tag>         filter by tag
  info <id|name>         show file information
  push <file> [flags]    upload a file
  push -r <dir> [flags]  upload an entire directory
  pull <id|name>         download a file to the current directory
  tag <id|name> <tag>    add a tag to a file
  untag <id|name> <tag>  remove a tag from a file
  deliver <id|name>      push a file to machines on your tailnet
  update <id|name> <file>         update file content (skips if unchanged)
  update -r <dir-id> <dir>        sync a directory (update/add files)

Global flags:
  --server   override server URL (default: $DEVBOX_SERVER)

Push flags:
  --desc     description
  --tags     comma-separated tags (e.g. bash,deploy)
  --lang     language override (default: auto-detected from extension)
  -r         push entire directory

Update flags:
  -r         sync entire directory
  -m         version message (optional)

Deliver flags:
  --to       target hostname (repeatable)
  --all      deliver to all online machines
  --dest     destination directory on target (default: ~/devbox-received)

Setup (run once on each machine):
  echo 'export DEVBOX_SERVER=https://wwwill-1.your-tailnet.ts.net' >> ~/.bashrc

Examples:
  devbox-cli ls
  devbox-cli push --desc "main deploy" --tags bash,deploy deploy.sh
  devbox-cli push -r ./nginx/ --tags nginx,config
  devbox-cli pull deploy.sh
  devbox-cli tag deploy.sh deploy
  devbox-cli untag deploy.sh deploy
  devbox-cli deliver deploy.sh --to macbook --to linux-box
  devbox-cli deliver deploy.sh --all
  devbox-cli update deploy.sh ./deploy.sh
  devbox-cli update deploy.sh ./deploy.sh -m "fix: retry logic"
  devbox-cli update -r nginx-dir-id ./nginx/
`

type devFile struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	FileName    string   `json:"file_name"`
	DirID       *string  `json:"dir_id"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Size        int64    `json:"size"`
	Version     int64    `json:"version"`
	UploadedBy  string   `json:"uploaded_by"`
	CreatedAt   string   `json:"created_at"`
	Tags        []string `json:"tags"`
}

type devDir struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Prefix      string `json:"prefix"`
	Description string `json:"description"`
	UploadedBy  string `json:"uploaded_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func main() {
	serverFlag := flag.String("server", "", "devbox server URL (overrides $DEVBOX_SERVER)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	args := os.Args[1:]
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
	case "dirs":
		runDirLS(server, args[1:])
	case "info":
		runInfo(server, args[1:])
	case "push":
		runPush(server, args[1:])
	case "pull":
		runPull(server, args[1:])
	case "tag":
		runTag(server, args[1:])
	case "untag":
		runUntag(server, args[1:])
	case "deliver":
		runDeliver(server, args[1:])
	case "update":
		runUpdate(server, args[1:])
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
	fmt.Fprintln(w, "ID\tNAME\tVER\tLANG\tSIZE\tTAGS\tUPLOADED BY\tCREATED")
	fmt.Fprintln(w, "──────────\t────────────────────────────\t───\t──────────\t───────\t────────────────\t────────────\t────────────────")
	for _, f := range files {
		fmt.Fprintf(w, "%s\t%s\tv%d\t%s\t%s\t%s\t%s\t%s\n",
			f.ID[:8],
			truncate(f.FileName, 28),
			f.Version,
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

func runDirLS(server string, args []string) {
	fs := flag.NewFlagSet("ls", flag.ExitOnError)
	tag := fs.String("tag", "", "filter by tag")
	fs.Parse(args)

	url := server + "/directories"
	if *tag != "" {
		url += "?tag=" + *tag
	}

	var dirs []devDir
	getJSON(url, &dirs)

	if len(dirs) == 0 {
		fmt.Println("no directories found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPREFIX\tDESCRIPTION\tUPLOADED BY\tCREATED\tUPDATED")
	fmt.Fprintln(w, "──────────\t────────────────────────────\t───────-----\t──────────\t───────\t────────────────\t────────────")
	for _, d := range dirs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			d.ID[:8],
			truncate(d.Name, 28),
			d.Prefix,
			d.Description,
			d.UploadedBy,
			formatDate(d.CreatedAt),
			formatDate(d.UpdatedAt),
		)
	}
	w.Flush()
	fmt.Printf("\n%d directory(ies)\n", len(dirs))
}

// ── info ────────────────────────────────────────────────────────────────────

func runInfo(server string, args []string) {
	if len(args) == 0 {
		fatalf("info requires a file id or name\n  usage: devbox-cli info <id|name>")
	}

	file, err := resolevFile(server, args[0])
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Printf("Name:        %s\n", file.FileName)
	fmt.Printf("Path:        %s\n", file.Path)
	fmt.Printf("Version:     v%d\n", file.Version)
	fmt.Printf("Description: %s\n", file.Description)
	fmt.Printf("Language:    %s\n", file.Language)
	fmt.Printf("Tags:        %s\n", strings.Join(file.Tags, ", "))
	fmt.Printf("Size:        %s\n", formatBytes(file.Size))
	fmt.Printf("Uploaded By: %s\n", file.UploadedBy)
	fmt.Printf("Created At:  %s\n", file.CreatedAt)
	fmt.Printf("ID:          %s\n", file.ID)
}

// ── push ───────────────────────────────────────────────────────────────────

func runPush(server string, args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	desc := fs.String("desc", "", "description")
	tags := fs.String("tags", "", "comma-separated tags")
	lang := fs.String("lang", "", "language override")
	recursive := fs.Bool("r", false, "push entire directory")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fatalf("push requires a path\n  usage: devbox-cli push <file|dir> [-r] [--desc ...] [--tags ...]")
	}

	target := fs.Arg(0)

	if *recursive {
		runPushDir(server, target, *desc, *tags)
		return
	}

	// Single file push.
	f, err := os.Open(target)
	if err != nil {
		fatalf("open %s: %v", target, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		fatalf("stat %s: %v", target, err)
	}
	if fi.IsDir() {
		fatalf("%s is a directory — use -r to push directories", target)
	}

	// Create a buffer to store the multipart data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	defer writer.Close()

	// Create form file field
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(target))
	if err != nil {
		return
	}

	// Open the file
	fileContent, err := os.Open(target)
	if err != nil {
		fatalf("failed to open file: %v", err)
	}
	defer fileContent.Close()

	// Copy file content to form
	_, err = io.Copy(fileWriter, fileContent)
	if err != nil {
		fatalf("error copying file content: %v", err)
	}

	metadataWriter, err := writer.CreateFormField("metadata")
	if err != nil {
		fatalf("error creating form field: %v", err)
	}

	metadataJSON, err := json.Marshal(map[string]string{"description": *desc, "tags": *tags, "language": *lang})
	if err != nil {
		fatalf("error marshalling metadata: %v", err)
	}

	_, err = metadataWriter.Write(metadataJSON)
	if err != nil {
		fatalf("error writing metadata: %v", err)
	}

	// Close the writer to finalize the multipart message
	writer.Close()

	req, err := http.NewRequest("POST", server+"/files", &buf)
	if err != nil {
		fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

	fmt.Printf("✓ uploaded  %s  (%s)\n", result.Path, formatBytes(fi.Size()))
	fmt.Printf("  id        %s\n", result.ID)
	fmt.Printf("  language  %s\n", result.Language)
	if len(result.Tags) > 0 {
		fmt.Printf("  tags      %s\n", strings.Join(result.Tags, ", "))
	}
}

// runPushDir uploads an entire directory to POST /directories.
func runPushDir(server, dirPath, desc, tags string) {
	dirPath = filepath.Clean(dirPath)
	fi, err := os.Stat(dirPath)
	if err != nil {
		fatalf("stat %s: %v", dirPath, err)
	}
	if !fi.IsDir() {
		fatalf("%s is not a directory (remove -r for single file push)", dirPath)
	}

	dirName := filepath.Base(dirPath)
	fmt.Printf("Scanning %s...\n", dirPath)

	type entry struct {
		absPath string
		relPath string
	}
	var entries []entry
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dirPath, path)
		entries = append(entries, entry{absPath: path, relPath: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		fatalf("walk %s: %v", dirPath, err)
	}
	if len(entries) == 0 {
		fatalf("directory %s is empty", dirPath)
	}

	fmt.Printf("Uploading %d files from %s...\n\n", len(entries), dirName)

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()

		mw.WriteField("dir_name", dirName)
		mw.WriteField("description", desc)
		if tags != "" {
			mw.WriteField("tags", tags)
		}

		for _, e := range entries {
			mw.WriteField("path[]", e.relPath)
			part, err := mw.CreateFormFile("file", e.relPath)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			f, err := os.Open(e.absPath)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := io.Copy(part, f); err != nil {
				f.Close()
				pw.CloseWithError(err)
				return
			}
			f.Close()
		}
	}()

	req, err := http.NewRequest("POST", server+"/directories", pr)
	if err != nil {
		fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient().Do(req)
	if err != nil {
		fatalf("upload directory: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	type dirResult struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		FileCount int       `json:"file_count"`
		Files     []devFile `json:"files"`
	}
	var result dirResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fatalf("decode response: %v", err)
	}

	fmt.Printf("✓ directory  %s\n", result.Name)
	fmt.Printf("  id         %s\n", result.ID)
	fmt.Printf("  files      %d uploaded\n\n", result.FileCount)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, f := range result.Files {
		fmt.Fprintf(w, "  %s\t%s\n", f.Path, f.Language)
	}
	w.Flush()
}

// ── pull ───────────────────────────────────────────────────────────────────

func runPull(server string, args []string) {
	if len(args) == 0 {
		fatalf("pull requires a file id or name\n  usage: devbox-cli pull <id|name>")
	}

	file, err := resolevFile(server, args[0])
	if err != nil {
		fatalf("%v", err)
	}

	resp, err := httpClient().Get(server + "/files/" + file.ID)
	if err != nil {
		fatalf("download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	out, err := os.Create(file.FileName)
	if err != nil {
		fatalf("create %s: %v", file.FileName, err)
	}
	defer out.Close()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		fatalf("write: %v", err)
	}

	fmt.Printf("✓ saved  %s  (%s)\n", file.FileName, formatBytes(n))
}

// ── tag ────────────────────────────────────────────────────────────────────

func runTag(server string, args []string) {
	if len(args) == 0 {
		fatalf("tag requires a file id or name and a tag\n  usage: devbox-cli tag <id|name> <tag>")
	}
	fs := flag.NewFlagSet("tag", flag.ExitOnError)
	recursive := fs.Bool("r", false, "tag entire directory")
	fs.Parse(args)

	if *recursive {
		runDirTag(server, args[1], args[2])
		return
	}

	file, err := resolevFile(server, args[0])
	if err != nil {
		fatalf("%v", err)
	}

	tag := args[1]

	body, _ := json.Marshal(map[string][]string{"tags": {tag}})
	resp, err := httpClient().Post(server+"/files/"+file.ID+"/tags", "application/json", strings.NewReader(string(body)))
	if err != nil {
		fatalf("tag request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	fmt.Printf("✓ tagged  %s  (%s)\n", file.FileName, tag)
}

func runDirTag(server string, dirName string, tags string) {
	dir, err := resolevDir(server, dirName)
	if err != nil {
		fatalf("%v", err)
	}

	body, _ := json.Marshal(map[string][]string{"tags": strings.Split(tags, ",")})
	resp, err := httpClient().Post(server+"/directories/"+dir.ID+"/tags", "application/json", strings.NewReader(string(body)))
	if err != nil {
		fatalf("tag request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	fmt.Printf("✓ tagged  %s  (%s)\n", dir.Name, tags)
}

// ── untag ──────────────────────────────────────────────────────────────────

func runUntag(server string, args []string) {
	if len(args) == 0 {
		fatalf("untag requires a file id or name and a tag\n  usage: devbox-cli untag <id|name> <tag>")
	}

	file, err := resolevFile(server, args[0])
	if err != nil {
		fatalf("%v", err)
	}

	tag := args[1]

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/files/%s/tags/%s", server, file.ID, tag), nil)
	if err != nil {
		fatalf("untag request: %v", err)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		fatalf("untag request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		fatalf("server error %d", resp.StatusCode)
	}

	fmt.Printf("✓ untagged  %s  (%s)\n", file.FileName, tag)
}

// ── update ─────────────────────────────────────────────────────────────────

func runUpdate(server string, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	recursive := fs.Bool("r", false, "sync entire directory")
	message := fs.String("m", "", "version message")
	fs.Parse(args)

	if fs.NArg() < 2 {
		fatalf("usage: devbox-cli update [-r] [-m message] <id|name> <file|dir>")
	}

	target := fs.Arg(0)
	localPath := fs.Arg(1)

	if *recursive {
		runUpdateDir(server, target, localPath, *message)
	} else {
		runUpdateFile(server, target, localPath, *message)
	}
}

func runUpdateFile(server, target, localPath, message string) {
	file, err := resolevFile(server, target)
	if err != nil {
		fatalf("%v", err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		fatalf("open %s: %v", localPath, err)
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()
		part, _ := mw.CreateFormFile("file", filepath.Base(localPath))
		io.Copy(part, f)
		if message != "" {
			mw.WriteField("message", message)
		}
	}()

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/files/%s/versions", server, file.ID), pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient().Do(req)
	if err != nil {
		fatalf("update request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Result string  `json:"result"`
		File   devFile `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fatalf("decode response: %v", err)
	}

	switch result.Result {
	case "unchanged":
		fmt.Printf("~ unchanged  %s  (sha256 matches, no new version created)\n", file.FileName)
	case "updated":
		fmt.Printf("✓ updated    %s  → v%d\n", file.FileName, result.File.Version)
		if message != "" {
			fmt.Printf("  message    %s\n", message)
		}
	}
}

func runUpdateDir(server, target, localPath, message string) {
	type devDir struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var dirs []devDir
	getJSON(server+"/directories", &dirs)

	var dirID string
	for _, d := range dirs {
		if d.ID == target || strings.HasPrefix(d.ID, target) || d.Name == target {
			dirID = d.ID
			break
		}
	}
	if dirID == "" {
		fatalf("no directory matching %q", target)
	}

	fi, err := os.Stat(localPath)
	if err != nil {
		fatalf("stat %s: %v", localPath, err)
	}
	if !fi.IsDir() {
		fatalf("%s is not a directory", localPath)
	}

	type entry struct{ abs, rel string }
	var entries []entry
	filepath.Walk(localPath, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(localPath, p)
		entries = append(entries, entry{abs: p, rel: filepath.ToSlash(rel)})
		return nil
	})

	if len(entries) == 0 {
		fatalf("directory %s is empty", localPath)
	}

	fmt.Printf("Syncing %d files to directory %s...\n\n", len(entries), target)

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()
		if message != "" {
			mw.WriteField("message", message)
		}
		for _, e := range entries {
			mw.WriteField("path[]", e.rel)
			part, _ := mw.CreateFormFile("file", e.rel)
			f, err := os.Open(e.abs)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			io.Copy(part, f)
			f.Close()
		}
	}()

	req, _ := http.NewRequest("PUT", server+"/directories/"+dirID, pr)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := httpClient().Do(req)
	if err != nil {
		fatalf("update request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fatalf("server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Updated   []string `json:"updated"`
		Unchanged []string `json:"unchanged"`
		Added     []string `json:"added"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fatalf("decode response: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range result.Updated {
		fmt.Fprintf(w, "  ✓ updated\t%s\n", p)
	}
	for _, p := range result.Added {
		fmt.Fprintf(w, "  + added\t%s\n", p)
	}
	for _, p := range result.Unchanged {
		fmt.Fprintf(w, "  ~ unchanged\t%s\n", p)
	}
	w.Flush()
	fmt.Printf("\n%d updated, %d added, %d unchanged\n",
		len(result.Updated), len(result.Added), len(result.Unchanged))
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

	file, err := resolevFile(server, fs.Arg(0))
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
		server+"/files/"+file.ID+"/deliver",
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

	fmt.Printf("Delivering %s:\n\n", file.FileName)
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

func resolevFile(server, query string) (file *devFile, err error) {
	var files []devFile
	getJSON(server+"/files", &files)

	// Exact UUID → UUID prefix → exact name → name prefix
	for _, f := range files {
		if f.ID == query || strings.HasPrefix(f.ID, query) || f.FileName == query || f.Path == query {
			return &f, nil
		}
	}

	return nil, fmt.Errorf("no file matching %q", query)
}

func resolevDir(server, query string) (dir *devDir, err error) {
	var dirs []devDir
	getJSON(server+"/directories", &dirs)

	// Exact UUID → UUID prefix → exact name → name prefix
	for _, d := range dirs {
		if d.ID == query || strings.HasPrefix(d.ID, query) || d.Name == query {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("no directory matching %q", query)
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

// multiFlag is a flag.Value that accumulates repeated --to flags.
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
