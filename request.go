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

func isValidRequest(w http.ResponseWriter, r *http.Request) bool {

	valid := false

	for _, c := range r.Cookies() {
		if "token" == c.Name {
			// Check whether presented token is valid
			var jwtToken = JWTToken{}
			jwtToken.LookupFromString(c.Value)
			if jwtToken.IsValid() {
				valid = true
				break
			} else {
				fmt.Println("Token invalid")
			}
		}

	}

	// Iterate over keylist of headers and make sure
	// our required headers are present.
	for k := range r.Header {
		if k == "X-Api-Key" {
			valid = true
		}
	}

	if !valid {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
	}

	//Piggyback off of request valid check for request counter (this is bad)

	if valid {
		requestCounter.Inc()
	} else {
		unauthorizedRequestCounter.Inc()
	}

	return valid
}
