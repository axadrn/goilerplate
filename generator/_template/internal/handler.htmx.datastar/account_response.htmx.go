package handler

import (
	"github.com/a-h/templ"
	"net/http"
)

func accountToast(w http.ResponseWriter, r *http.Request, component templ.Component) {
	renderOOB(w, r, component, "beforeend:#toast-container")
}
