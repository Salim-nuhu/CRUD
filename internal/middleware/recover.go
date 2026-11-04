package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/Salim-nuhu/CRUD/internal/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Go uses this special value to deliberately abort a
				// response. It is not a bug, so let it continue upward.
				if err == http.ErrAbortHandler {
					panic(err)
				}
				log.Printf("panic: %v\n%s", err, debug.Stack())
				response.Error(w, http.StatusInternalServerError, "something went wrong")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
