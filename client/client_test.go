package client_test

import (
	"encoding/json"
	"github.com/tywkeene/autobd/api"
	"github.com/tywkeene/autobd/client"
	"github.com/tywkeene/autobd/index"
	"github.com/tywkeene/autobd/node"
	"github.com/tywkeene/autobd/version"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockServer func(http.ResponseWriter, *http.Request)

func newMockServer(handle mockServer) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(w, r)
	}))
}

func TestServeServerVersion(t *testing.T) {
	s := newMockServer(api.ServeServerVer)
	defer s.Close()

	//Gotta make sure the version is actually set on the server
	version.Set("commit", "api", "node", "cli")

	client := client.NewClient(s.URL)
	response, err := client.RequestVersion()
	if err != nil {
		t.Fatal(err)
	}

	if response == nil {
		t.Fatal("Empty response from server")
	}
	t.Log("Got response:", string(response))
}

func TestServeIndex(t *testing.T) {
	s := newMockServer(api.ServeIndex)
	defer s.Close()

	client := client.NewClient(s.URL)

	//Register a dummy node with the mock server
	api.CurrentNodes = make(map[string]*api.Node)
	api.CurrentNodes["test"] = &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false}

	//What we expect from the server
	expect, err := index.GetIndex("./")
	if err != nil {
		t.Fatal(err)
	}

	//Get our response
	serial, err := client.RequestIndex("./", "test", "autobd-test")
	if err != nil {
		t.Fatal(err)
	}

	var responseIndex map[string]*index.Index
	if err := json.Unmarshal(serial, &responseIndex); err != nil {
		t.Fatal(err)
	}

	if need := node.CompareDirs(responseIndex, expect); need == nil {
		t.Fatal("Mismatched indexes")
	}
}

/* TODO: Fix this, obviously */
/*
func TestRequestSyncDir(t *testing.T) {
	s := newMockServer(api.ServeSync)
	defer s.Close()

	os.Mkdir("./test", 0700)

	client := NewClient(s.URL)
	version.Set("commit", "api", "node")

	//Register a dummy node with the mock server
	api.CurrentNodes = make(map[string]*api.Node)
	api.CurrentNodes["test"] = &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false}

	if err := client.RequestSyncDir("./test", "test"); err != nil {
		t.Error(err)
	}
}
*/

/*TODO: TestRequestSyncFile*/

func TestCompareIndexSuccess(t *testing.T) {
	s := newMockServer(api.ServeIndex)
	defer s.Close()

	c := client.NewClient(s.URL)

	//Register a dummy node with the mock server
	api.CurrentNodes = make(map[string]*api.Node)
	api.CurrentNodes["test"] = &api.Node{"0.0.0.0", "0.0.0", time.Now().Format(time.RFC850), true, false}

	//What we expect from the server
	expect, err := index.GetIndex("./")
	if err != nil {
		t.Fatal(err)
	}

	//Get our response
	serial, err := c.RequestIndex("./", "test", "autobd-test")
	if err != nil {
		t.Fatal(err)
	}
	var responseIndex map[string]*index.Index
	if err := json.Unmarshal(serial, &responseIndex); err != nil {
		t.Fatal(err)
	}

	if need := node.CompareDirs(responseIndex, expect); need == nil {
		t.Fatal("Mismatched indexes")
	}

}

func TestIdentifyWithServer(t *testing.T) {
	s := newMockServer(api.Identify)
	defer s.Close()

	c := client.NewClient(s.URL)
	c.IdentifyWithServer("test", "autobd-test")

	if api.CurrentNodes == nil {
		api.CurrentNodes = make(map[string]*api.Node)
	}

	if api.CurrentNodes["test"] == nil {
		t.Fatal("Node should exist")
	}
}

func TestSendHeartbeat(t *testing.T) {
	s := newMockServer(api.HeartBeat)
	defer s.Close()

	c := client.NewClient(s.URL)
	c.SendHeartbeat("test", true, "autobd-test")

	node := api.CurrentNodes["test"]

	if node == nil {
		t.Fatal("Node should exist")
	}

	if node.Synced == false {
		t.Fatal("Node was not synced")
	}
}
