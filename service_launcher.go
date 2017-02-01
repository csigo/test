package test

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/fsouza/go-dockerclient"
)

// supported service types
const (
	ZooKeeper     ServiceType = "zookeeper"
	HBase         ServiceType = "hbase"
	Redis         ServiceType = "redis"
	Etcd          ServiceType = "etcd"
	Gnatsd        ServiceType = "gnatsd"
	Disque        ServiceType = "disque"
	Consul        ServiceType = "consul"
	ElasticSearch ServiceType = "elasticsearch"
)

// service running state
const (
	stateNew      int32 = iota
	stateStarting       = iota
	stateReady          = iota
	stateStopped        = iota
)

var (
	srvFactories = struct {
		sync.RWMutex
		facs map[ServiceType]ServiceFactory
	}{facs: map[ServiceType]ServiceFactory{}}
)

// ServiceType defines type
type ServiceType string

// ServiceOption defines option function to setup service
type ServiceOption func(Service) error

// ServiceLauncher defines an interface to create service
type ServiceLauncher interface {
	// Start creates and starts an instance of supported service by the give type. It
	// returns its listening ip:port and the corresponding stop function.
	Start(ServiceType, ...ServiceOption) (ipport string, stopFunc func() error, err error)
	// StopAll stop all created services
	StopAll() error
	// Get retruns service, return nil if no service for the given ipport
	Get(ipport string) interface{}
}

// Service represents a service
type Service interface {
	// Start launches the service and return its listening port
	Start() (string, error)
	// Stop stops the service
	Stop() error
	// StartDocker launches the service and return its listening port via docker
	StartDocker(*docker.Client) (string, error)
	// Stop stops the service via docker
	StopDocker(*docker.Client) error
}

// ServiceFactory represents service factory
type ServiceFactory func() Service

// RegisterService registers a service factory of the given type
func RegisterService(t ServiceType, f ServiceFactory) {
	srvFactories.Lock()
	defer srvFactories.Unlock()

	if _, ok := srvFactories.facs[t]; ok {
		panic(fmt.Errorf("aready register service type %s", t))
	}
	srvFactories.facs[t] = f
}

// NewServiceLauncher returns an instance of ServiceLauncher
func NewServiceLauncher() ServiceLauncher {
	return &serviceLauncherImpl{
		services: map[string]Service{},
	}
}

// serviceLauncherImpl implements ServiceLauncher
type serviceLauncherImpl struct {
	// service stores created services
	services map[string]Service
	// mutx to protected services
	sync.Mutex
}

// Create returns an instance of supported service by the give type
func (s *serviceLauncherImpl) Start(t ServiceType, options ...ServiceOption) (string, func() error, error) {
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
func (s *serviceLauncherImpl) StopAll() error {
	s.Lock()
	defer s.Unlock()

	errs := []error{}
	for _, s := range s.services {
		errs = append(errs, s.Stop())
	}
	s.services = map[string]Service{}
	return CombineError(errs...)
}

// Get return service by givne ip port
func (s *serviceLauncherImpl) Get(ipport string) interface{} {
	s.Lock()
	defer s.Unlock()
	return s.services[ipport]
}

// stateChkService helps to guard status of the embed service
// state machine: new -> starting -> ready -> stopped
type stateChkService struct {
	Service
	state int32
	cl    *docker.Client
}

func (s *stateChkService) Start() (ipport string, err error) {
	if !atomic.CompareAndSwapInt32(&s.state, stateNew, stateStarting) {
		return "", fmt.Errorf("state is not ready")
	}
	if s.cl != nil {
		ipport, err = s.Service.StartDocker(s.cl)
	} else {
		ipport, err = s.Service.Start()
	}
	if err == nil {
		atomic.StoreInt32(&s.state, stateReady)
	}
	return ipport, err
}

func (s *stateChkService) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.state, stateReady, stateStopped) {
		return fmt.Errorf("state is not ready")
	}
	if s.cl != nil {
		return s.Service.StopDocker(s.cl)
	}
	return s.Service.Stop()
}
