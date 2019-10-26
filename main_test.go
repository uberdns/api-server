package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var testRecord = Record{
	ID:       1,
	Name:     "test",
	IP:       "127.0.0.1",
	TTL:      30,
	DomainID: 1,
	OwnerID:  1,
}

// TestIsValidRequest - tests using indexView
func TestIsValidRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)

	req.Header.Add("X-API-Key", "test123")

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexView)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Welcome home"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v wanted %v",
			rr.Body.String(), expected)
	}
}

func TestIndexView(t *testing.T) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)

	req.Header.Add("X-API-Key", "test123")

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexView)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "Welcome home"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v wanted %v",
			rr.Body.String(), expected)
	}
}

func TestIndexView_MissingAPIKey(t *testing.T) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexView)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())

}
