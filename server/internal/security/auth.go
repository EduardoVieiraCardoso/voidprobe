package security

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrMissingAuth   = errors.New("authorization header missing")
	ErrInvalidToken  = errors.New("invalid authentication token")
	ErrInvalidFormat = errors.New("invalid authorization format")
)

// TokenValidator valida tokens de autenticação
type TokenValidator struct {
	validTokens map[string]bool
}

// NewTokenValidator cria um novo validador
func NewTokenValidator(tokens []string) *TokenValidator {
	validator := &TokenValidator{
		validTokens: make(map[string]bool),
	}
	for _, token := range tokens {
		validator.validTokens[token] = true
	}
	return validator
}

// Validate verifica se o token é válido usando comparação de tempo constante
func (tv *TokenValidator) Validate(token string) error {
	if token == "" {
		return ErrMissingAuth
	}

	for validToken := range tv.validTokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1 {
			return nil
		}
	}

	return ErrInvalidToken
}

// UnaryInterceptor cria um interceptor para chamadas unárias
func (tv *TokenValidator) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if err := tv.authenticate(ctx); err != nil {
			log.Printf("Authentication failed for %s: %v", info.FullMethod, err)
			return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}
		return handler(ctx, req)
	}
}

// StreamInterceptor cria um interceptor para chamadas de stream
func (tv *TokenValidator) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := tv.authenticate(ss.Context()); err != nil {
			log.Printf("Authentication failed for %s: %v", info.FullMethod, err)
			return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}
		return handler(srv, ss)
	}
}

// authenticate extrai e valida o token do contexto
func (tv *TokenValidator) authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ErrMissingAuth
	}

	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return ErrMissingAuth
	}

	// Formato esperado: "Bearer <token>"
	authHeader := authHeaders[0]
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ErrInvalidFormat
	}

	return tv.Validate(parts[1])
}

// ClientAuthInterceptor adiciona token nas chamadas do cliente
type ClientAuthInterceptor struct {
	token string
}

// NewClientAuthInterceptor cria um interceptor de cliente
func NewClientAuthInterceptor(token string) *ClientAuthInterceptor {
	return &ClientAuthInterceptor{token: token}
}

// Unary adiciona autenticação em chamadas unárias
func (i *ClientAuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(i.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

// Stream adiciona autenticação em chamadas de stream
func (i *ClientAuthInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(i.attachToken(ctx), desc, cc, method, opts...)
	}
}

// attachToken adiciona o token ao contexto
func (i *ClientAuthInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+i.token)
}
