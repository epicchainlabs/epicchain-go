package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nspcc-dev/neo-go/internal/fakechain"
	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/network/payload"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type testDiscovery struct {
	sync.Mutex
	bad          []string
	connected    []string
	unregistered []string
	backfill     []string
}

func newTestDiscovery([]string, time.Duration, Transporter) Discoverer { return new(testDiscovery) }

func (d *testDiscovery) BackFill(addrs ...string) {
	d.Lock()
	defer d.Unlock()
	d.backfill = append(d.backfill, addrs...)
}
func (d *testDiscovery) PoolCount() int { return 0 }
func (d *testDiscovery) RegisterSelf(p AddressablePeer) {
	d.Lock()
	defer d.Unlock()
	d.bad = append(d.bad, p.ConnectionAddr())
}
func (d *testDiscovery) GetFanOut() int {
	d.Lock()
	defer d.Unlock()
	return (len(d.connected) + len(d.backfill)) * 2 / 3
}
func (d *testDiscovery) NetworkSize() int {
	d.Lock()
	defer d.Unlock()
	return len(d.connected) + len(d.backfill)
}
func (d *testDiscovery) RegisterGood(AddressablePeer) {}
func (d *testDiscovery) RegisterConnected(p AddressablePeer) {
	d.Lock()
	defer d.Unlock()
	d.connected = append(d.connected, p.ConnectionAddr())
}
func (d *testDiscovery) UnregisterConnected(p AddressablePeer, force bool) {
	d.Lock()
	defer d.Unlock()
	d.unregistered = append(d.unregistered, p.ConnectionAddr())
}
func (d *testDiscovery) UnconnectedPeers() []string {
	d.Lock()
	defer d.Unlock()
	return d.unregistered
}
func (d *testDiscovery) RequestRemote(n int) {}
func (d *testDiscovery) BadPeers() []string {
	d.Lock()
	defer d.Unlock()
	return d.bad
}
func (d *testDiscovery) GoodPeers() []AddressWithCapabilities { return []AddressWithCapabilities{} }

var defaultMessageHandler = func(t *testing.T, msg *Message) {}

type localPeer struct {
	netaddr        net.TCPAddr
	server         *Server
	version        *payload.Version
	lastBlockIndex uint32
	handshaked     int32 // TODO: use atomic.Bool after #2626.
	isFullNode     bool
	t              *testing.T
	messageHandler func(t *testing.T, msg *Message)
	pingSent       int
	getAddrSent    int
	droppedWith    atomic.Value
}

func newLocalPeer(t *testing.T, s *Server) *localPeer {
	naddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	return &localPeer{
		t:              t,
		server:         s,
		netaddr:        *naddr,
		messageHandler: defaultMessageHandler,
	}
}

func (p *localPeer) ConnectionAddr() string {
	return p.netaddr.String()
}
func (p *localPeer) RemoteAddr() net.Addr {
	return &p.netaddr
}
func (p *localPeer) PeerAddr() net.Addr {
	return &p.netaddr
}
func (p *localPeer) StartProtocol() {}
func (p *localPeer) Disconnect(err error) {
	if p.droppedWith.Load() == nil {
		p.droppedWith.Store(err)
	}
	fmt.Println("peer dropped:", err)
	p.server.unregister <- peerDrop{p, err}
}

func (p *localPeer) BroadcastPacket(_ context.Context, m []byte) error {
	if len(m) == 0 {
		return errors.New("empty msg")
	}
	msg := &Message{}
	r := io.NewBinReaderFromBuf(m)
	for r.Len() > 0 {
		err := msg.Decode(r)
		if err == nil {
			p.messageHandler(p.t, msg)
		}
	}
	return nil
}
func (p *localPeer) EnqueueP2PMessage(msg *Message) error {
	return p.EnqueueHPMessage(msg)
}
func (p *localPeer) EnqueueP2PPacket(m []byte) error {
	return p.BroadcastPacket(context.TODO(), m)
}
func (p *localPeer) BroadcastHPPacket(ctx context.Context, m []byte) error {
	return p.BroadcastPacket(ctx, m)
}
func (p *localPeer) EnqueueHPMessage(msg *Message) error {
	p.messageHandler(p.t, msg)
	return nil
}
func (p *localPeer) EnqueueHPPacket(m []byte) error {
	return p.BroadcastPacket(context.TODO(), m)
}
func (p *localPeer) Version() *payload.Version {
	return p.version
}
func (p *localPeer) LastBlockIndex() uint32 {
	return p.lastBlockIndex
}
func (p *localPeer) HandleVersion(v *payload.Version) error {
	p.version = v
	return nil
}
func (p *localPeer) SendVersion() error {
	m, err := p.server.getVersionMsg(nil)
	if err != nil {
		return err
	}
	_ = p.EnqueueHPMessage(m)
	return nil
}
func (p *localPeer) SendVersionAck(m *Message) error {
	_ = p.EnqueueHPMessage(m)
	return nil
}
func (p *localPeer) HandleVersionAck() error {
	atomic.StoreInt32(&p.handshaked, 1)
	return nil
}
func (p *localPeer) SetPingTimer() {
	p.pingSent++
}
func (p *localPeer) HandlePing(ping *payload.Ping) error {
	p.lastBlockIndex = ping.LastBlockIndex
	return nil
}

func (p *localPeer) HandlePong(pong *payload.Ping) error {
	p.lastBlockIndex = pong.LastBlockIndex
	p.pingSent--
	return nil
}

func (p *localPeer) Handshaked() bool {
	return atomic.LoadInt32(&p.handshaked) != 0
}

func (p *localPeer) IsFullNode() bool {
	return p.isFullNode
}

func (p *localPeer) AddGetAddrSent() {
	p.getAddrSent++
}
func (p *localPeer) CanProcessAddr() bool {
	p.getAddrSent--
	return p.getAddrSent >= 0
}

func newTestServer(t *testing.T, serverConfig ServerConfig) *Server {
	return newTestServerWithCustomCfg(t, serverConfig, nil)
}

func newTestServerWithCustomCfg(t *testing.T, serverConfig ServerConfig, protocolCfg func(*config.Blockchain)) *Server {
	if len(serverConfig.Addresses) == 0 {
		// Normally it will be done by ApplicationConfiguration.GetAddresses().
		serverConfig.Addresses = []config.AnnounceableAddress{{Address: ":0"}}
	}
	s, err := newServerFromConstructors(serverConfig, fakechain.NewFakeChainWithCustomCfg(protocolCfg), new(fakechain.FakeStateSync), zaptest.NewLogger(t),
		newFakeTransp, newTestDiscovery)
	require.NoError(t, err)
	return s
}
