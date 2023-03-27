// Package main implements the main entrypoint for ecnet-controller and utility routines to
// bootstrap the various internal components of ecnet-controller.
// ecnet-controller is the core control plane component in ECNET responsible for programming sidecar proxies.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	configClientset "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/config/clientset/versioned"
	multiclusterClientset "github.com/flomesh-io/ErieCanal/pkg/ecnet/gen/client/multicluster/clientset/versioned"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/catalog"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/configurator"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/constants"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/errcode"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/health"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/httpserver"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/events"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/k8s/informers"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/logger"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/messaging"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver/registry"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/proxyserver/server"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/endpoint"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/multicluster"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/providers/fsm"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/service/providers/kube"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/signals"
	"github.com/flomesh-io/ErieCanal/pkg/ecnet/version"
)

var (
	verbosity           string
	meshName            string // An ID that uniquely identifies an ECNET instance
	ecnetNamespace      string
	ecnetServiceAccount string
	ecnetMeshConfigName string
	ecnetVersion        string
	trustDomain         string

	scheme = runtime.NewScheme()
)

var (
	flags = pflag.NewFlagSet(`ecnet-controller`, pflag.ExitOnError)
	log   = logger.New("ecnet-controller/main")
)

func init() {
	flags.StringVarP(&verbosity, "verbosity", "v", constants.DefaultECNETLogLevel, "Set boot log verbosity level")
	flags.StringVar(&meshName, "mesh-name", "", "ECNET mesh name")
	flags.StringVar(&ecnetNamespace, "ecnet-namespace", "", "ECNET controller's namespace")
	flags.StringVar(&ecnetServiceAccount, "ecnet-service-account", "", "ECNET controller's service account")
	flags.StringVar(&ecnetMeshConfigName, "ecnet-config-name", "ecnet-mesh-config", "Name of the ECNET MeshConfig")
	flags.StringVar(&ecnetVersion, "ecnet-version", "", "Version of ECNET")

	// TODO (#4502): Remove when we add full MRC support
	flags.StringVar(&trustDomain, "trust-domain", "cluster.local", "The trust domain to use as part of the common name when requesting new certificates")

	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

func main() {
	log.Info().Msgf("Starting ecnet-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
	if err := parseFlags(); err != nil {
		log.Fatal().Err(err).Str(errcode.Kind, errcode.ErrInvalidCLIArgument.String()).Msg("Error parsing cmd line arguments")
	}

	if err := logger.SetLogLevel(verbosity); err != nil {
		log.Fatal().Err(err).Msg("Error setting log level")
	}

	// Initialize kube config and client
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating kube configs using in-cluster config")
	}
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	configClient := configClientset.NewForConfigOrDie(kubeConfig)
	multiclusterClient := multiclusterClientset.NewForConfigOrDie(kubeConfig)

	k8s.SetTrustDomain(trustDomain)

	// Initialize the generic Kubernetes event recorder and associate it with the ecnet-controller pod resource
	controllerPod, err := getECNETControllerPod(kubeClient)
	if err != nil {
		log.Fatal().Msg("Error fetching ecnet-controller pod")
	}
	eventRecorder := events.GenericEventRecorder()
	if err := eventRecorder.Initialize(controllerPod, kubeClient, ecnetNamespace); err != nil {
		log.Fatal().Msg("Error initializing generic event recorder")
	}

	// This ensures CLI parameters (and dependent values) are correct.
	if err := validateCLIParams(); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InvalidCLIParameters, "Error validating CLI parameters")
	}

	_, cancel := context.WithCancel(context.Background())
	stop := signals.RegisterExitHandlers(cancel)

	msgBroker := messaging.NewBroker(stop)

	informerCollection, err := informers.NewInformerCollection(meshName, stop,
		informers.WithKubeClient(kubeClient),
		informers.WithConfigClient(configClient, ecnetMeshConfigName, ecnetNamespace),
		informers.WithMultiClusterClient(multiclusterClient),
	)
	if err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error creating informer collection")
	}

	// This component will be watching resources in the config.openservicemesh.io API group
	cfg := configurator.NewConfigurator(informerCollection, ecnetNamespace, ecnetMeshConfigName, msgBroker)
	k8sClient := k8s.NewKubernetesController(informerCollection, msgBroker)
	multiclusterController := multicluster.NewMultiClusterController(informerCollection, kubeClient, k8sClient, msgBroker)
	kubeProvider := kube.NewClient(k8sClient, cfg)
	multiclusterProvider := fsm.NewClient(multiclusterController, cfg)
	endpointsProviders := []endpoint.Provider{kubeProvider, multiclusterProvider}
	serviceProviders := []service.Provider{kubeProvider, multiclusterProvider}

	meshCatalog := catalog.NewMeshCatalog(
		k8sClient,
		multiclusterController,
		stop,
		cfg,
		serviceProviders,
		endpointsProviders,
		msgBroker,
	)

	proxyRegistry := registry.NewProxyRegistry(msgBroker)
	// Create and start the pipy repo http service
	repoServer := server.NewRepoServer(meshCatalog, proxyRegistry, ecnetNamespace, cfg, k8sClient, msgBroker)
	// Create and start the proxy service
	if err = repoServer.Start(cfg.GetProxyServerPort()); err != nil {
		events.GenericEventRecorder().FatalEvent(err, events.InitializationError, "Error initializing proxy control server")
	}

	// Initialize ECNET's http service server
	httpServer := httpserver.NewHTTPServer(constants.ECNETHTTPServerPort)
	// Health/Liveness probes
	funcProbes := []health.Probes{repoServer}
	httpServer.AddHandlers(map[string]http.Handler{
		constants.ECNETControllerReadinessPath: health.ReadinessHandler(funcProbes, nil),
		constants.ECNETControllerLivenessPath:  health.LivenessHandler(funcProbes, nil),
	})
	// Version
	httpServer.AddHandler(constants.VersionPath, version.GetVersionHandler())

	// Start HTTP server
	err = httpServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to start ECNET metrics/probes HTTP server")
	}

	// Start the global log level watcher that updates the log level dynamically
	go k8s.WatchAndUpdateLogLevel(msgBroker, stop)

	<-stop
	cancel()
	log.Info().Msgf("Stopping ecnet-controller %s; %s; %s", version.Version, version.GitCommit, version.BuildDate)
}

func parseFlags() error {
	if err := flags.Parse(os.Args); err != nil {
		return err
	}
	_ = flag.CommandLine.Parse([]string{})
	return nil
}

// getECNETControllerPod returns the ecnet-controller pod.
// The pod name is inferred from the 'CONTROLLER_POD_NAME' env variable which is set during deployment.
func getECNETControllerPod(kubeClient kubernetes.Interface) (*corev1.Pod, error) {
	podName := os.Getenv("CONTROLLER_POD_NAME")
	if podName == "" {
		return nil, fmt.Errorf("CONTROLLER_POD_NAME env variable cannot be empty")
	}

	pod, err := kubeClient.CoreV1().Pods(ecnetNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingControllerPod)).
			Msgf("Error retrieving ecnet-controller pod %s", podName)
		return nil, err
	}

	return pod, nil
}
