package app

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
)

func TestDiscoverTailscaleIPv4(t *testing.T) {
	t.Parallel()

	want := netip.MustParseAddr("100.64.0.23")
	got, err := discoverTailscaleIPv4(func(name string) (interfaceSnapshot, error) {
		if name != TailscaleInterfaceName {
			t.Fatalf("lookup name = %q", name)
		}
		return interfaceSnapshot{
			name:      TailscaleInterfaceName,
			flags:     net.FlagUp,
			addresses: []string{"fd7a:115c:a1e0::23/128", "100.64.0.23/32"},
		}, nil
	})
	if err != nil {
		t.Fatalf("discoverTailscaleIPv4() error = %v", err)
	}
	if got != want {
		t.Fatalf("discoverTailscaleIPv4() = %s, want %s", got, want)
	}
}

func TestDiscoverTailscaleIPv4FailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		snapshot  interfaceSnapshot
		lookupErr error
	}{
		{name: "missing interface", lookupErr: errors.New("not found")},
		{name: "wrong interface", snapshot: interfaceSnapshot{name: "eth0", flags: net.FlagUp, addresses: []string{"100.64.0.23/32"}}},
		{name: "down interface", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, addresses: []string{"100.64.0.23/32"}}},
		{name: "IPv6 only", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"fd7a:115c:a1e0::23/128"}}},
		{name: "wildcard", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"0.0.0.0/32"}}},
		{name: "localhost", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"127.0.0.1/8"}}},
		{name: "link local", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"169.254.1.2/16"}}},
		{name: "multiple IPv4 addresses", snapshot: interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"100.64.0.23/32", "100.64.0.24/32"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := discoverTailscaleIPv4(func(string) (interfaceSnapshot, error) {
				return test.snapshot, test.lookupErr
			})
			if err == nil {
				t.Fatal("discoverTailscaleIPv4() error = nil")
			}
		})
	}
}

func TestListenTailscaleIPv4UsesExactAddress(t *testing.T) {
	t.Parallel()

	fake := &recordingListener{address: stringAddress("100.64.0.23:8788")}
	listener, err := listenTailscaleIPv4(
		context.Background(),
		8788,
		func(string) (interfaceSnapshot, error) {
			return interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"100.64.0.23/32"}}, nil
		},
		func(_ context.Context, network string, address string) (net.Listener, error) {
			if network != "tcp4" {
				t.Fatalf("network = %q", network)
			}
			if address != "100.64.0.23:8788" {
				t.Fatalf("address = %q", address)
			}
			return fake, nil
		},
	)
	if err != nil {
		t.Fatalf("listenTailscaleIPv4() error = %v", err)
	}
	if listener != fake {
		t.Fatal("listenTailscaleIPv4() returned a different listener")
	}
}

func TestListenTailscaleIPv4DoesNotListenAfterDiscoveryFailure(t *testing.T) {
	t.Parallel()

	listenCalled := false
	_, err := listenTailscaleIPv4(
		context.Background(),
		8788,
		func(string) (interfaceSnapshot, error) { return interfaceSnapshot{}, errors.New("missing") },
		func(context.Context, string, string) (net.Listener, error) {
			listenCalled = true
			return nil, errors.New("must not be called")
		},
	)
	if err == nil {
		t.Fatal("listenTailscaleIPv4() error = nil")
	}
	if listenCalled {
		t.Fatal("listener was attempted after Tailscale discovery failed")
	}
}

func TestListenTailscaleIPv4ClosesMismatchedListener(t *testing.T) {
	t.Parallel()

	fake := &recordingListener{address: stringAddress("0.0.0.0:8788")}
	_, err := listenTailscaleIPv4(
		context.Background(),
		8788,
		func(string) (interfaceSnapshot, error) {
			return interfaceSnapshot{name: TailscaleInterfaceName, flags: net.FlagUp, addresses: []string{"100.64.0.23/32"}}, nil
		},
		func(context.Context, string, string) (net.Listener, error) { return fake, nil },
	)
	if err == nil {
		t.Fatal("listenTailscaleIPv4() error = nil")
	}
	if !fake.closed {
		t.Fatal("mismatched listener was not closed")
	}
}

type stringAddress string

func (address stringAddress) Network() string { return "tcp" }
func (address stringAddress) String() string  { return string(address) }

type recordingListener struct {
	address net.Addr
	closed  bool
}

func (listener *recordingListener) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}

func (listener *recordingListener) Close() error {
	listener.closed = true
	return nil
}

func (listener *recordingListener) Addr() net.Addr { return listener.address }
