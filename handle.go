package serve

import (
	"log"
	"net/http"
	"time"
)

func Handle(w http.ResponseWriter, r *http.Request, fn func() (int64, error)) {
	startTime := time.Now()

	// TODO: improve size logging (both incoming and outgoing)

	logUrl := r.URL.String()
	if len(logUrl) > 120 {
		logUrl = logUrl[:117] + "..."
	}
	n, err := fn()
	if n == 0 && err != nil {
		Error(w, err)
		log.Printf("[%d] %s %s (%v) ERROR: %v", statusCode(err), r.Method, logUrl, time.Since(startTime), err)
	} else {
		log.Printf("[200] %s %s (%d bytes, %v)", r.Method, logUrl, n, time.Since(startTime))
	}
}
