package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Create(output string, files []string) error {
	out, err := os.Create(output)
	if err != nil { return err }
	defer out.Close()
	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	cwd, _ := os.Getwd()
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil { return err }
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil { return err }
		rel, err := filepath.Rel(cwd, f)
		if err != nil { rel = filepath.Base(f) }
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil { return err }
		src, err := os.Open(f)
		if err != nil { return err }
		_, err = io.Copy(tw, src)
		src.Close()
		if err != nil { return err }
	}
	return nil
}

func Extract(archivePath, dstDir string) error {
	in, err := os.Open(archivePath)
	if err != nil { return err }
	defer in.Close()
	gr, err := gzip.NewReader(in)
	if err != nil { return err }
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF { break }
		if err != nil { return err }
		p := filepath.Join(dstDir, filepath.FromSlash(sanitize(hdr.Name)))
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(p, hdr.FileInfo().Mode().Perm()); err != nil { return err }
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { return err }
		out, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
		if err != nil { return err }
		if _, err := io.Copy(out, tr); err != nil { out.Close(); return err }
		out.Close()
	}
	return nil
}

func sanitize(name string) string {
	n := filepath.Clean(name)
	n = strings.TrimPrefix(n, "../")
	n = strings.ReplaceAll(n, "..\\", "")
	return n
}
