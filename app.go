package serve

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
)

func App(fsys fs.FS, indexFile string, opts ...Option) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// TODO: configurable handle logging

		Handle(w, r, func() (int64, error) {
			if r.Method != http.MethodGet {
				return 0, ErrMethodNotAllowed
			}
			if f, err := fsys.Open(r.URL.Path); err != nil {
				if !os.IsNotExist(errors.Unwrap(err)) {
					return 0, err
				}
			} else {
				defer f.Close()

				if info, err := f.Stat(); err != nil {
					return 0, err
				} else if !info.IsDir() {
					return File(w, r, f, opts...)
				}
			}
			if indexFile != "" {
				return FSFile(w, r, fsys, indexFile, opts...)
			} else {
				return 0, fs.ErrNotExist
			}
		})
	})
}
