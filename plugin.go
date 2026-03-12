package xffrefiner

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
)

// Config holds the middleware configuration.
type Config struct {
	// Depth is the index of the IP address to select from the X-Forwarded-For header.
	// Defaults to 0 (the first IP).
	Depth int `json:"depth,omitempty" yaml:"depth,omitempty" mapstructure:"depth,omitempty"`

	// OverrideRemoteAddr, if true, sets the request's RemoteAddr to the selected IP.
	// This helps in keeping only one IP in X-Forwarded-For when Traefik appends it.
	// Defaults to false.
	OverrideRemoteAddr bool `json:"overrideRemoteAddr,omitempty" yaml:"overrideRemoteAddr,omitempty" mapstructure:"overrideRemoteAddr,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Depth:              0,
		OverrideRemoteAddr: true,
	}
}

// Middleware is the XFF refiner middleware.
type Middleware struct {
	next               http.Handler
	depth              int
	overrideRemoteAddr bool
}

// New creates a new middleware instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	return &Middleware{
		next:               next,
		depth:              config.Depth,
		overrideRemoteAddr: config.OverrideRemoteAddr,
	}, nil
}

// ServeHTTP handles the HTTP request.
func (m *Middleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Collect all IPs from X-Forwarded-For
	xffValues := req.Header.Values("X-Forwarded-For")
	var ips []string
	for _, xff := range xffValues {
		for _, ipStr := range strings.Split(xff, ",") {
			trimmedIP := strings.TrimSpace(ipStr)
			if trimmedIP != "" {
				ips = append(ips, trimmedIP)
			}
		}
	}

	// Also include RemoteAddr as the last hop in the chain
	remoteAddr, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && remoteAddr != "" {
		ips = append(ips, remoteAddr)
	} else if req.RemoteAddr != "" {
		ips = append(ips, req.RemoteAddr)
	}

	if len(ips) > 0 {
		// Handle negative depth to count from the right (e.g., -1 is the last IP)
		index := m.depth
		if index < 0 {
			index = len(ips) + index
		}

		// Ensure the configured depth is within the bounds of the available IPs.
		if index >= 0 && index < len(ips) {
			selectedIP := ips[index]

			// Keep original XFF chain if needed (including RemoteAddr for completeness)
			fullXFF := strings.Join(ips, ", ")
			req.Header.Set("X-Original-Forwarded-For", fullXFF)

			// If override is enabled, update req.RemoteAddr to the selected IP
			if m.overrideRemoteAddr {
				// We should keep the port if possible
				_, port, err := net.SplitHostPort(req.RemoteAddr)
				if err == nil && port != "" {
					req.RemoteAddr = net.JoinHostPort(selectedIP, port)
				} else {
					req.RemoteAddr = selectedIP
				}
				// Clear X-Forwarded-For header so that Traefik's subsequent
				// append results in exactly one IP (the selected IP)
				req.Header.Del("X-Forwarded-For")
			} else {
				// Standard behavior: just set X-Forwarded-For to the selected IP
				req.Header.Set("X-Forwarded-For", selectedIP)
			}

			// Set additional headers as requested
			req.Header.Set("X-Forwarded-For-Proxy-Protocol", selectedIP)

			// Also setting X-Real-Ip for backward compatibility or general usefulness
			req.Header.Set("X-Real-Ip", selectedIP)
		}
	}
	m.next.ServeHTTP(rw, req)
}
