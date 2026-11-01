package handler

import (
	"github.com/a-h/templ"
	"net/http"
)

func accountToast(w http.ResponseWriter, r *http.Request, component templ.Component) {
	appendElement(w, r, component, "#toast-container")
}
