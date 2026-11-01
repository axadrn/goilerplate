package api

import (
	"net/http"

	"{{.ModulePath}}/internal/brand"
)

// MetaHandler answers public facts about the project for a frontend.
type MetaHandler struct {
	supportEmail string
}

func NewMetaHandler(supportEmail string) *MetaHandler {
	return &MetaHandler{supportEmail: supportEmail}
}

type metaResponse struct {
	Name         string `json:"name"`
	SupportEmail string `json:"support_email"`
}

// Meta returns the project name and the support address.
func (h *MetaHandler) Meta(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, metaResponse{Name: brand.Name, SupportEmail: h.supportEmail})
}
