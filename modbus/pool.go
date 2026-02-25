package modbus

import (
	"fmt"
	"sync"
	"time"

	"github.com/goburrow/modbus"
)

type connection struct {
	handler *modbus.TCPClientHandler
	created time.Time
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

// Get returns a handler for the given address or creates a new one
func (p *modbusConnectionPool) Get(address string) (*modbus.TCPClientHandler, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Reuse existing connection
	if conn, ok := p.conns[address]; ok {
		if time.Since(conn.created) < p.ttl {
			return conn.handler, nil
		}
		// expired
		conn.handler.Close()
		delete(p.conns, address)
	}

	// Create new connection
	handler := modbus.NewTCPClientHandler(address)
	handler.Timeout = 10 * time.Second
	handler.IdleTimeout = p.ttl
	if err := handler.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	p.conns[address] = &connection{
		handler: handler,
		created: time.Now(),
	}
	return handler, nil
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
	defer p.mu.Unlock()
	now := time.Now()
	for addr, conn := range p.conns {
		if now.Sub(conn.created) > p.ttl {
			conn.handler.Close()
			delete(p.conns, addr)
		}
	}
}
