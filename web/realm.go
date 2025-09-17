package web

import (
	"context"
	"fmt"
	"net/http"
)

const realmCookieName = "whatever.realmID"

const realmContextKey = "realm"

func realmFromRequest(r *http.Request) string {
	return r.Context().Value(realmContextKey).(string)
}

func (s *webservice) realmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the realm stored in the cookie, ensure it is
		// valid, or create a new realm if necessary.

		var realmID string
		cookie, err := r.Cookie(realmCookieName)
		if err == nil { // no error
			realm, err := s.app.Realms().FindByID(cookie.Value)
			if err != nil {
				realmID = ""
			} else {
				realmID = realm.ID
			}
		}

		// no realm from cookie, see if any realm at all exists and try to use that
		if realmID == "" {
			realm, err := s.app.Realms().FindOldest()
			if err != nil {
				realmID = ""
			} else {
				realmID = realm.ID
			}
		}

		// still no realm, create a new one
		if realmID == "" {
			aggID, err := s.app.Commands().CreateRealm("personal")
			if err != nil {
				http.Error(w, fmt.Sprintf("CreateRealm: %s", err), http.StatusInternalServerError)
				return
			}

			// since the realm projection is syncronously materialized in the command handler, we can do this:
			realmID = aggID
		}

		// set/refresh cookie
		s.setRealmCookie(w, r, realmID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), realmContextKey, realmID)))
	})
}

func (s *webservice) setRealmCookie(w http.ResponseWriter, r *http.Request, value string) {
	cookie := http.Cookie{
		Name:     realmCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: true,
		Secure:   r.URL.Scheme == "https",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}
