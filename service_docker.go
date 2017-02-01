package test

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

// ServiceDocker defines an interface to create service via docker
type ServiceDocker interface {
	// Start creates and starts an instance of supported service by the give type. It
	// returns its listening ip:port and the corresponding stop function.
	Start(ServiceType, ...ServiceOption) (ipport string, stopFunc func() error, err error)
	// StopAll stop all created services
	StopAll() error
	// Get retruns service, return nil if no service for the given ipport
	Get(ipport string) interface{}
}

// NewServiceDocker returns an instance of ServiceDocker
// TODO: with specified docker address
func NewServiceDocker() ServiceDocker {
	return &serviceDockerImpl{
		services:     map[string]Service{},
		dockerclient: NewDockerClient(),
	}
}

// ContainerOptionFunc is a function that configures a Client.
// It is used in createDockerOptions.
type ContainerOptionFunc func(*docker.CreateContainerOptions) error

type ContainerNode struct {
	Command []string
	Env     []string
	Image   string
	Port    []string
}

// serviceDockerImpl implements ServiceDocker
type serviceDockerImpl struct {
	// docker client
	dockerclient *docker.Client
	// service stores created services
	services map[string]Service
	// mutx to protected services
	sync.Mutex
}

// Create returns an instance of supported service by the give type
func (s *serviceDockerImpl) Start(t ServiceType, options ...ServiceOption) (string, func() error, error) {
	s.Lock()
	defer s.Unlock()

	srvFactories.RLock()
	fac, ok := srvFactories.facs[t]
	srvFactories.RUnlock()
	if !ok {
		return "", nil, fmt.Errorf("unsupported service type %v", t)
	}
	// guard with state checker
	srv := &stateChkService{
		state:   stateNew,
		Service: fac(),
		cl:      s.dockerclient,
	}
	// apply option functions
	for _, opt := range options {
		if err := opt(srv.Service); err != nil {
			return "", nil, fmt.Errorf("failed to apply option %v", opt)
		}
	}
	// start service
	ipport, err := srv.Start()
	if err != nil {
		return "", nil, fmt.Errorf("unable to start service %v, err %v", t, err)
	}
	// store raw service
	s.services[ipport] = srv.Service
	return ipport, srv.Stop, nil
}

// StopAll stop all created services
func (s *serviceDockerImpl) StopAll() error {
	s.Lock()
	defer s.Unlock()

	errs := []error{}
	for _, ss := range s.services {
		errs = append(errs, ss.StopDocker(s.dockerclient))
	}
	s.services = map[string]Service{}
	return CombineError(errs...)
}

// Get return service by givne ip port
func (s *serviceDockerImpl) Get(ipport string) interface{} {
	s.Lock()
	defer s.Unlock()
	return s.services[ipport]
}

// NewDockerClient create docker client via unix socket
func NewDockerClient() (cl *docker.Client) {
	endpoint := "unix:///var/run/docker.sock"
	var err error
	cl, err = docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return cl
}

// RemoveContainer remove the started container
func RemoveContainer(client *docker.Client, container *docker.Container) error {
	return client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
}

// StartContainer starts the required container
func StartContainer(client *docker.Client, options ...ContainerOptionFunc) (c *docker.Container, ipaddr string, err error) {
	// TODO: handle context
	ctx := context.Background()
	opts, err := createDockerOptions(ctx, options...)
	if err != nil {
		return c, "", err
	}
	c, err = client.CreateContainer(opts)
	if err != nil {
		return c, "", err
	}

	err = client.StartContainerWithContext(c.ID, nil, ctx)
	if err != nil {
		RemoveContainer(client, c)
		return c, "", err
	}

	// wait for container to wake up
	if err := waitStarted(client, c.ID, 5*time.Second); err != nil {
		RemoveContainer(client, c)
		return c, "", err
	}
	c, err = client.InspectContainerWithContext(c.ID, ctx)
	if err != nil {
		RemoveContainer(client, c)
		return c, "", err
	}

	// determine IP address for Component
	var ip string
	if runtime.GOOS == "darwin" {
		ip = "localhost"
	} else {
		ip = strings.TrimSpace(c.NetworkSettings.IPAddress)
	}
	port := opts.Context.Value("csigo_test_port")

	// wait Component to wake up
	ipaddr = fmt.Sprintf("%s:%s", ip, port)
	fmt.Printf("%s\n", ipaddr)
	if err = waitReachable(ctx, ipaddr, 10*time.Second); err != nil {
		RemoveContainer(client, c)
		return c, "", err
	}
	return c, ipaddr, nil

}

//TODO:
//PortBindings: map[docker.Port][]docker.PortBinding{
//	"8888/tcp": {{HostIP: "", HostPort: "12345"}},
//},
// SetExposedPorts accepts format as "1234/tcp", "5678/udp"
func SetExposedPorts(ports []string) ContainerOptionFunc {
	return func(opts *docker.CreateContainerOptions) error {
		for i, exp := range ports {

			if i == 0 { // only first port will be checked and exposed
				connectPort := docker.Port(exp).Port()
				if runtime.GOOS == "darwin" {
					connectPort = strconv.Itoa(GetPort())
					opts.HostConfig.PortBindings[docker.Port(exp)] = append(opts.HostConfig.PortBindings[docker.Port(exp)], docker.PortBinding{
						HostPort: connectPort,
						HostIP:   "",
					})
				}
				opts.Context = context.WithValue(opts.Context, "csigo_test_port", connectPort)
			}
			opts.Config.ExposedPorts[docker.Port(exp)] = struct{}{}
		}
		return nil
	}
}

// SetImage set the docker image for the container
func SetImage(image string) ContainerOptionFunc {
	return func(opts *docker.CreateContainerOptions) error {
		opts.Config.Image = image
		return nil
	}
}

// SetEnv set the command for the container
func SetEnv(env []string) ContainerOptionFunc {
	return func(opts *docker.CreateContainerOptions) error {
		opts.Config.Env = env
		return nil
	}
}

// SetCommand set the command for the container
func SetCommand(cmd []string) ContainerOptionFunc {
	return func(opts *docker.CreateContainerOptions) error {
		opts.Config.Cmd = cmd
		return nil
	}
}

func createDockerOptions(ctx context.Context, options ...ContainerOptionFunc) (docker.CreateContainerOptions, error) {
	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			ExposedPorts: map[docker.Port]struct{}{},
		},
		HostConfig: &docker.HostConfig{
			PortBindings: map[docker.Port][]docker.PortBinding{},
		},
		NetworkingConfig: &docker.NetworkingConfig{},
		Context:          ctx,
	}

	// Run the options on it
	for _, option := range options {
		if err := option(&opts); err != nil {
			return opts, err
		}
	}
	return opts, nil
}

// waitReachable waits for hostport to became reachable for the maxWait time.
func waitReachable(ctx context.Context, hostport string, maxWait time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, maxWait)
	defer cancel()
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			c, err := net.DialTimeout("tcp", hostport, 3*time.Second)
			fmt.Printf("%s %v\n", hostport, err)
			if err == nil {
				c.Close()
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("cannot connect %v for %v: %v", hostport, maxWait, ctx.Err())
		}
	}
}

// waitStarted waits for a container to start for the maxWait time.
func waitStarted(client *docker.Client, id string, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		c, err := client.InspectContainer(id)
		if err != nil {
			break
		}
		if c.State.Running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("cannot start container %s for %v", id, maxWait)
}
