package middleware

import "net/http"

func redirect(w http.ResponseWriter, r *http.Request, location string, status int) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, location, status)
}
