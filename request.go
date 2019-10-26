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

func getAPIKey(r *http.Request) string {
	accessToken := fmt.Sprintf("%s", r.Header["X-Api-Key"])
	accessToken = strings.Trim(accessToken, "[")
	accessToken = strings.Trim(accessToken, "]")

	return accessToken
}

func isValidRequest(w http.ResponseWriter, r *http.Request) bool {
	valid := false

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

	return valid
}
