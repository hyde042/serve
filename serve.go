package serve

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	IndexFileName = "index.html"

	DefaultRangeBufferSize = 1 << 22 // 4 MiB

	MimeHeader        = "Content-Type"
	SizeHeader        = "Content-Length"
	ModTimeHeader     = "Last-Modified"
	DispositionHeader = "Content-Disposition"

	year = time.Hour * 24 * 365
)

var (
	JSONMime = Mime("application/json; charset=utf-8")

	rangeParser = regexp.MustCompile(`bytes=(\d+)-(\d+)?`)
)

type config struct {
	size        int64
	mime        string
	modTime     time.Time
	maxAge      time.Duration
	immutable   bool
	disposition string
	compress    bool
}

type Option func(c *config)

func Size(n int64) Option           { return func(c *config) { c.size = n } }
func SizeOf(data []byte) Option     { return func(c *config) { c.size = int64(len(data)) } }
func Mime(s string) Option          { return func(c *config) { c.mime = s } }
func ModTime(t time.Time) Option    { return func(c *config) { c.modTime = t } }
func MaxAge(d time.Duration) Option { return func(c *config) { c.maxAge = d } }
func Immutable(b bool) Option       { return func(c *config) { c.immutable = b } }
func Compress(b bool) Option        { return func(c *config) { c.compress = b } }

func Attachment(name string) Option {
	return func(c *config) {
		if name == "" {
			c.disposition = ""
		} else {
			c.disposition = fmt.Sprintf(`attachment; filename="%s"`, name)
		}
	}
}

func JSON(w http.ResponseWriter, r *http.Request, v interface{}, opts ...Option) (int64, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	return Reader(w, r, bytes.NewReader(data), append([]Option{SizeOf(data), JSONMime}, opts...)...)
}

func File(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string, opts ...Option) (int64, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return Reader(w, r, f, append([]Option{
		Size(info.Size()),
		Mime(mime.TypeByExtension(path.Ext(name))),
	}, opts...)...)
}

func Bytes(w http.ResponseWriter, r *http.Request, data []byte, opts ...Option) (int64, error) {
	return Reader(w, r, bytes.NewReader(data), append(opts, SizeOf(data))...)
}

func Reader(rw http.ResponseWriter, req *http.Request, r io.Reader, opts ...Option) (int64, error) {
	var c config
	for _, opt := range opts {
		opt(&c)
	}

	// TODO: support content-type detection?

	if c.mime != "" {
		rw.Header().Set(MimeHeader, c.mime)
	}
	if c.maxAge > 0 || c.immutable {
		if c.immutable && c.maxAge < year {
			c.maxAge = year
		}
		v := fmt.Sprintf("public, max-age=%d", int(c.maxAge/time.Second))
		if c.immutable {
			v += ", immutable"
		}
		rw.Header().Set("Cache-Control", v)
	}
	if !c.modTime.IsZero() {
		rw.Header().Set(ModTimeHeader, c.modTime.Format(http.TimeFormat))
	}

	// TODO: support range requests with io.Seeker
	// TODO: improve range request spec compilance / deal with misbehaving clients

	if rat, ok := r.(io.ReaderAt); ok {
		rw.Header().Set("Accept-Ranges", "bytes")

		if rh := req.Header.Get("Range"); rh != "" {
			begin, end := parseRangeHeader(rh, c.size)
			buf := make([]byte, end-begin)
			if _, err := rat.ReadAt(buf, begin); err != nil {
				return 0, err
			}
			cr := fmt.Sprintf("bytes %d-%d/%d", begin, end-1, c.size)
			rw.Header().Set("Content-Range", cr)
			rw.Header().Set(SizeHeader, strconv.Itoa(len(buf)))

			rw.WriteHeader(http.StatusPartialContent)
			if req.Method == http.MethodHead {
				return 0, nil
			}
			n, err := rw.Write(buf)
			return int64(n), err
		}
	}
	if c.disposition != "" {
		rw.Header().Set(DispositionHeader, c.disposition)
	}
	w := io.Writer(rw)
	if c.compress {
		rw.Header().Set("Vary", "Content-Encoding")
		if c.size > 1400 && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			c.size = -1
			rw.Header().Set("Content-Encoding", "gzip")
			gw := gzip.NewWriter(w)
			defer gw.Flush()
			w = gw
		}
	}
	if c.size > 0 {
		rw.Header().Set(SizeHeader, strconv.FormatInt(c.size, 10))
	}
	if req.Method == http.MethodHead {
		return 0, nil
	}
	return io.Copy(w, r)
}

func parseRangeHeader(s string, size int64) (int64, int64) {
	var (
		sms      = rangeParser.FindStringSubmatch(s)
		begin, _ = strconv.ParseInt(sms[1], 10, 64)
		end, _   = strconv.ParseInt(sms[2], 10, 64)
	)
	if end <= 0 || end == begin || end-begin > DefaultRangeBufferSize {
		end = begin + DefaultRangeBufferSize
		if end > size {
			end = size
		}
	}
	return begin, end
}
