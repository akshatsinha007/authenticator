package middleware

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
)

// Authorizer is a middleware for authorization
func Authorizer(sessionManager *SessionManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			//var users []string
			cookie, _ := r.Cookie("argocd.token")
			token := ""
			if cookie != nil {
				token = cookie.Value
				r.Header.Set("token", token)
			}
			if token == "" && cookie == nil {
				token = r.Header.Get("token")
				//if cookie == nil && len(token) != 0 {
				//	http.SetCookie(w, &http.Cookie{Name: "argocd.token", Value: token, Path: "/"})
				//}
			}
			//users = append(users, "anonymous")
			authEnabled := true
			pass := false
			config := GetConfig()
			authEnabled = config.AuthEnabled

			if token != "" && authEnabled && !contains(r.URL.Path) {
				_, err := sessionManager.VerifyToken(token)
				if err != nil {
					log.Printf("Error verifying token: %+v\n", err)
					http.SetCookie(w, &http.Cookie{Name: "argocd.token", Value: token, Path: "/", MaxAge: -1})
					writeResponse(http.StatusUnauthorized, "Unauthorized", w, err)
					return
				}
				pass = true
				//TODO - we also can set user info in session (to avoid fetch it for all create n active)
			}
			if pass {
				next.ServeHTTP(w, r)
			} else if contains(r.URL.Path) {
				next.ServeHTTP(w, r)
			} else if token == "" {
				writeResponse(http.StatusUnauthorized, "UN-AUTHENTICATED", w, fmt.Errorf("unauthenticated"))
				return
			} else {
				writeResponse(http.StatusForbidden, "FORBIDDEN", w, fmt.Errorf("unauthorized"))
				return
			}
		}

		return http.HandlerFunc(fn)
	}
}

func contains(url string) bool {
	urls := []string{
		"/auth/login",
		"/auth/callback",

		"/",
	}
	for _, a := range urls {
		if a == url {
			return true
		}
	}
	prefixUrls := []string{
		"/api/dex/",
	}
	for _, a := range prefixUrls {
		if strings.Contains(url, a) {
			return true
		}
	}
	return false
}

func writeResponse(status int, message string, w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	type Response struct {
		Code   int         `json:"code,omitempty"`
		Status string      `json:"status,omitempty"`
		Result interface{} `json:"result,omitempty"`
	}
	response := Response{}
	response.Code = status
	response.Result = message
	b, err := json.Marshal(response)
	if err != nil {
		b = []byte("OK")
		log.Error("Unexpected error in apiError", "err", err)
	}
	_, err = w.Write(b)
	if err != nil {
		log.Error("error", "err", err)
	}
}