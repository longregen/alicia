package handlers

import (
	"net/http"

	"github.com/longregen/alicia/api/config"
)

type VpnHandler struct {
	cfg *config.Config
}

func NewVpnHandler(cfg *config.Config) *VpnHandler {
	return &VpnHandler{cfg: cfg}
}

// GetAuthKey handles POST /vpn/auth-key
// Returns the pre-configured Headscale pre-auth key and server URL.
func (h *VpnHandler) GetAuthKey(w http.ResponseWriter, r *http.Request) {
	hsCfg := h.cfg.Headscale

	respondJSON(w, map[string]string{
		"server_url": hsCfg.URL,
		"auth_key":   hsCfg.PreAuthKey,
	}, http.StatusOK)
}
