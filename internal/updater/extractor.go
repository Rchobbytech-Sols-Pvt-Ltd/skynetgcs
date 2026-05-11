package updater

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Extract(zipPath, destDir string) error {
	return ExtractContext(context.Background(), zipPath, destDir)
}

func ExtractContext(ctx context.Context, zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractEntry(ctx, f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(ctx context.Context, f *zip.File, destDir string) error {
	target := filepath.Join(destDir, f.Name)

	if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) &&
		target != filepath.Clean(destDir) {
		return fmt.Errorf("zip slip detected: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, f.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(out, &contextReader{ctx: ctx, reader: rc})
	return err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
