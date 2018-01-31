package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const InvalidToken = "invalid_token"
const TimeoutErrorQuery = "timeout_query"
const InternalErrorQuery = "fatal_query"
const BadRequestErrorQuery = "bad_request_query"
const BadRequestUnknownErrorQuery = "bad_request_unknown_query"
const InvalidJsonErrorQuery = "invalid_json_query"

type UserRow struct {
	Id     int    `xml:"id"`
	Name   string `xml:"first_name"`
	Age    int    `xml:"age"`
	About  string `xml:"about"`
	Gender string `xml:"gender"`
}

type Users struct {
	List []UserRow `xml:"row"`
}

type TestCaseWithError struct {
	Request       SearchRequest
	URL           string
	AccessToken   string
	ErrorExact    string
	ErrorContains string
}

type TestCase struct {
	Request SearchRequest
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// force timeout error
	if r.FormValue("query") == TimeoutErrorQuery {
		time.Sleep(time.Second * 2)
	}

	// force invalid token
	if r.Header.Get("AccessToken") == InvalidToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// force internal error
	if r.FormValue("query") == InternalErrorQuery {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// force bad request error
	if r.FormValue("query") == BadRequestErrorQuery {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// force bad request with unknown error
	if r.FormValue("query") == BadRequestUnknownErrorQuery {
		resp, _ := json.Marshal(SearchErrorResponse{"UnknownError"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	// return invalid json
	if r.FormValue("query") == InvalidJsonErrorQuery {
		w.Write([]byte("invalid_json"))
		return
	}

	// check order_field
	orderField := r.FormValue("order_field")
	if orderField == "" {
		orderField = "Name"
	}
	if orderField != "Id" && orderField != "Age" && orderField != "Name" {
		resp, _ := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println("cant open file:", err)
		return
	}
	defer xmlFile.Close()

	var data Users
	byteValue, _ := ioutil.ReadAll(xmlFile)
	xml.Unmarshal(byteValue, &data)

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		fmt.Println("cant convert offset to int: ", err)
		return
	}
	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		fmt.Println("cant convert limit to int: ", err)
		return
	}

	resp, err := json.Marshal(data.List[offset:limit])
	if err != nil {
		fmt.Println("cant pack result json:", err)
		return
	}

	w.Write(resp)

}

func TestFindUsersWithErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	cases := []TestCaseWithError{
		{
			Request:    SearchRequest{Limit: -1},
			ErrorExact: "limit must be > 0",
		},
		{
			Request:    SearchRequest{Offset: -1},
			ErrorExact: "offset must be > 0",
		},
		{
			URL:           "http://",
			ErrorContains: "unknown error",
		},
		{
			Request:       SearchRequest{Query: TimeoutErrorQuery},
			ErrorContains: "timeout for",
		},
		{
			AccessToken: InvalidToken,
			ErrorExact:  "Bad AccessToken",
		},
		{
			Request:    SearchRequest{Query: InternalErrorQuery},
			ErrorExact: "SearchServer fatal error",
		},
		{
			Request:       SearchRequest{Query: BadRequestErrorQuery},
			ErrorContains: "cant unpack error json",
		},
		{
			Request:       SearchRequest{Query: BadRequestUnknownErrorQuery},
			ErrorContains: "unknown bad request error",
		},
		{
			Request:    SearchRequest{OrderField: "order_field"},
			ErrorExact: "OrderFeld order_field invalid",
		},
		{
			Request:       SearchRequest{Query: InvalidJsonErrorQuery},
			ErrorContains: "cant unpack result json",
		},
	}

	for caseNum, item := range cases {
		url := server.URL
		if item.URL != "" {
			url = item.URL
		}

		client := SearchClient{
			URL:         url,
			AccessToken: item.AccessToken,
		}
		response, err := client.FindUsers(item.Request)

		if response != nil || err == nil {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if item.ErrorExact != "" && err.Error() != item.ErrorExact {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.ErrorExact, err.Error())
		}

		if item.ErrorContains != "" && !strings.Contains(err.Error(), item.ErrorContains) {
			t.Errorf("[%d] wrong result, expected %#v to contain %#v", caseNum, err.Error(), item.ErrorContains)
		}
	}

}

func TestFindUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	cases := []TestCase{
		{
			SearchRequest{Limit: 1},
		},
		{
			SearchRequest{Limit: 30},
		},
		{
			SearchRequest{Limit: 25, Offset: 1},
		},
	}

	for caseNum, item := range cases {
		client := SearchClient{
			URL: server.URL,
		}
		response, err := client.FindUsers(item.Request)

		// we just need to cover 100% - so need in real testing
		if response == nil || err != nil {
			t.Errorf("[%d] expected response, got error", caseNum)
		}
	}
}
