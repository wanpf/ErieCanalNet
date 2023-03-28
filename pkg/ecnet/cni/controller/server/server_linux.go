// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/flomesh-io/ErieCanal/pkg/ecnet/cni/config"
)

type qdisc struct {
	netns         string
	device        string
	managedClsact bool
}

type server struct {
	sync.Mutex
	serviceMeshMode string
	unixSockPath    string
	bpfMountPath    string
	// qdiscs is for cleaning up all tc programs when exists
	// key: netns(inode), value: qdisc info
	qdiscs map[uint64]qdisc
	// listeners are the dummy sockets created for eBPF programs to fetch the current pod ip
	// key: netns(inode), value: net.Listener
	listeners map[uint64]net.Listener

	cniReady chan struct{}
	stop     chan struct{}
}

// NewServer returns a new CNI Server.
// the path this the unix path to listen.
func NewServer(unixSockPath string, bpfMountPath string, cniReady, stop chan struct{}) Server {
	if unixSockPath == "" {
		unixSockPath = path.Join(config.HostVarRun, "ecnet-cni.sock")
	}
	if bpfMountPath == "" {
		bpfMountPath = "/sys/fs/bpf"
	}
	return &server{
		unixSockPath: unixSockPath,
		bpfMountPath: bpfMountPath,
		qdiscs:       make(map[uint64]qdisc),
		listeners:    make(map[uint64]net.Listener),
		cniReady:     cniReady,
		stop:         stop,
	}
}

func (s *server) Start() error {
	if err := os.RemoveAll(s.unixSockPath); err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("unix", s.unixSockPath)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	r := mux.NewRouter()
	r.Path(config.CNICreatePodURL).
		Methods("POST").
		HandlerFunc(s.PodCreated)

	r.Path(config.CNIDeletePodURL).
		Methods("POST").
		HandlerFunc(s.PodDeleted)

	ss := http.Server{
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	go func() {
		go ss.Serve(l) // nolint: errcheck
		// TODO: unify all clean-up functions
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGABRT)
		select {
		case <-ch:
			s.Stop()
		case <-s.stop:
			s.Stop()
		}
		_ = ss.Shutdown(context.Background())
	}()

	s.installCNI()
	// wait for cni to be ready
	<-s.cniReady
	if err = s.checkAndRepairPodPrograms(); err != nil {
		log.Errorf("Failed to check existing pods: %v", err)
	}
	return nil
}

func (s *server) installCNI() {
	install := newInstaller()
	go func() {
		if err := install.Run(context.TODO(), s.cniReady); err != nil {
			log.Error(err)
			close(s.cniReady)
		}
		if err := install.Cleanup(); err != nil {
			log.Errorf("Failed to clean up CNI: %v", err)
		}
	}()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGABRT)
		<-ch
		if err := install.Cleanup(); err != nil {
			log.Errorf("Failed to clean up CNI: %v", err)
		}
	}()
}

func (s *server) Stop() {
	log.Infof("cni-server stop ...")
	s.cleanUpTC()
	close(s.stop)
}
