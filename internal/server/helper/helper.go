package helper

import "net/http"

func IsHTMX(r *http.Request) bool {
	return r.Header.Get("Hx-Request") == "true"
}
