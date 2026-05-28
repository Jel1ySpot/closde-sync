package xray

import (
	"net"
	"strconv"
	"testing"
)

func TestFindAvailablePortReturnsStartWhenFree(t *testing.T) {
	host := "127.0.0.1"

	// Reserve a port, close it, then expect findAvailablePort to hand it back.
	probe, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatal(err)
	}
	start := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()

	got, err := findAvailablePort(host, start)
	if err != nil {
		t.Fatal(err)
	}
	if got != start {
		t.Fatalf("findAvailablePort() = %d, want %d", got, start)
	}
}

func TestFindAvailablePortShiftsPastBusyPort(t *testing.T) {
	host := "127.0.0.1"

	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	busy := listener.Addr().(*net.TCPAddr).Port

	got, err := findAvailablePort(host, busy)
	if err != nil {
		t.Fatal(err)
	}
	if got == busy {
		t.Fatalf("findAvailablePort() = %d, want a port other than the busy %d", got, busy)
	}
	if got < busy {
		t.Fatalf("findAvailablePort() = %d, want a port >= start %d", got, busy)
	}

	// The returned port must actually be bindable.
	check, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(got)))
	if err != nil {
		t.Fatalf("returned port %d is not bindable: %v", got, err)
	}
	_ = check.Close()
}
