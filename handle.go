package serve

import (
	"log"
	"net/http"
	"time"
)

func Handle(w http.ResponseWriter, r *http.Request, fn func() (int64, error)) {
	startTime := time.Now()

	// TODO: improve size logging (both incoming and outgoing)
	// TODO: truncate massive URLs for logging

	n, err := fn()
	if n == 0 && err != nil {
		Error(w, err)
		log.Printf("[%d] %s %s (%v) ERROR: %v", statusCode(err), r.Method, r.URL, time.Since(startTime), err)
	} else {
		log.Printf("[200] %s %s (%d bytes, %v)", r.Method, r.URL, n, time.Since(startTime))
	}
}
