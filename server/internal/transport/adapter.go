// Package transport adapta streams gRPC para conexões compatíveis com yamux.
package transport

import (
	"io"
	"sync"

	pb "github.com/voidprobe/server/api/proto"
)

// GrpcStream interface para o stream gRPC.
type GrpcStream interface {
	Send(*pb.Chunk) error
	Recv() (*pb.Chunk, error)
}

// Adapter adapta um stream gRPC para io.ReadWriteCloser,
// necessário para integração com yamux.
type Adapter struct {
	Stream   GrpcStream
	buffer   []byte
	readMu   sync.Mutex // Mutex separado para leitura
	writeMu  sync.Mutex // Mutex separado para escrita
	closed   bool
	closedMu sync.RWMutex
}

// NewAdapter cria um novo adaptador.
func NewAdapter(stream GrpcStream) *Adapter {
	return &Adapter{
		Stream: stream,
		buffer: make([]byte, 0),
	}
}

// isClosed verifica se o adapter está fechado.
func (a *Adapter) isClosed() bool {
	a.closedMu.RLock()
	defer a.closedMu.RUnlock()
	return a.closed
}

// setClosed marca o adapter como fechado.
func (a *Adapter) setClosed() {
	a.closedMu.Lock()
	defer a.closedMu.Unlock()
	a.closed = true
}

// Read implementa io.Reader.
func (a *Adapter) Read(p []byte) (int, error) {
	a.readMu.Lock()
	defer a.readMu.Unlock()

	if a.isClosed() {
		return 0, io.EOF
	}

	// Se há dados no buffer, retorna eles primeiro
	if len(a.buffer) > 0 {
		n := copy(p, a.buffer)
		a.buffer = a.buffer[n:]
		return n, nil
	}

	// Recebe novos dados do stream (NÃO bloqueia escrita)
	msg, err := a.Stream.Recv()
	if err != nil {
		a.setClosed()
		return 0, err
	}

	// Copia dados para o buffer de leitura
	n := copy(p, msg.Data)

	// Se não couber tudo, guarda o resto no buffer
	if n < len(msg.Data) {
		a.buffer = make([]byte, len(msg.Data)-n)
		copy(a.buffer, msg.Data[n:])
	}

	return n, nil
}

// Write implementa io.Writer.
func (a *Adapter) Write(p []byte) (int, error) {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()

	if a.isClosed() {
		return 0, io.ErrClosedPipe
	}

	// Cria uma cópia dos dados para evitar race conditions
	data := make([]byte, len(p))
	copy(data, p)

	// Envia pelo stream (NÃO bloqueia leitura)
	err := a.Stream.Send(&pb.Chunk{Data: data})
	if err != nil {
		a.setClosed()
		return 0, err
	}

	return len(p), nil
}

// Close implementa io.Closer.
func (a *Adapter) Close() error {
	a.setClosed()

	a.readMu.Lock()
	a.buffer = nil
	a.readMu.Unlock()

	return nil
}
