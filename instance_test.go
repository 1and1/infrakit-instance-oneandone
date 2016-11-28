package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

var (
	tags          = map[string]string{"group": "test"}
	instanceProps = json.RawMessage(`{
			"Appliance": "CoreOS_Stable_64std",
			"Datacenter": "DE",
			"FixedServerSize": "S",
			"Password": "Hn249NsxPb",
			"SSHKey": "ssh-rsa AAAAABBBBCCCC12334",
			"SSHKeyPath": "/home/nb/.ssh/id_rsa.pub",
			"FirewallID": "ABCDEF12345",
			"LoadBalancerID": "01234FEDCBA",
			"MonitorPolicyID": "ABC987DEF654"
		}
	`)
	serverIn = `{
		"id": "A0B0C0D0E0F0",
		"name": "instance-0123456789",
		"status": {
			"state": "POWERED_ON",
			"percent": 0
		},
		"ips": [{
			"id": "59491508DF4B",
			"ip": "11.22.33.44"
		}]
	}`
)

func serverResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, serverIn)
}

func listServerResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, fmt.Sprintf("[%s]", serverIn))
}

func instanceHandler() *mux.Router {
	r := mux.NewRouter()
	r.Queries("q", "instance-0123456789")
	r.Queries("keep_ips", "false")
	r.HandleFunc("/servers", serverResponse).Methods("POST")
	r.HandleFunc("/servers", listServerResponse).Methods("GET")
	r.HandleFunc("/servers/instance-0123456789", serverResponse).Methods("DELETE")

	return r
}

func TestInstanceLifecycle(t *testing.T) {
	ts := httptest.NewServer(instanceHandler())
	defer ts.Close()

	prop := properties{}

	require.NoError(t, json.Unmarshal([]byte(instanceProps), &prop))

	p := &Provisioner{
		client: oneandone.New("dummykey", ts.URL),
		prop:   prop,
		Dir:    "./",
		fs:     afero.NewMemMapFs(),
		wait:   false,
	}

	instanceID, err := p.Provision(instance.Spec{Properties: &instanceProps, Tags: tags})
	require.NoError(t, err)

	var buff []byte
	buff, err = afero.ReadFile(p.fs, filepath.Join(p.Dir, string(*instanceID)+".1and1.spec"))
	require.NoError(t, err)

	desc := instance.Description{}
	err = json.Unmarshal(buff, &desc)
	require.NoError(t, err)
	require.Equal(t, string(*instanceID), string(desc.ID))

	require.NoError(t, p.Destroy(*instanceID))
}

func TestCreateInstanceError(t *testing.T) {
	prop := properties{}
	require.NoError(t, json.Unmarshal([]byte(instanceProps), &prop))

	p := &Provisioner{
		client: oneandone.New("dummykey", "http://localhost/dummy"),
		prop:   prop,
		Dir:    "./",
		fs:     afero.NewMemMapFs(),
		wait:   true,
	}

	id, err := p.Provision(instance.Spec{Properties: &instanceProps, Tags: tags})

	require.Error(t, err)
	require.Nil(t, id)
}
