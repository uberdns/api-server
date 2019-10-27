package main

import (
	"fmt"
	"net/http"
	"strings"
)

// requestRecord -- struct for storing information regarding a received api request
type requestRecord struct {
	Name      string
	IPAddress string
}

func (ri *RequestCounter) Inc() {
	ri.mu.Lock()
	ri.Total++
	ri.mu.Unlock()

	return
}

func (ri *RequestCounter) Count() int {
	var v int
	ri.mu.Lock()
	v = ri.Total
	ri.mu.Unlock()

	return v
}

func getAPIKey(r *http.Request) string {
	accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
	accessToken = strings.Trim(accessToken, "[")
	accessToken = strings.Trim(accessToken, "]")

	return accessToken
}

func RequestHasAPIKey(r *http.Request) bool {
	for k := range r.Header {
		if k == "X-Api-Key" {
			return true
		}
	}

	return false
}

func RequestHasBearerToken(r *http.Request) bool {
	for _, c := range r.Cookies() {
		if "token" == c.Name {
			return true
		}
	}

	for k := range r.Header {
		if k == "Authorization" {
			return true
		}
	}

	return false
}

func getUserFromRequest(r *http.Request) User {
	var user User
	if RequestHasAPIKey(r) {
		accessToken := getAPIKey(r)
		user.LookupFromAPIKey(accessToken)
	}

	if RequestHasBearerToken(r) {
		for _, c := range r.Cookies() {
			if "token" == c.Name {
				var jwtToken = JWTToken{}
				jwtToken.LookupFromString(c.Value)
				if jwtToken.IsValid() {
					user.ID = jwtToken.UserID
					user.LookupFromID()
					break
				}
			}
		}

		for k := range r.Header {
			if k == "Authorization" {
				bearerToken := strings.Split(r.Header.Get(k), "Bearer")[1]
				bearerToken = strings.Trim(bearerToken, " ")
				var jwtToken = JWTToken{}
				jwtToken.LookupFromString(bearerToken)
				if jwtToken.IsValid() {
					user.ID = jwtToken.UserID
					user.LookupFromID()
					break
				}
			}
		}
	}

	return user
}

func requestMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for a request body
		//if r.ContentLength == 0 {
		//	w.WriteHeader(http.StatusBadRequest)
		//	return
		//}

		// this is required with cors site checks....once everything runs under the same domain we should remove this
		switch r.Method {
		case "OPTIONS":
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			return
		case "GET":
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		case "POST":
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Add("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		}

		// Check for valid Bearer token as a cookie stored as token
		for _, c := range r.Cookies() {
			if "token" == c.Name {
				// Check whether presented token is valid
				jwtToken := JWTToken{}
				jwtToken.LookupFromString(c.Value)
				if !jwtToken.IsValid() {
					unauthorizedRequestCounter.Inc()
					w.WriteHeader(http.StatusUnauthorized)
					//w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Increment authorized request counter
				requestCounter.Inc()
				next.ServeHTTP(w, r)
			}
		}

		// check for valid bearer token as a header (curl/api)
		for k := range r.Header {
			if k == "Authorization" {
				bearerToken := strings.Split(r.Header.Get(k), "Bearer")[1]
				bearerToken = strings.Trim(bearerToken, " ")
				// chrome sometimes sends null bearers lol
				if bearerToken == "null" {
					unauthorizedRequestCounter.Inc()
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				jwtToken := JWTToken{}
				jwtToken.LookupFromString(bearerToken)
				if !jwtToken.IsValid() {
					unauthorizedRequestCounter.Inc()
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Hey its a valid jwt token!
				requestCounter.Inc()
				next.ServeHTTP(w, r)
				return
			}
		}

		fmt.Println("No bearer token")
		// No bearer token presented, no worries - we can check api key!
		// Iterate over keylist of headers and make sure
		// our required headers are present.
		headerValid := RequestHasAPIKey(r)
		for k := range r.Header {
			if k == "X-Api-Key" {
				headerValid = true
			}
		}

		if !headerValid {
			w.WriteHeader(http.StatusBadRequest)
			unauthorizedRequestCounter.Inc()
			return
		}
	})
}
