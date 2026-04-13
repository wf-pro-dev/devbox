package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	"github.com/wf-pro-dev/devbox/internal/progress"
)

var SERVER_URL string

func Client() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func ProgressClient(p *progress.Progress) *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				return &progress.ConnReader{
					Conn:    conn,
					OnWrite: p.Increment,
				}, nil
			},
		},
	}
}

func Server() string {

	if SERVER_URL == "" {
		SERVER_URL = os.Getenv("DEVBOX_SERVER")
	}
	return SERVER_URL
}

func SetServer(server string) {
	SERVER_URL = server
}

// ── Request helpers ───────────────────────────────────────────────────────────

func GetJSON(url string) (*http.Response, error) {
	resp, err := Client().Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	return resp, nil
}

func Del(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("DELETE %s: %w", url, err)
	}
	return resp, nil
}

func PostJSON(url string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := Client().Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	return resp, nil
}

func PatchJSON(url string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("PATCH %s: %w", url, err)
	}
	return resp, nil
}

// uploadFile sends a multipart POST with one file and optional string fields.
// fields is a map of form field name → value.
func UploadFile(url, localPath string, fields map[string]string) (*http.Response, error) {

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}

	multipartWriter, err := mw.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return nil, err
	}

	fileContent, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", localPath, err)
	}
	defer fileContent.Close()

	ctx := context.Background()
	progresManager := progress.GetManager(ctx)

	fileInfo, err := fileContent.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", localPath, err)
	}
	totalSize := fileInfo.Size()

	if _, err := io.Copy(multipartWriter, fileContent); err != nil {
		return nil, err
	}

	mw.Close()

	connID := uuid.New().String()
	connProgress := progresManager.Create(connID, totalSize)
	defer progresManager.Remove(connID)

	connBar := progressbar.DefaultBytes(totalSize, "Uploading file")

	connProgress.OnProgress(func(progress *progress.Progress) {
		snapshot := progress.Snapshot()

		if snapshot.Current >= totalSize {
			connBar.Finish()
		} else {
			connBar.Set64(snapshot.Current)
		}
	})

	resp, err := ProgressClient(connProgress).Post(url, mw.FormDataContentType(), &buf)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("POST %s: %s", url, resp.Status)
	}
	return resp, nil
}

// uploadFileUpdate sends a multipart PUT to update file content.
func UploadFileUpdate(url, localPath, message string) (*http.Response, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	fileName := filepath.Base(localPath)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	mw.WriteField("file_name", fileName)

	if message != "" {
		mw.WriteField("message", message)
	}

	part, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPut, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("PUT %s: %w", url, err)
	}
	return resp, nil
}

// uploadDir sends a multipart POST with multiple files for a collection.
// files is a slice of {localPath, relativePath} pairs.
type DirFile struct {
	LocalPath string
	RelPath   string
}

func UploadDirFiles(url string, fields map[string]string, files []DirFile) (*http.Response, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range fields {
		mw.WriteField(k, v)
	}

	for _, df := range files {
		// path[] field for the relative path
		mw.WriteField("path[]", df.RelPath)
		mw.WriteField("local_path[]", df.LocalPath)

		// file part
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(df.LocalPath)))
		h.Set("Content-Type", "application/octet-stream")
		part, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		f, err := os.Open(df.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", df.LocalPath, err)
		}
		_, copyErr := io.Copy(part, f)
		f.Close()
		if copyErr != nil {
			return nil, copyErr
		}
	}
	mw.Close()

	resp, err := Client().Post(url, mw.FormDataContentType(), &buf)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	return resp, nil
}

// syncDirFiles sends a multipart PUT to sync files to an existing collection.
func SyncDirFiles(url string, fields map[string]string, files []DirFile) (*http.Response, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	for k, v := range fields {
		mw.WriteField(k, v)
	}

	for _, df := range files {
		mw.WriteField("path[]", df.RelPath)
		mw.WriteField("local_path[]", df.LocalPath)

		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(df.LocalPath)))
		h.Set("Content-Type", "application/octet-stream")
		part, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		f, err := os.Open(df.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", df.LocalPath, err)
		}
		_, copyErr := io.Copy(part, f)
		f.Close()
		if copyErr != nil {
			return nil, copyErr
		}
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPut, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("PUT %s: %w", url, err)
	}
	return resp, nil
}

// ── Response helpers ──────────────────────────────────────────────────────────

// decode reads JSON from a response body into v. Non-2xx responses are
// returned as an error using the server's "error" field if present.
func Decode(resp *http.Response, v any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract server error message.
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error != "" {
			return fmt.Errorf("server error %d: %s", resp.StatusCode, e.Error)
		}
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(body, v)
}

// checkNoContent asserts a 204 and drains the body.
func CheckNoContent(resp *http.Response) error {
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ── Walk helpers ──────────────────────────────────────────────────────────────

// walkDir returns all regular files under root as (localPath, relPath) pairs.
func WalkDir(root string) ([]DirFile, error) {
	var files []DirFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, DirFile{
			LocalPath: path,
			RelPath:   filepath.ToSlash(rel),
		})
		return nil
	})
	return files, err
}
