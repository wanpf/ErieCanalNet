package repo

import (
	"fmt"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy/client"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/sidecar/providers/pipy/registry"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/workerpool"
)

const (
	// ServerType is the type identifier for the ADS server
	ServerType = "pipy-Repo"

	// workerPoolSize is the default number of workerpool workers (0 is GOMAXPROCS)
	workerPoolSize = 0

	ecnetCodebaseConfig = "config.json"
)

var (
	ecnetCodebase      = "ecnet/base"
	ecnetProxyCodebase = "ecnet"
	ecnetCodebaseRepo  = fmt.Sprintf("/%s", ecnetCodebase)
)

// NewRepoServer creates a new Aggregated Discovery Service server
func NewRepoServer(meshCatalog catalog.MeshCataloger, proxyRegistry *registry.ProxyRegistry, ecnetNamespace string, cfg configurator.Configurator, kubecontroller k8s.Controller, msgBroker *messaging.Broker) *Server {
	if len(cfg.GetRepoServerCodebase()) > 0 {
		ecnetCodebase = fmt.Sprintf("%s/%s", cfg.GetRepoServerCodebase(), ecnetCodebase)
		ecnetProxyCodebase = fmt.Sprintf("%s/%s", cfg.GetRepoServerCodebase(), ecnetProxyCodebase)
		ecnetCodebaseRepo = fmt.Sprintf("/%s", ecnetCodebase)
	}

	server := Server{
		catalog:        meshCatalog,
		proxyRegistry:  proxyRegistry,
		ecnetNamespace: ecnetNamespace,
		cfg:            cfg,
		workQueues:     workerpool.NewWorkerPool(workerPoolSize),
		kubeController: kubecontroller,
		configVerMutex: sync.Mutex{},
		configVersion:  make(map[string]uint64),
		pluginSet:      mapset.NewSet(),
		msgBroker:      msgBroker,
		repoClient:     client.NewRepoClient(cfg.GetRepoServerIPAddr(), uint16(cfg.GetProxyServerPort())),
	}

	return &server
}

// Start starts the codebase push server
func (s *Server) Start(_ uint32) error {
	// wait until pipy repo is up
	err := wait.PollImmediate(5*time.Second, 90*time.Second, func() (bool, error) {
		success, err := s.repoClient.IsRepoUp()
		if success {
			log.Info().Msg("Repo is READY!")
			return success, nil
		}
		log.Error().Msg("Repo is not up, sleeping ...")
		return success, err
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	_, err = s.repoClient.Batch(fmt.Sprintf("%d", 0), []client.Batch{
		{
			Basepath: ecnetCodebase,
			Items:    ecnetCodebaseItems,
		},
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	// wait until base codebase is ready
	err = wait.PollImmediate(5*time.Second, 90*time.Second, func() (bool, error) {
		success, _, _ := s.repoClient.GetCodebase(ecnetCodebase)
		if success {
			log.Info().Msg("Base codebase is READY!")
			return success, nil
		}
		log.Error().Msg("Base codebase is NOT READY, sleeping ...")
		return success, err
	})
	if err != nil {
		log.Error().Err(err)
		return err
	}

	// Start broadcast listener thread
	go s.broadcastListener()

	s.ready = true

	return nil
}
