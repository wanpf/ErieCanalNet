package server

import (
	"net"
	"sync"
)

// Server CNI Server.
type Server interface {
	Start() error
	Stop()
}

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

	cniReady       chan struct{}
	stop           chan struct{}
	hotUpgradeFlag bool
	wg             sync.WaitGroup
}
