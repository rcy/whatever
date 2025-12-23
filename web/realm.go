package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/rcy/whatever/commands"
)

const realmCookieName = "whatever.realmID"

const realmContextKey = "realm"

func realmFromRequest(r *http.Request) uuid.UUID {
	return r.Context().Value(realmContextKey).(uuid.UUID)
}

func (s *webservice) realmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the realm stored in the cookie, ensure it is
		// valid, or create a new realm if necessary.

		var realmID uuid.UUID
		cookie, err := r.Cookie(realmCookieName)
		if err == nil { // no error
			realmID, err := uuid.Parse(cookie.Value)

			realm, err := s.app.Realms.FindByID(realmID)
			if err != nil {
				realmID = uuid.Nil
			} else {
				realmID = realm.ID
			}
		}

		// no realm from cookie, see if any realm at all exists and try to use that
		if realmID == uuid.Nil {
			realm, err := s.app.Realms.FindOldest()
			if err != nil {
				realmID = uuid.Nil
			} else {
				realmID = realm.ID
			}
		}

		// still no realm, create a new one
		if realmID == uuid.Nil {
			realmID = uuid.New()
			err := s.app.Commander.Send(commands.CreateRealm{RealmID: realmID, Name: "default"})
			if err != nil {
				http.Error(w, fmt.Sprintf("CreateRealm: %s", err), http.StatusInternalServerError)
				return
			}
		}

		// set/refresh cookie
		s.setRealmCookie(w, r, realmID.String())
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
		Secure:   s.secureCookie(r),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
}
