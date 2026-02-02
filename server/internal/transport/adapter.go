package transport

import (
	"io"
	"sync"

	pb "github.com/voidprobe/server/api/proto"
)

// GrpcStream interface para o stream gRPC
type GrpcStream interface {
	Send(*pb.Chunk) error
	Recv() (*pb.Chunk, error)
}

// Adapter adapta um stream gRPC para io.ReadWriteCloser
// necessário para integração com yamux
type Adapter struct {
	Stream GrpcStream
	buffer []byte
	mu     sync.Mutex
	closed bool
}

// NewAdapter cria um novo adaptador
func NewAdapter(stream GrpcStream) *Adapter {
	return &Adapter{
		Stream: stream,
		buffer: make([]byte, 0),
	}
}

// Read implementa io.Reader
func (a *Adapter) Read(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return 0, io.EOF
	}

	// Se há dados no buffer, retorna eles primeiro
	if len(a.buffer) > 0 {
		n := copy(p, a.buffer)
		a.buffer = a.buffer[n:]
		return n, nil
	}

	// Recebe novos dados do stream
	msg, err := a.Stream.Recv()
	if err != nil {
		a.closed = true
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

// Write implementa io.Writer
func (a *Adapter) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return 0, io.ErrClosedPipe
	}

	// Cria uma cópia dos dados para evitar race conditions
	data := make([]byte, len(p))
	copy(data, p)

	// Envia pelo stream
	err := a.Stream.Send(&pb.Chunk{Data: data})
	if err != nil {
		a.closed = true
		return 0, err
	}

	return len(p), nil
}

// Close implementa io.Closer
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.closed = true
	a.buffer = nil

	return nil
}
