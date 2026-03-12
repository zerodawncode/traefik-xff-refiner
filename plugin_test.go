package traefik_xff_refiner

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXFFRefiner(t *testing.T) {
	t.Run("default configuration uses overrideRemoteAddr=true", func(t *testing.T) {
		cfg := CreateConfig()
		if cfg.OverrideRemoteAddr != true {
			t.Errorf("expected default OverrideRemoteAddr to be true, got %v", cfg.OverrideRemoteAddr)
		}
	})

	tests := []struct {
		name               string
		xffHeader          string
		xffHeaders         []string
		configDepth        int
		expectedIP         string
		expectHeaderSet    bool
		overrideRemoteAddr bool
	}{
		{
			name:            "default depth (0), multiple IPs in X-Forwarded-For",
			xffHeader:       "203.0.113.5, 10.0.0.1, 192.168.1.1",
			configDepth:     0,
			expectedIP:      "203.0.113.5",
			expectHeaderSet: true,
		},
		{
			name:            "depth 1, multiple IPs in X-Forwarded-For",
			xffHeader:       "203.0.113.5, 10.0.0.1, 192.168.1.1",
			configDepth:     1,
			expectedIP:      "10.0.0.1",
			expectHeaderSet: true,
		},
		{
			name:            "depth 2, multiple IPs in X-Forwarded-For",
			xffHeader:       "203.0.113.5, 10.0.0.1, 192.168.1.1",
			configDepth:     2,
			expectedIP:      "192.168.1.1",
			expectHeaderSet: true,
		},
		{
			name:            "no X-Forwarded-For header",
			xffHeader:       "",
			configDepth:     0,
			expectedIP:      "192.0.2.1", // Default httptest RemoteAddr
			expectHeaderSet: true,
		},
		{
			name:            "multiple X-Forwarded-For header lines",
			xffHeaders:      []string{"203.0.113.5, 10.0.0.1", "192.168.1.1"},
			configDepth:     2,
			expectedIP:      "192.168.1.1",
			expectHeaderSet: true,
		},
		{
			name:            "negative depth (-1), get last IP (RemoteAddr)",
			xffHeader:       "203.0.113.5, 10.0.0.1",
			configDepth:     -1,
			expectedIP:      "192.0.2.1", // RemoteAddr
			expectHeaderSet: true,
		},
		{
			name:            "negative depth (-2), get second to last IP",
			xffHeader:       "203.0.113.5, 10.0.0.1",
			configDepth:     -2,
			expectedIP:      "10.0.0.1",
			expectHeaderSet: true,
		},
		{
			name:               "override remote addr enabled",
			xffHeader:          "203.0.113.5, 10.0.0.1",
			configDepth:        0,
			expectedIP:         "203.0.113.5",
			expectHeaderSet:    true,
			overrideRemoteAddr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			originalRemoteAddr, _, _ := net.SplitHostPort(req.RemoteAddr)
			if len(tt.xffHeaders) > 0 {
				for _, h := range tt.xffHeaders {
					req.Header.Add("X-Forwarded-For", h)
				}
			} else if tt.xffHeader != "" {
				req.Header.Set("X-Forwarded-For", tt.xffHeader)
			}

			rec := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectHeaderSet {
					// Check X-Forwarded-For (should be single IP OR empty if overriding)
					xffValues := r.Header.Values("X-Forwarded-For")
					if tt.overrideRemoteAddr {
						if len(xffValues) != 0 {
							t.Errorf("expected 0 X-Forwarded-For headers when overriding, got %d (%v)", len(xffValues), xffValues)
						}
					} else {
						if len(xffValues) != 1 {
							t.Errorf("expected 1 X-Forwarded-For header, got %d", len(xffValues))
						}
						if len(xffValues) > 0 && xffValues[0] != tt.expectedIP {
							t.Errorf("expected X-Forwarded-For = %q, got %q", tt.expectedIP, xffValues[0])
						}
					}

					// Check X-Original-Forwarded-For
					xoff := r.Header.Get("X-Original-Forwarded-For")
					expectedXOFF := tt.xffHeader
					if len(tt.xffHeaders) > 0 {
						expectedXOFF = strings.Join(tt.xffHeaders, ", ")
					}
					// Add RemoteAddr which is added by the middleware now
					if expectedXOFF != "" {
						expectedXOFF = expectedXOFF + ", " + originalRemoteAddr
					} else {
						expectedXOFF = originalRemoteAddr
					}

					if xoff != expectedXOFF {
						t.Errorf("expected X-Original-Forwarded-For = %q, got %q", expectedXOFF, xoff)
					}

					// Check X-Forwarded-For-Proxy-Protocol
					xffpp := r.Header.Get("X-Forwarded-For-Proxy-Protocol")
					if xffpp != tt.expectedIP {
						t.Errorf("expected X-Forwarded-For-Proxy-Protocol = %q, got %q", tt.expectedIP, xffpp)
					}

					// Check X-Real-Ip
					xrip := r.Header.Get("X-Real-Ip")
					if xrip != tt.expectedIP {
						t.Errorf("expected X-Real-Ip = %q, got %q", tt.expectedIP, xrip)
					}
				} else {
					if _, ok := r.Header["X-Original-Forwarded-For"]; ok {
						t.Errorf("X-Original-Forwarded-For should not be set")
					}
				}
			})

			cfg := CreateConfig()
			cfg.Depth = tt.configDepth
			cfg.OverrideRemoteAddr = tt.overrideRemoteAddr
			mw, err := New(context.Background(), next, cfg, "test-xff-refiner")
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			mw.ServeHTTP(rec, req)

			if tt.overrideRemoteAddr {
				expectedRemoteAddr := tt.expectedIP + ":1234" // Default port in httptest
				if req.RemoteAddr != expectedRemoteAddr {
					t.Errorf("expected RemoteAddr = %q, got %q", expectedRemoteAddr, req.RemoteAddr)
				}
			}
		})
	}
}
