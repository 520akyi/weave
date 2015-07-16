package proxy

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fsouza/go-dockerclient"
	. "github.com/weaveworks/weave/common"
)

type startContainerInterceptor struct{ proxy *Proxy }

func (i *startContainerInterceptor) InterceptRequest(r *http.Request) error {
	return nil
}

func (i *startContainerInterceptor) InterceptResponse(r *http.Response) error {
	container, err := inspectContainerInPath(i.proxy.client, r.Request.URL.Path)
	if err != nil {
		return err
	}

	if !validNetworkMode(container.HostConfig) {
		Debug.Printf("Ignoring container %s with --net=%s", container.ID, networkMode(container.HostConfig))
		return nil
	}
	cidrs, ok := i.proxy.weaveCIDRsFromConfig(container.Config, container.HostConfig)
	if !ok {
		Debug.Print("No Weave CIDR, ignoring")
		return nil
	}
	Info.Printf("Attaching container %s with WEAVE_CIDR \"%s\" to weave network", container.ID, strings.Join(cidrs, " "))
	args := []string{"attach"}
	args = append(args, cidrs...)
	args = append(args, "--or-die", container.ID)
	if output, err := callWeave(args...); err != nil {
		Warning.Printf("Attaching container %s to weave network failed: %s", container.ID, string(output))
		return errors.New(string(output))
	}

	return i.proxy.client.KillContainer(docker.KillContainerOptions{ID: container.ID, Signal: docker.SIGUSR2})
}
