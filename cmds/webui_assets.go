package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var webuiStaticDirPath = filepath.Join("internal", "app", "httpserver", "static")

func defaultEnsureWebUIAssets() error {
	indexPath := filepath.Join(webuiStaticDirPath, "index.html")
	indexInfo, err := os.Stat(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("webui static assets missing: %s (run: cd webui && pnpm build)", indexPath)
		}
		return fmt.Errorf("check webui static index failed: %w", err)
	}
	if indexInfo.IsDir() {
		return fmt.Errorf("webui static index is a directory: %s", indexPath)
	}

	assetCandidates, err := filepath.Glob(filepath.Join(webuiStaticDirPath, "assets", "*.js"))
	if err != nil {
		return fmt.Errorf("scan webui static assets failed: %w", err)
	}
	if len(assetCandidates) == 0 {
		return fmt.Errorf("webui static bundle missing in %s (run: cd webui && pnpm build)", filepath.Join(webuiStaticDirPath, "assets"))
	}
	return nil
}
