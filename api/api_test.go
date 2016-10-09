package api_test

import (
	"encoding/json"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func getResponseFromBody(t *testing.T, body io.Reader) interface{} {
	buffer, err := ioutil.ReadAll(body)
	var response interface{}
	if err = json.Unmarshal(buffer, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

//Ensure the server serves gzip encoded content if we say we can handle it
func TestGzip(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.GzipHandler(api.ServeServerVer))

	req, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "application/x-gzip")
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	isGzipped := recorder.HeaderMap.Get("Content-Encoding")
	if isGzipped == "" {
		t.Errorf("Server did not gzip response")
	}
}

//Ensure the /index endpoint fails if we specify a directory to index but no UUID
func TestServeIndexNoUUID(t *testing.T) {
	req, err := http.NewRequest("GET", "/index?dir=/", nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	expected := "Invalid or empty node UUID"
	response := getResponseFromBody(t, recorder.Body)
	if response != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			recorder.Body.String(), string(expected))
	}
}

//Ensure the /index endpoint fails if we don't specify a UUID but no directory to index
func TestServeIndexNoDir(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	options.Config.HeartBeatTrackInterval = "1s"
	options.Config.HeartBeatOffline = "3s"

	api.AddNode("testing", &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false})

	req, err := http.NewRequest("GET", "/index?uuid=testing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	expected := "Must specify dir"
	response := getResponseFromBody(t, recorder.Body)
	if response != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			recorder.Body.String(), string(expected))
	}
}

//Ensure the /index endpoint succeeds specify a UUID and directory to index
func TestServeIndex(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	options.Config.HeartBeatTrackInterval = "1s"
	options.Config.HeartBeatOffline = "3s"

	api.AddNode("testing", &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false})

	req, err := http.NewRequest("GET", "/index?dir=/&uuid=testing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	respJSON, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		t.Fatal(err)
	}
	var respIndex *index.Index
	if err := json.Unmarshal(respJSON, &respIndex); err != nil {
		t.Fatal(err)
	}

	expectedIndex, err := index.GetIndex("./")
	if err != nil {
		t.Fatal(err)
	}

	if need := node.CompareDirs(expectedIndex, respIndex.Files); len(need) > 0 {
		t.Fatal("Index mismatch")
	}
}

//Ensure we can get a version from the server
func TestServeServerVer(t *testing.T) {
	//TODO: Figure out how to actually set the version from here so we can test this properly
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeServerVer)

	req, err := http.NewRequest("GET", "/version", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

//Ensure we get content when trying to sync a file
func TestServeSync(t *testing.T) {
	//TODO: Add checksum and other file-metadata verification
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeSync)

	api.AddNode("testing", &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false})

	req, err := http.NewRequest("GET", "/sync?grab=api.go&uuid=testing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	response, err := ioutil.ReadAll(recorder.Body)
	if err != nil {
		t.Error(err)
	}
	if string(response) == "" {
		t.Errorf("Server failed to sync file")
	}
}

//Ensure we get a consistent list of nodes
func TestListNodes(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ListNodes)

	api.CurrentNodes = nil
	api.AddNode("testing0", &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false})
	api.AddNode("testing1", &api.Node{"0.0.0.1", "0.0.0", time.Now().Format(time.RFC850), true, false})
	api.AddNode("testing2", &api.Node{"0.0.0.2", "0.0.0", time.Now().Format(time.RFC850), true, false})
	api.AddNode("testing3", &api.Node{"0.0.0.3", "0.0.0", time.Now().Format(time.RFC850), true, false})

	req, err := http.NewRequest("GET", "/nodes?uuid=testing0", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	buffer, err := ioutil.ReadAll(recorder.Body)
	var response map[string]*api.Node
	if err = json.Unmarshal(buffer, &response); err != nil {
		t.Error(err)
	}

	if response == nil {
		t.Error("Failed to get node list from server")
	}
	if reflect.DeepEqual(response, api.CurrentNodes) == false {
		t.Errorf("Node lists do not match")
	}
}

//Ensure we can identify as a node with the server
func TestIdentify(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.Identify)

	req, err := http.NewRequest("GET", "/identify?uuid=testing&version=testing", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if api.GetNodeByUUID("testing") == nil {
		t.Errorf("Node was not properly registered")
	}
}

//Ensure the server properly handles heartbeats from a node
func TestHeartBeat(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.HeartBeat)

	api.AddNode("testing", &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false})

	req, err := http.NewRequest("GET", "/heartbeat?uuid=testing&synced=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if api.GetNodeByUUID("testing").Synced != true {
		t.Errorf("Node was not updated")
	}
}
