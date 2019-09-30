package app

import (
	metrics "code.cloudfoundry.org/go-metric-registry"
	"code.cloudfoundry.org/tlsconfig"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// Agent is a Leadership Election Agent. It determines if the local process
// should act as a leader or not.
type Agent struct {
	log  *log.Logger
	port int
	lis  net.Listener
	m    Metrics

	nodeIndex int
	nodes     []string

	mu      sync.RWMutex
	r       *raft.Raft
	network *errorCheckingTransport
}

// New returns a new Agent.
func New(nodeIndex int, nodes []string, opts ...AgentOption) *Agent {
	a := &Agent{
		log:  log.New(ioutil.Discard, "", 0),
		port: 8080,

		nodeIndex: nodeIndex,
		nodes:     nodes,
		m:         NopMetrics{},
	}

	for _, o := range opts {
		o(a)
	}

	return a
}

// AgentOption configures an Agent by overriding defaults.
type AgentOption func(*Agent)

// WithLogger returns an AgentOption that configures the logger for the Agent.
// It defaults to a silent logger.
func WithLogger(log *log.Logger) AgentOption {
	return func(a *Agent) {
		a.log = log
	}
}

// WithPort configures the port to bind the HTTP server to. It will always
// bind to localhost. Defaults to 8080.
func WithPort(port int) AgentOption {
	return func(a *Agent) {
		a.port = port
	}
}

// Metrics registers Gauge metrics.
type Metrics interface {
	NewGauge(name, helpText string, opts ...metrics.MetricOption) metrics.Gauge
}

type NopGauge struct{}

func (n NopGauge) Add(float64) {}

func (n NopGauge) Set(float64) {}

// NopMetrics implements Metrics, but simply discards them.
type NopMetrics struct{}

// NewGauge implements Metrics.
func (m NopMetrics) NewGauge(name, helpText string, opts ...metrics.MetricOption) metrics.Gauge {
	return NopGauge{}
}

// WithMetrics configures the metrics for Agent. Defaults to NopMetrics.
func WithMetrics(m Metrics) AgentOption {
	return func(a *Agent) {
		a.m = m
	}
}

// Start starts the Agent. It does not block.
func (a *Agent) Start(caFile, certFile, keyFile string) {
	tlsConfig, err := buildTLSConfig(caFile, certFile, keyFile)

	lis, err := tls.Listen("tcp", fmt.Sprintf("localhost:%d", a.port), tlsConfig)
	if err != nil {
		a.log.Fatalf("failed to listen on localhost:%d", a.port)
	}
	a.lis = lis

	setLeadershipStatus := a.m.NewGauge("leadership_status", "1 if this instance is the leader, 0 otherwise.")

	isLeader := a.startRaft()

	go func() {
		for range time.Tick(time.Second) {
			if isLeader() {
				setLeadershipStatus.Set(1)
				continue
			}
			setLeadershipStatus.Set(0)
		}
	}()

	srv := leaderStatusServer(isLeader)

	go func() {
		a.log.Fatal(srv.Serve(lis))
	}()
}

func leaderStatusServer(isLeader func() bool) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/leader", func(w http.ResponseWriter, r *http.Request) {
		if isLeader() {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusLocked)
	})
	srv := &http.Server{
		Handler: mux,
	}
	return srv
}

func buildTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(certFile, keyFile),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(caFile),
	)
	if err != nil {
		log.Fatal(err)
	}
	return tlsConfig, err
}

func (a *Agent) startRaft() func() bool {
	localAddr := a.nodes[a.nodeIndex]
	addr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		a.log.Fatalf("failed to resolve address %s: %s", localAddr, err)
	}

	tcpTransport, err := raft.NewTCPTransportWithLogger(
		localAddr,
		addr,
		100,
		30*time.Second,
		a.log,
	)
	if err != nil {
		a.log.Fatalf("failed to create raft TCP transport: %s", err)
	}
	a.network = newErrorCheckingTransport(tcpTransport)

	a.maintainRaft(localAddr)

	go func() {
		// Prune dead nodes
		for range time.Tick(10 * time.Millisecond) {
			a.maintainRaft(localAddr)
		}
	}()

	return func() bool {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return a.r.Leader() == raft.ServerAddress(addr.String())
	}
}

func (a *Agent) maintainRaft(localAddr string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.network.anyDead() && a.r != nil {
		return
	}

	firstBootup := a.r == nil

	if !firstBootup {
		a.r.Shutdown()
	}

	var peers []raft.Server
	for _, addr := range a.nodes {
		id := raft.ServerID(addr)
		if !firstBootup && !a.network.isWorking(id) && addr != localAddr {
			// Dead/faulty node. Don't include as peer.
			a.network.resetError(id)
			continue
		}

		// Not dead yet
		peers = append(peers, raft.Server{
			ID:      id,
			Address: raft.ServerAddress(addr),
		})
	}

	store := raft.NewInmemStore()
	var err error
	a.r, err = raft.NewRaft(
		&raft.Config{
			ProtocolVersion:    raft.ProtocolVersionMax,
			LocalID:            raft.ServerID(localAddr),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    1 * time.Second,
			CommitTimeout:      1 * time.Second,
			MaxAppendEntries:   100,
			SnapshotInterval:   time.Second,
			LeaderLeaseTimeout: 100 * time.Millisecond,
			LogOutput:          ioutil.Discard,
		},
		nil,
		store,
		store,
		raft.NewInmemSnapshotStore(),
		a.network,
	)

	if err != nil {
		a.log.Fatalf("failed to create raft cluster: %s", err)
	}

	a.r.BootstrapCluster(raft.Configuration{Servers: peers})
}

// Addr returns the address the Agent is listening to for HTTP requests (e.g.,
// 127.0.0.1:8080). It is only valid after calling Start().
func (a *Agent) Addr() string {
	return a.lis.Addr().String()
}

type errorCheckingTransport struct {
	raft.Transport
	nodeStatus map[raft.ServerID]serverStatus
	mu         sync.RWMutex
}

type serverStatus struct {
	errCount        int
	verifiedWorking bool
}

func newErrorCheckingTransport(t raft.Transport) *errorCheckingTransport {
	return &errorCheckingTransport{
		Transport:  t,
		nodeStatus: make(map[raft.ServerID]serverStatus),
	}
}

// AppendEntriesPipeline returns an interface that can be used to pipeline
// AppendEntries requests.
func (t *errorCheckingTransport) AppendEntriesPipeline(id raft.ServerID, target raft.ServerAddress) (raft.AppendPipeline, error) {
	p, err := t.Transport.AppendEntriesPipeline(id, target)

	if err != nil {
		t.incError(id)
		return nil, err
	}
	t.markWorking(id)
	return p, nil
}

// AppendEntries sends the appropriate RPC to the target node.
func (t *errorCheckingTransport) AppendEntries(id raft.ServerID, target raft.ServerAddress, args *raft.AppendEntriesRequest, resp *raft.AppendEntriesResponse) error {
	err := t.Transport.AppendEntries(id, target, args, resp)

	if err != nil {
		t.incError(id)
		return err
	}
	t.markWorking(id)
	return nil
}

// RequestVote sends the appropriate RPC to the target node.
func (t *errorCheckingTransport) RequestVote(id raft.ServerID, target raft.ServerAddress, args *raft.RequestVoteRequest, resp *raft.RequestVoteResponse) error {
	err := t.Transport.RequestVote(id, target, args, resp)

	if err != nil {
		t.incError(id)
		return err
	}
	t.markWorking(id)
	return nil
}

func (t *errorCheckingTransport) isWorking(id raft.ServerID) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.nodeStatus[id].verifiedWorking
}

func (t *errorCheckingTransport) anyDead() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, v := range t.nodeStatus {
		if v.errCount > 2 {
			return true
		}
	}

	return false
}

func (t *errorCheckingTransport) resetError(id raft.ServerID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodeStatus[id] = serverStatus{}
}

func (t *errorCheckingTransport) incError(id raft.ServerID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.nodeStatus[id]
	s.errCount++
	s.verifiedWorking = false
	t.nodeStatus[id] = s
}

func (t *errorCheckingTransport) markWorking(id raft.ServerID) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.nodeStatus[id] = serverStatus{verifiedWorking: true}
}
