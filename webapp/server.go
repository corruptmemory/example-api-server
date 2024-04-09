package webapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
)

type Server interface {
	Start() error
	Stop(ctx context.Context) error
	Wait()
}

type exampleAPIServer struct {
	port       uint
	address    string
	server     *http.Server
	ctx        context.Context
	cancelFunc context.CancelFunc
	listener   net.Listener
	wg         *sync.WaitGroup
	tlsconfig  *tls.Config
	handler    http.Handler
}

const keyServerAddr = "serverAddr"

func (s *exampleAPIServer) ServeHTTP(result http.ResponseWriter, request *http.Request) {
	s.handler.ServeHTTP(result, request)
}

func (s *exampleAPIServer) Start() (err error) {
	log.Printf("Starting server on %s:%v\n", s.address, s.port)
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.address, s.port),
		Handler: s,
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(s.ctx, keyServerAddr, l.Addr().String())
		},
		TLSConfig: s.tlsconfig,
	}
	s.listener, err = net.Listen("tcp", s.server.Addr)
	if err != nil {
		return
	}
	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)
	go func() {
		if s.tlsconfig == nil {
			log.Println("Starting HTTP server")
			err := s.server.Serve(s.listener)
			if err != http.ErrServerClosed {
				log.Printf("Error shutting down server: %v", err)
			}
		} else {
			log.Println("Starting HTTPS server")
			err := s.server.ServeTLS(s.listener, "", "")
			if err != http.ErrServerClosed {
				log.Printf("Error shutting down server: %v", err)
			}
		}
		s.wg.Done()
	}()
	return
}

func (s *exampleAPIServer) Wait() {
	s.wg.Wait()
}

func (s *exampleAPIServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func NewDefaultServer(port uint, handler http.Handler) Server {
	return &exampleAPIServer{
		port:    port,
		handler: handler,
	}
}

func NewServerWithAddress(address string, port uint, handler http.Handler) Server {
	return &exampleAPIServer{
		port:    port,
		address: address,
		handler: handler,
	}
}
