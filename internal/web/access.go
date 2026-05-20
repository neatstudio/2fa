package web

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
)

const DefaultPort = 23832

var defaultAllowPrefixes = []netip.Prefix{
	mustPrefix("127.0.0.0/8"),
	mustPrefix("::1/128"),
	mustPrefix("100.64.0.0/10"),
}

type BindTarget struct {
	Addr netip.Addr
	Port int
}

func (b BindTarget) ListenAddress() string {
	return net.JoinHostPort(b.Addr.String(), strconv.Itoa(b.Port))
}

func (b BindTarget) URL(token string) string {
	return "http://" + net.JoinHostPort(b.Addr.String(), strconv.Itoa(b.Port)) + "/?token=" + token
}

func DefaultBindTargets(port int, addrs []netip.Addr) []BindTarget {
	seen := map[netip.Addr]bool{}
	targets := []BindTarget{{Addr: netip.MustParseAddr("127.0.0.1"), Port: port}}
	seen[targets[0].Addr] = true
	for _, addr := range addrs {
		if !addr.Is4() || !mustPrefix("100.64.0.0/10").Contains(addr) || seen[addr] {
			continue
		}
		targets = append(targets, BindTarget{Addr: addr, Port: port})
		seen[addr] = true
	}
	return targets
}

func LocalInterfaceAddrs() ([]netip.Addr, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var addrs []netip.Addr
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, ifaceAddr := range ifaceAddrs {
			prefix, err := netip.ParsePrefix(ifaceAddr.String())
			if err == nil {
				addrs = append(addrs, prefix.Addr())
			}
		}
	}
	return addrs, nil
}

func ParseAllowPrefixes(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(defaultAllowPrefixes)+len(values))
	prefixes = append(prefixes, defaultAllowPrefixes...)
	for _, value := range values {
		prefix, err := parsePrefixOrIP(value)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

func RemoteAllowed(remoteAddr string, prefixes []netip.Prefix) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func ResolveBindTargets(addr string, port int, localAddrs []netip.Addr, explicitAllows []string) ([]BindTarget, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	if strings.TrimSpace(addr) == "" {
		return DefaultBindTargets(port, localAddrs), nil
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
		portText = strconv.Itoa(port)
	}
	parsedPort, err := strconv.Atoi(portText)
	if err != nil || parsedPort <= 0 || parsedPort > 65535 {
		return nil, fmt.Errorf("invalid listen address: %s", addr)
	}
	if isWildcardHost(host) && len(explicitAllows) == 0 {
		return nil, errors.New("wildcard --addr requires at least one --allow CIDR")
	}
	parsedAddr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return nil, fmt.Errorf("invalid listen address: %s", addr)
	}
	return []BindTarget{{Addr: parsedAddr, Port: parsedPort}}, nil
}

func isWildcardHost(host string) bool {
	host = strings.Trim(host, "[]")
	return host == "" || host == "0.0.0.0" || host == "::"
}

func parsePrefixOrIP(value string) (netip.Prefix, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Prefix{}, errors.New("allow prefix is required")
	}
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("invalid allow prefix %q: %w", value, err)
		}
		return prefix, nil
	}
	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid allow address %q: %w", value, err)
	}
	if addr.Is4() {
		return netip.PrefixFrom(addr, 32), nil
	}
	return netip.PrefixFrom(addr, 128), nil
}

func mustPrefix(value string) netip.Prefix {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		panic(err)
	}
	return prefix
}
