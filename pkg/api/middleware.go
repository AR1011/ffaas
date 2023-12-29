package api

import (
	"fmt"
	"net/http"
	"time"
)

// tempory middleware which injects time taken to process request
// maybe it should be kept though ???

// essentially it wraps a handlerFunc and passes an overriden ResponseWriter,
// which sets a time taken header when status code or body is written

type timedResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	StartTime  time.Time
	TimeTaken  time.Duration
}

// override Write header to inject time taken
// not all requests will have a body, so we need to set the time taken here
func (rw *timedResponseWriter) WriteHeader(statusCode int) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error occurred in WriteHeader:", err)
			rw.StatusCode = 0
			rw.TimeTaken = 0
		}
	}()

	rw.StatusCode = statusCode
	if rw.TimeTaken == 0 {
		rw.TimeTaken = time.Since(rw.StartTime)
		rw.Header().Set("X-Response-Time", rw.TimeTaken.String())
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

// override Write to inject time taken
func (rw *timedResponseWriter) Write(b []byte) (int, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error occurred in Write:", err)
		}
	}()

	if rw.TimeTaken == 0 {
		rw.TimeTaken = time.Since(rw.StartTime)
		rw.Header().Set("X-Response-Time", rw.TimeTaken.String())
	}

	return rw.ResponseWriter.Write(b)
}

func UseTimerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := &timedResponseWriter{
			ResponseWriter: w,
			StartTime:      time.Now(),
		}
		next.ServeHTTP(rw, r)
	}
}
