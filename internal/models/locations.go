package models

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/wf-pro-dev/devbox/internal/db"
	"github.com/wf-pro-dev/devbox/types"
	"github.com/wf-pro-dev/tailkit"
	tailkitTypes "github.com/wf-pro-dev/tailkit/types"
)

func GetLocations(ctx context.Context, srv *tailkit.Server) ([]types.Location, error) {
	peers, err := tailkit.AllPeers(ctx, srv)
	if err != nil {
		return nil, err
	}
	var validPeers []tailkitTypes.Peer
	for _, peer := range peers {
		if peer.Tailkit == nil {
			continue
		}
		validPeers = append(validPeers, peer)
	}
	locations := []types.Location{}

	fleet := tailkit.Nodes(srv, validPeers)

	configs, errs := fleet.Files().Config(ctx)
	if configs == nil {
		return nil, fmt.Errorf("failed to get file configs: %w", errs)
	}
	for hostname, config := range configs {
		locations = append(locations, types.Location{
			Hostname: hostname,
			Paths:    config.Paths,
		})
	}

	return locations, nil
}

func GetLocationDirs(ctx context.Context, srv *tailkit.Server, hostname string) ([]types.DirListing, error) {

	node := tailkit.Node(srv, hostname)
	if node == nil {
		return nil, fmt.Errorf("node not found: %s", hostname)
	}

	config, err := node.Files().Config(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get file config: %w", err)
	}

	location := types.Location{
		Hostname: hostname,
		Paths:    config.Paths,
	}

	var dirListings []types.DirListing
	for _, path := range location.Paths {
		entries, err := node.Files().List(ctx, path.Dir)
		if err != nil {
			return nil, fmt.Errorf("failed to get directory list: %w", err)
		}

		var dirListing types.DirListing
		dirListing.Entries = make([]types.DirEntry, 0, len(entries))
		for _, entry := range entries {
			prefix := fmt.Sprintf("%s%s", path.Dir, entry.Name)
			if entry.IsDir {
				prefix += "/"
			}
			e := types.DirEntry{
				Name:   entry.Name,
				IsDir:  entry.IsDir,
				Prefix: prefix,
				Stats: &types.DirectoryStats{
					TotalSize: entry.Size,
				},
			}

			dirListing.Entries = append(dirListing.Entries, e)
		}

		dirListing.Prefix = path.Dir
		dirListings = append(dirListings, dirListing)
	}

	return dirListings, nil
}

func GetLocationDir(ctx context.Context, srv *tailkit.Server, hostname, prefix string) (types.DirListing, error) {
	node := tailkit.Node(srv, hostname)
	if node == nil {
		return types.DirListing{}, fmt.Errorf("node not found: %s", hostname)
	}

	config, err := node.Files().Config(ctx)
	if err != nil {
		return types.DirListing{}, fmt.Errorf("failed to get file config: %w", err)
	}
	if _, _, ok := config.MatchPathRule(prefix); !ok {
		return types.DirListing{}, fmt.Errorf("path not shared: %s", prefix)
	}

	entries, err := node.Files().List(ctx, prefix)
	if err != nil {
		return types.DirListing{}, fmt.Errorf("failed to get directory list: %w", err)
	}

	listing := types.DirListing{
		Prefix:  prefix,
		Entries: make([]types.DirEntry, 0, len(entries)),
	}
	for _, entry := range entries {
		fullPath := joinRemotePath(prefix, entry.Name, entry.IsDir)
		dirEntry := types.DirEntry{
			Name:  entry.Name,
			IsDir: entry.IsDir,
			Stats: &types.DirectoryStats{
				TotalSize:       entry.Size,
				LatestUpdatedAt: entry.ModTime.Format(time.RFC3339),
			},
		}
		if entry.IsDir {
			dirEntry.Prefix = fullPath
		} else {
			dirEntry.FileCount = 1
			dirEntry.File = &types.File{
				File:     dbFileForRemote(fullPath, entry.Name, entry.Size, entry.ModTime),
				Source:   "remote",
				Hostname: hostname,
			}
		}
		listing.Entries = append(listing.Entries, dirEntry)
	}
	log.Printf("prefix: %s, listing: %+v", prefix, len(listing.Entries))

	return listing, nil
}

func ReadLocationFile(ctx context.Context, srv *tailkit.Server, hostname, path string) (*types.File, string, error) {
	node := tailkit.Node(srv, hostname)
	if node == nil {
		return nil, "", fmt.Errorf("node not found: %s", hostname)
	}

	config, err := node.Files().Config(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file config: %w", err)
	}
	rule, _, ok := config.MatchPathRule(path)
	if !ok || !rule.Permits("read") || !rule.Share {
		return nil, "", fmt.Errorf("path not shared: %s", path)
	}

	content, err := node.Files().Read(ctx, path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}

	remoteFile := &types.File{
		File:     dbFileForRemote(path, filepath.Base(path), int64(len(content)), time.Time{}),
		Source:   "remote",
		Hostname: hostname,
	}
	remoteFile.Language = detectLanguage(path)
	return remoteFile, content, nil
}

func joinRemotePath(prefix, name string, isDir bool) string {
	base := prefix
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	full := base + name
	if isDir && !strings.HasSuffix(full, "/") {
		full += "/"
	}
	return full
}

func dbFileForRemote(path, name string, size int64, modTime time.Time) db.File {
	ts := ""
	if !modTime.IsZero() {
		ts = modTime.UTC().Format(time.RFC3339)
	}
	return db.File{
		Path:      path,
		FileName:  name,
		Language:  detectLanguage(path),
		Size:      size,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".yaml", ".yml":
		return "yaml"
	case ".sql":
		return "sql"
	case ".md":
		return "markdown"
	case ".sh":
		return "bash"
	case ".service":
		return "systemd"
	case ".ini", ".conf":
		return "ini"
	case ".dockerfile":
		return "dockerfile"
	default:
		if strings.EqualFold(filepath.Base(path), "dockerfile") {
			return "dockerfile"
		}
		return "text"
	}
}
