package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
)

const TailscaleInterfaceName = "tailscale0"

type interfaceSnapshot struct {
	name      string
	flags     net.Flags
	addresses []string
}

type interfaceLookup func(string) (interfaceSnapshot, error)
type tcpListener func(context.Context, string, string) (net.Listener, error)

func systemInterface(name string) (interfaceSnapshot, error) {
	device, err := net.InterfaceByName(name)
	if err != nil {
		return interfaceSnapshot{}, err
	}
	addresses, err := device.Addrs()
	if err != nil {
		return interfaceSnapshot{}, fmt.Errorf("read interface addresses: %w", err)
	}

	snapshot := interfaceSnapshot{name: device.Name, flags: device.Flags}
	for _, address := range addresses {
		snapshot.addresses = append(snapshot.addresses, address.String())
	}
	return snapshot, nil
}

func systemListen(ctx context.Context, network string, address string) (net.Listener, error) {
	var listenerConfig net.ListenConfig
	return listenerConfig.Listen(ctx, network, address)
}

// DiscoverTailscaleIPv4 returns the one suitable IPv4 address assigned to
// tailscale0. It never searches other interfaces.
func DiscoverTailscaleIPv4() (netip.Addr, error) {
	return discoverTailscaleIPv4(systemInterface)
}

func discoverTailscaleIPv4(lookup interfaceLookup) (netip.Addr, error) {
	snapshot, err := lookup(TailscaleInterfaceName)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("find %s: %w", TailscaleInterfaceName, err)
	}
	if snapshot.name != TailscaleInterfaceName {
		return netip.Addr{}, fmt.Errorf("interface lookup returned %q instead of %q", snapshot.name, TailscaleInterfaceName)
	}
	if snapshot.flags&net.FlagUp == 0 {
		return netip.Addr{}, fmt.Errorf("interface %s is not up", TailscaleInterfaceName)
	}

	var selected netip.Addr
	for _, rawAddress := range snapshot.addresses {
		prefix, parseErr := netip.ParsePrefix(rawAddress)
		if parseErr != nil {
			continue
		}
		address := prefix.Addr().Unmap()
		if !suitableTailscaleIPv4(address) {
			continue
		}
		if selected.IsValid() && address != selected {
			return netip.Addr{}, fmt.Errorf("interface %s has multiple suitable IPv4 addresses", TailscaleInterfaceName)
		}
		selected = address
	}
	if !selected.IsValid() {
		return netip.Addr{}, fmt.Errorf("interface %s has no suitable IPv4 address", TailscaleInterfaceName)
	}
	return selected, nil
}

func suitableTailscaleIPv4(address netip.Addr) bool {
	return address.IsValid() && address.Is4() && !address.IsUnspecified() &&
		!address.IsLoopback() && !address.IsMulticast() &&
		!address.IsLinkLocalUnicast()
}

// ListenTailscaleIPv4 opens a TCP listener only on the discovered tailscale0
// IPv4 address. Discovery or binding failures are returned without fallback.
func ListenTailscaleIPv4(ctx context.Context, port int) (net.Listener, error) {
	return listenTailscaleIPv4(ctx, port, systemInterface, systemListen)
}

func listenTailscaleIPv4(
	ctx context.Context,
	port int,
	lookup interfaceLookup,
	listen tcpListener,
) (net.Listener, error) {
	if port < 1024 || port > 65535 {
		return nil, fmt.Errorf("Tailscale listener port %d is outside the non-privileged range", port)
	}
	address, err := discoverTailscaleIPv4(lookup)
	if err != nil {
		return nil, err
	}
	wanted := net.JoinHostPort(address.String(), strconv.Itoa(port))
	listener, err := listen(ctx, "tcp4", wanted)
	if err != nil {
		return nil, fmt.Errorf("bind private Tailscale listener %s: %w", wanted, err)
	}
	if err := verifyListenerAddress(listener, address, port); err != nil {
		return nil, errors.Join(err, listener.Close())
	}
	return listener, nil
}

func verifyListenerAddress(listener net.Listener, wantedAddress netip.Addr, wantedPort int) error {
	actual, err := netip.ParseAddrPort(listener.Addr().String())
	if err != nil {
		return fmt.Errorf("inspect private listener address: %w", err)
	}
	if actual.Addr().Unmap() != wantedAddress || int(actual.Port()) != wantedPort {
		return fmt.Errorf("private listener opened on %s instead of %s", actual, netip.AddrPortFrom(wantedAddress, uint16(wantedPort)))
	}
	return nil
}
