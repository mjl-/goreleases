package goreleases

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

// Fetch downloads, extracts and verifies a Go release represented by file into directory dst.
// After a successful fetch, dst contains a directory "go" with the specified release.
// Directory dst must exist. It must not already contain a "go" subdirectory.
// Only files with filenames ending .tar.gz are supported. So no macOS or Windows.
func Fetch(file File, dst string) error {
	if !strings.HasSuffix(file.Filename, ".tar.gz") {
		return fmt.Errorf("file extension not supported, only .tar.gz supported")
	}

	fi, err := os.Stat(dst)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("dst does not exist")
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("dst is not a directory")
	}
	fi, err = os.Stat(path.Join(dst, "go"))
	if err == nil {
		return fmt.Errorf(`directory "go" already exists`)
	}
	// we assume it's a not-exists error. if it isn't, eg noperm, we'll probably get the same error later on, which is fine.

	dst = path.Clean(dst) + "/"

	url := "https://golang.org/dl/" + file.Filename
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("downloading file: status %d: %s", resp.StatusCode, resp.Status)
	}

	hr := &hashReader{resp.Body, sha256.New()}

	gzr, err := gzip.NewReader(hr)
	if err != nil {
		return fmt.Errorf("gzip reader: %s", err)
	}
	defer gzr.Close()

	success := false
	defer func() {
		if !success {
			os.RemoveAll(path.Join(dst, "go"))
		}
	}()

	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading next header from tar file: %s", err)
		}

		name, err := dstName(dst, h.Name)
		if err != nil {
			return err
		}

		err = store(dst, tr, h, name)
		if err != nil {
			return err
		}
	}

	sum := fmt.Sprintf("%x", hr.h.Sum(nil))
	if sum != file.Sha256 {
		return fmt.Errorf("checksum mismatch, got %x, expected %s", sum, file.Sha256)
	}
	success = true
	return nil
}

type hashReader struct {
	r io.Reader
	h hash.Hash
}

func (hr *hashReader) Read(buf []byte) (n int, err error) {
	n, err = hr.r.Read(buf)
	if n > 0 {
		hr.h.Write(buf[:n])
	}
	return
}

func dstName(dst, name string) (string, error) {
	r := path.Clean(path.Join(dst, name))
	if !strings.HasPrefix(r, dst) {
		return "", fmt.Errorf("bad path %q in archive, resulting in path %q outside dst %q", name, r, dst)
	}
	return r, nil
}

func store(dst string, tr *tar.Reader, h *tar.Header, name string) error {
	os.MkdirAll(path.Dir(name), 0777)

	switch h.Typeflag {
	case tar.TypeReg:
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer func() {
			if f != nil {
				f.Close()
			}
		}()
		lr := io.LimitReader(tr, h.Size)
		n, err := io.Copy(f, lr)
		if err != nil {
			return fmt.Errorf("extracting: %v", err)
		}
		if n != h.Size {
			return fmt.Errorf("extracting %d bytes, expected %d", n, h.Size)
		}
		err = f.Chmod(os.FileMode(h.Mode) & os.ModePerm)
		if err != nil {
			return fmt.Errorf("chmod: %s", err)
		}
		err = f.Close()
		if err != nil {
			return fmt.Errorf("close: %s", err)
		}
		f = nil
		return nil
	case tar.TypeLink:
		linkname, err := dstName(dst, h.Linkname)
		if err != nil {
			return err
		}
		return os.Link(linkname, name)
	case tar.TypeSymlink:
		linkname, err := dstName(dst, h.Linkname)
		if err != nil {
			return err
		}
		return os.Symlink(linkname, name)
	case tar.TypeDir:
		return os.Mkdir(name, 0777)
	case tar.TypeXGlobalHeader, tar.TypeGNUSparse:
		return nil
	}
	return fmt.Errorf("unsupported tar header typeflag %v", h.Typeflag)
}
