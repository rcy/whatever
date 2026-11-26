package web

import (
	"net/http"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/workers/bskysync"
)

func (s *webservice) bskyConnect(w http.ResponseWriter, r *http.Request) {
	handle := r.FormValue("handle")
	appPassword := r.FormValue("app_password")
	if handle == "" || appPassword == "" {
		http.Error(w, "handle and app password are required", http.StatusBadRequest)
		return
	}

	userID := getUserInfo(r).Id
	err := s.app.SaveBlueskyCredentials(userID, app.BlueskyCredentials{
		Handle:      handle,
		AppPassword: appPassword,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *webservice) bskySync(w http.ResponseWriter, r *http.Request) {
	userID := getUserInfo(r).Id
	err := s.app.River.Insert(r.Context(), bskysync.JobArgs{UserID: userID}, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
