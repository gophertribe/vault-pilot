package api

import (
	"net/http"

	"github.com/mklimuk/vault-pilot/pkg/ai"
	"github.com/mklimuk/vault-pilot/pkg/db"
	"github.com/mklimuk/vault-pilot/pkg/sync"
	"github.com/mklimuk/vault-pilot/pkg/vault"
)

// NewRouter creates a new HTTP router
func NewRouter(repo *db.Repository, aiClient ai.Generator, tmplEngine *vault.TemplateEngine, vaultPath string, gitManager *sync.GitManager) *http.ServeMux {
	mux := http.NewServeMux()

	h := &Handler{
		Repo:       repo,
		AI:         aiClient,
		TmplEngine: tmplEngine,
		VaultPath:  vaultPath,
		Git:        gitManager,
	}

	mux.HandleFunc("POST /inbox", h.HandleCreateInboxItem)
	mux.HandleFunc("GET /projects", h.HandleListProjects)
	mux.HandleFunc("POST /review/weekly", h.HandleGenerateWeeklyReview)
	mux.HandleFunc("POST /automations", h.HandleCreateAutomation)
	mux.HandleFunc("GET /automations", h.HandleListAutomations)
	mux.HandleFunc("PATCH /automations/{id}", h.HandleUpdateAutomation)
	mux.HandleFunc("POST /automations/{id}/run-now", h.HandleRunAutomationNow)

	return mux
}
