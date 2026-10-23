package modbus

import (
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

type connection struct {
	handler *modbus.TCPClientHandler
	created time.Time
	// mu serializes access to handler: one TCP connection may be shared by
	// requests targeting different slave IDs, so callers must hold this for
	// the full request lifetime (from setting SlaveId through the actual I/O).
	mu sync.Mutex
}

type modbusConnectionPool struct {
	mu        sync.Mutex
	conns     map[string]*connection
	ttl       time.Duration
	cleanFreq time.Duration
	stopCh    chan struct{}
}

func newModbusConnectionPool(ttl, cleanFreq time.Duration) *modbusConnectionPool {
	pool := &modbusConnectionPool{
		conns:     make(map[string]*connection),
		ttl:       ttl,
		cleanFreq: cleanFreq,
		stopCh:    make(chan struct{}),
	}

	go pool.cleanupLoop()
	return pool
}

// Get returns a handler for the given address (creating one if needed) along with
// its mutex. Callers must hold the mutex for the full request, from setting
// handler.SlaveId through the actual I/O, since the connection may be shared.
// The handler is not connected, because dialing under the pool lock would block
// requests to every other address until the dial times out.
func (p *modbusConnectionPool) Get(address string) (*modbus.TCPClientHandler, *sync.Mutex) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Reuse existing connection
	if conn, ok := p.conns[address]; ok {
		if time.Since(conn.created) < p.ttl {
			return conn.handler, &conn.mu
		}
		// Close waits for an in-flight request, so it runs outside the pool lock.
		go conn.handler.Close()
		delete(p.conns, address)
	}

	// Create new connection
	handler := modbus.NewTCPClientHandler(address)
	handler.Timeout = 10 * time.Second
	handler.IdleTimeout = p.ttl

	conn := &connection{
		handler: handler,
		created: time.Now(),
	}
	p.conns[address] = conn
	return conn.handler, &conn.mu
}

// Close all connections
func (p *modbusConnectionPool) Close() {
	close(p.stopCh)
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, conn := range p.conns {
		conn.handler.Close()
		delete(p.conns, addr)
	}
}

// Background cleanup of expired connections
func (p *modbusConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(p.cleanFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanup()
		case <-p.stopCh:
			return
		}
	}
}

// Removes expired connections
func (p *modbusConnectionPool) cleanup() {
	p.mu.Lock()
	now := time.Now()
	var expired []*connection
	for addr, conn := range p.conns {
		if now.Sub(conn.created) > p.ttl {
			expired = append(expired, conn)
			delete(p.conns, addr)
		}
	}
	p.mu.Unlock()

	// Close waits for an in-flight request, so it runs outside the pool lock.
	for _, conn := range expired {
		conn.handler.Close()
	}
}
