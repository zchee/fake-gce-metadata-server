// Copyright 2022 The gce-metadata-server Authors
// SPDX-License-Identifier: BSD-3-Clause

package fakemetadata

import (
	"math/rand"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/google/go-safeweb/safehttp"
	"github.com/google/go-safeweb/safehttp/plugins/staticheaders"
	"golang.org/x/net/http2"
)

var portRandSrc = rand.NewSource(time.Now().UTC().UnixNano())

// randomPort generates random port number and checks whether the it available (unused) port.
func randomPort(network string) string {
	var p string
	rnd := rand.New(portRandSrc)
	for {
		p = strconv.Itoa(rnd.Intn(55535) + 1000)
		if conn, err := net.Dial(network, p); err == nil {
			// not available
			conn.Close()
			continue
		}

		return p
	}
}

// Server represents a fake metadata server.
type Server struct {
	*safehttp.Server
}

// NewServer returns the new fake metadata server.
func NewServer() *Server {
	addr := net.JoinHostPort("localhost", randomPort("tcp4"))

	// inject MetadataHostEnv hostname
	os.Setenv(MetadataHostEnv, addr)

	muxConfig := safehttp.NewServeMuxConfig(nil)
	muxConfig.Intercept(metadataInterceptor{})
	muxConfig.Intercept(serverInterceptor{})
	muxConfig.Intercept(staticheaders.Interceptor{})

	mux := muxConfig.Mux()
	mux.Handle("/", safehttp.MethodGet, safehttp.HandlerFunc(rootHandler))
	mux.Handle("/computeMetadata", safehttp.MethodGet, safehttp.HandlerFunc(rootHandler))
	mux.Handle("/computeMetadata/", safehttp.MethodGet, safehttp.HandlerFunc(rootHandler))
	mux.Handle("/computeMetadata/v1", safehttp.MethodGet, safehttp.HandlerFunc(rootHandler))
	mux.Handle("/computeMetadata/v1/", safehttp.MethodGet, safehttp.HandlerFunc(rootHandler))
	(&ProjectHandler{}).RegisterHandlers(mux)
	(&InstanceHandler{}).RegisterHandlers(mux)

	return &Server{
		Server: &safehttp.Server{
			Addr: addr,
			Mux:  mux,
		},
	}
}

//go:linkname buildStd github.com/google/go-safeweb/safehttp.(*Server).buildStd
//go:noescape
func buildStd(s *safehttp.Server) error

func configureHTTP2Server(s *safehttp.Server, conf *http2.Server) *safehttp.Server {
	v := reflect.ValueOf(s).Elem()
	srv := v.FieldByName("srv")
	addr := unsafe.Pointer(srv.UnsafeAddr())
	(*http.Server)(addr).TLSConfig = s.TLSConfig

	http2.ConfigureServer((*http.Server)(addr), conf)

	return s
}

// ListenAndServe is a wrapper for https://golang.org/pkg/net/http/#Server.ListenAndServe
func (s *Server) ListenAndServe() error {
	if err := buildStd(s.Server); err != nil {
		return err
	}
	configureHTTP2Server(s.Server, &http2.Server{})

	return s.Server.ListenAndServe()
}

// ListenAndServeTLS is a wrapper for https://golang.org/pkg/net/http/#Server.ListenAndServeTLS
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	if err := buildStd(s.Server); err != nil {
		return err
	}
	configureHTTP2Server(s.Server, &http2.Server{})

	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

// Serve is a wrapper for https://golang.org/pkg/net/http/#Server.Serve
func (s *Server) Serve(l net.Listener) error {
	if err := buildStd(s.Server); err != nil {
		return err
	}
	configureHTTP2Server(s.Server, &http2.Server{})

	return s.Server.Serve(l)
}

// ServeTLS is a wrapper for https://golang.org/pkg/net/http/#Server.ServeTLS
func (s *Server) ServeTLS(l net.Listener, certFile, keyFile string) error {
	if err := buildStd(s.Server); err != nil {
		return err
	}
	configureHTTP2Server(s.Server, &http2.Server{})

	return s.Server.ServeTLS(l, certFile, keyFile)
}

// Addr returns the fake metadata server addr.
func (s *Server) Addr() string { return s.Server.Addr }
