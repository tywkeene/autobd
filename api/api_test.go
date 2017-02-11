package api_test

import (
	"bytes"
	"encoding/json"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/options"
	"github.com/tywkeene/autobd/utils"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func getResponseFromBody(t *testing.T, recorder *httptest.ResponseRecorder) string {
	buffer, err := ioutil.ReadAll(recorder.Body)
	var response string
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

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			recorder.Code, http.StatusUnauthorized)
	}
	expected := &utils.APIError{
		ErrorMessage: "Invalid node UUID",
		HTTPStatus:   http.StatusUnauthorized,
	}
	var response *utils.APIError
	buffer, err := ioutil.ReadAll(recorder.Body)
	if err = json.Unmarshal(buffer, &response); err != nil {
		t.Fatal(err)
	}
	if response.ErrorMessage != expected.ErrorMessage ||
		response.HTTPStatus != expected.HTTPStatus {
		t.Errorf("handler returned unexpected body: got %v want %v",
			recorder.Body.String(), expected)
	}
}

//Ensure the /index endpoint fails if we don't specify a UUID but no directory to index
func TestServeIndexNoDir(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	options.Config.HeartBeatTrackInterval = "1s"
	options.Config.HeartBeatOffline = "3s"

	api.AddNode("test", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test",
			Version: "0.0.0",
		},
	})

	req, err := http.NewRequest("GET", "/index?uuid=test", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("handler returned wrong status code: got %v want %v",
			recorder.Code, http.StatusBadRequest)
	}
	expected := &utils.APIError{
		ErrorMessage: "Must specify directory",
		HTTPStatus:   http.StatusBadRequest,
	}
	var response *utils.APIError
	buffer, err := ioutil.ReadAll(recorder.Body)
	if err = json.Unmarshal(buffer, &response); err != nil {
		t.Fatal(err)
	}

	if response == nil {
		t.Errorf("empty response from server")
	}
	if response.ErrorMessage != expected.ErrorMessage ||
		response.HTTPStatus != expected.HTTPStatus {
		t.Errorf("handler returned unexpected body: got %v want %v",
			response, expected)
	}
}

//Ensure the /index endpoint succeeds specify a UUID and directory to index
func TestServeIndex(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	options.Config.HeartBeatTrackInterval = "1s"
	options.Config.HeartBeatOffline = "3s"

	api.AddNode("test", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test",
			Version: "0.0.0",
		},
	})

	req, err := http.NewRequest("GET", "/index?dir=/&uuid=test", nil)
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

func BenchmarkServeIndex(b *testing.B) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeIndex)

	options.Config.HeartBeatTrackInterval = "1s"
	options.Config.HeartBeatOffline = "3s"

	api.AddNode("test", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test",
			Version: "0.0.0",
		},
	})

	req, err := http.NewRequest("GET", "/index?dir=/&uuid=test", nil)
	if err != nil {
		b.Fatal(err)
	}
	for n := 0; n < 20000; n++ {
		handler.ServeHTTP(recorder, req)
	}
}

//Ensure we can get a version from the server
func TestServeServerVer(t *testing.T) {
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
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.ServeSync)

	api.AddNode("test", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test",
			Version: "0.0.0",
		},
	})

	req, err := http.NewRequest("GET", "/sync?grab=api.go&uuid=test", nil)
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
	api.AddNode("test0", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test0",
			Version: "0.0.0",
		},
	})
	api.AddNode("test1", &api.Node{
		Address:    "0.0.0.1",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test1",
			Version: "0.0.0",
		},
	})
	api.AddNode("test2", &api.Node{
		Address:    "0.0.0.2",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test2",
			Version: "0.0.0",
		},
	})
	api.AddNode("test3", &api.Node{
		Address:    "0.0.0.3",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test3",
			Version: "0.0.0",
		},
	})

	req, err := http.NewRequest("GET", "/nodes?uuid=test0", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			recorder.Code, http.StatusOK)
	}

	buffer, err := ioutil.ReadAll(recorder.Body)
	var response map[string]*api.Node
	if err = json.Unmarshal(buffer, &response); err != nil {
		t.Error(err)
	}

	if response == nil {
		t.Error("Failed to get node list from server")
	}
	for key, _ := range response {
		if _, ok := api.CurrentNodes[key]; ok == false {
			t.Error("Node not in node list")
		}
	}
}

//Ensure we can identify as a node with the server
func TestIdentify(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.Identify)

	api.CurrentNodes = nil

	serial, err := json.Marshal(&api.NodeMetadata{
		Version: "0.0.0",
		UUID:    "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/identify", bytes.NewBuffer(serial))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Fatal("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if api.GetNodeByUUID("test") == nil {
		t.Fatal("Node was not properly registered")
	}
}

//Ensure the server properly handles heartbeats from a node
func TestHeartBeat(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(api.HeartBeat)

	api.AddNode("test", &api.Node{
		Address:    "0.0.0.0",
		LastOnline: time.Now().Format(time.RFC850),
		IsOnline:   true,
		Synced:     false,
		Meta: &api.NodeMetadata{
			UUID:    "test",
			Version: "0.0.0",
		},
	})
	heartbeat := &api.NodeHeartbeat{
		UUID:   "test",
		Synced: strconv.FormatBool(true),
	}

	serial, err := json.Marshal(&heartbeat)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/heartbeat", bytes.NewBuffer(serial))
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(recorder, req)

	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if api.GetNodeByUUID("test").Synced == false {
		t.Errorf("Node was not updated")
	}
}
