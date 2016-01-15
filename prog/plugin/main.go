package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/docker/libnetwork/ipamapi"
	. "github.com/weaveworks/weave/common"
	"github.com/weaveworks/weave/common/docker"
	ipamplugin "github.com/weaveworks/weave/plugin/ipam"
	netplugin "github.com/weaveworks/weave/plugin/net"
	"github.com/weaveworks/weave/plugin/skel"
)

var version = "(unreleased version)"

func main() {
	var (
		justVersion      bool
		address          string
		nameserver       string
		meshAddress      string
		logLevel         string
		noMulticastRoute bool
	)

	flag.BoolVar(&justVersion, "version", false, "print version and exit")
	flag.StringVar(&logLevel, "log-level", "info", "logging level (debug, info, warning, error)")
	flag.StringVar(&address, "socket", "/run/docker/plugins/weave.sock", "socket on which to listen")
	flag.StringVar(&nameserver, "nameserver", "", "nameserver to provide to containers")
	flag.StringVar(&meshAddress, "meshsocket", "/run/docker/plugins/weavemesh.sock", "socket on which to listen in mesh mode")
	flag.BoolVar(&noMulticastRoute, "no-multicast-route", false, "do not add a multicast route to network endpoints")

	flag.Parse()

	if justVersion {
		fmt.Printf("weave plugin %s\n", version)
		os.Exit(0)
	}

	SetLogLevel(logLevel)

	// API 1.21 is the first version that supports docker network commands
	dockerClient, err := docker.NewVersionedClientFromEnv("1.21")
	if err != nil {
		Log.Fatalf("unable to connect to docker: %s", err)
	}

	Log.Println("Weave plugin", version, "Command line options:", os.Args[1:])
	Log.Info(dockerClient.Info())

	var globalListener, meshListener net.Listener
	endChan := make(chan error, 1)
	if address != "" {
		globalListener, err := listenAndServe(dockerClient, address, nameserver, noMulticastRoute, endChan, "global", false)
		if err != nil {
			Log.Fatalf("unable to create driver: %s", err)
		}
		defer globalListener.Close()
	}
	if meshAddress != "" {
		meshListener, err := listenAndServe(dockerClient, meshAddress, nameserver, noMulticastRoute, endChan, "local", true)
		if err != nil {
			Log.Fatalf("unable to create driver: %s", err)
		}
		defer meshListener.Close()
	}

	err = <-endChan
	if err != nil {
		Log.Errorf("Error from listener: %s", err)
		globalListener.Close()
		meshListener.Close()
		os.Exit(1)
	}
}

func listenAndServe(dockerClient *docker.Client, address, nameserver string, noMulticastRoute bool, endChan chan<- error, scope string, withIpam bool) (net.Listener, error) {
	d, err := netplugin.New(dockerClient, version, nameserver, scope, noMulticastRoute)
	if err != nil {
		return nil, err
	}

	var i ipamapi.Ipam
	if withIpam {
		if i, err = ipamplugin.NewIpam(dockerClient, version); err != nil {
			return nil, err
		}
	}

	var listener net.Listener

	// remove sockets from last invocation
	if err := os.Remove(address); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	listener, err = net.Listen("unix", address)
	if err != nil {
		return nil, err
	}
	Log.Printf("Listening on %s for %s scope", address, scope)

	go func() {
		endChan <- skel.Listen(listener, d, i)
	}()

	return listener, nil
}
