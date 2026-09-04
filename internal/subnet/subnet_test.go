package subnet

import (
	"math/big"
	"strings"
	"testing"
)

func TestGetDetailsIPv4(t *testing.T) {
	tests := []struct {
		name       string
		cidr       string
		wantCIDR   string
		wantFirst  string
		wantLast   string
		wantBcast  string
		wantUsable int64
		wantTotal  int64
	}{
		{"/24", "192.168.1.10/24", "192.168.1.0/24", "192.168.1.1", "192.168.1.254", "192.168.1.255", 254, 256},
		{"/30", "10.0.0.5/30", "10.0.0.4/30", "10.0.0.5", "10.0.0.6", "10.0.0.7", 2, 4},
		{"/31", "10.0.0.4/31", "10.0.0.4/31", "10.0.0.4", "10.0.0.5", "10.0.0.5", 2, 2},
		{"/32", "10.0.0.4/32", "10.0.0.4/32", "10.0.0.4", "10.0.0.4", "10.0.0.4", 1, 1},
		{"/16", "172.16.5.5/16", "172.16.0.0/16", "172.16.0.1", "172.16.255.254", "172.16.255.255", 65534, 65536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := GetDetails(tt.cidr)
			if err != nil {
				t.Fatalf("GetDetails(%q) error: %v", tt.cidr, err)
			}
			if d.CIDR != tt.wantCIDR {
				t.Errorf("CIDR = %q, want %q", d.CIDR, tt.wantCIDR)
			}
			if d.IsIPv6 {
				t.Errorf("IsIPv6 = true, want false")
			}
			if d.FirstUsable.String() != tt.wantFirst {
				t.Errorf("FirstUsable = %q, want %q", d.FirstUsable.String(), tt.wantFirst)
			}
			if d.LastUsable.String() != tt.wantLast {
				t.Errorf("LastUsable = %q, want %q", d.LastUsable.String(), tt.wantLast)
			}
			if d.Broadcast.String() != tt.wantBcast {
				t.Errorf("Broadcast = %q, want %q", d.Broadcast.String(), tt.wantBcast)
			}
			if d.UsableHosts.Cmp(big.NewInt(tt.wantUsable)) != 0 {
				t.Errorf("UsableHosts = %s, want %d", d.UsableHosts.String(), tt.wantUsable)
			}
			if d.TotalAddresses.Cmp(big.NewInt(tt.wantTotal)) != 0 {
				t.Errorf("TotalAddresses = %s, want %d", d.TotalAddresses.String(), tt.wantTotal)
			}
		})
	}
}

func TestGetDetailsIPv6(t *testing.T) {
	d, err := GetDetails("2001:db8::/64")
	if err != nil {
		t.Fatalf("GetDetails error: %v", err)
	}
	if !d.IsIPv6 {
		t.Errorf("IsIPv6 = false, want true")
	}
	if d.CIDR != "2001:db8::/64" {
		t.Errorf("CIDR = %q, want 2001:db8::/64", d.CIDR)
	}
	if d.FirstUsable.String() != "2001:db8::" {
		t.Errorf("FirstUsable = %q, want 2001:db8::", d.FirstUsable.String())
	}
	if d.LastUsable.String() != "2001:db8::ffff:ffff:ffff:ffff" {
		t.Errorf("LastUsable = %q, want 2001:db8::ffff:ffff:ffff:ffff", d.LastUsable.String())
	}
	wantTotal := new(big.Int).Lsh(big.NewInt(1), 64)
	if d.TotalAddresses.Cmp(wantTotal) != 0 {
		t.Errorf("TotalAddresses = %s, want %s", d.TotalAddresses.String(), wantTotal.String())
	}
	if d.UsableHosts.Cmp(wantTotal) != 0 {
		t.Errorf("UsableHosts = %s, want %s", d.UsableHosts.String(), wantTotal.String())
	}

	d128, err := GetDetails("2001:db8::1/128")
	if err != nil {
		t.Fatalf("GetDetails /128 error: %v", err)
	}
	if d128.UsableHosts.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("UsableHosts for /128 = %s, want 1", d128.UsableHosts.String())
	}
}

func TestGetDetailsInvalid(t *testing.T) {
	cases := []string{"", "not-a-cidr", "192.168.1.1", "192.168.1.1/33", "2001:db8::/129"}
	for _, c := range cases {
		if _, err := GetDetails(c); err == nil {
			t.Errorf("GetDetails(%q) expected error, got nil", c)
		}
	}
}

func TestSplitByPrefix(t *testing.T) {
	subs, err := SplitByPrefix("192.168.0.0/24", 26)
	if err != nil {
		t.Fatalf("SplitByPrefix error: %v", err)
	}
	want := []string{"192.168.0.0/26", "192.168.0.64/26", "192.168.0.128/26", "192.168.0.192/26"}
	if len(subs) != len(want) {
		t.Fatalf("got %d subnets, want %d", len(subs), len(want))
	}
	for i, w := range want {
		if subs[i].String() != w {
			t.Errorf("subnet[%d] = %q, want %q", i, subs[i].String(), w)
		}
	}
}

func TestSplitByPrefixIPv6(t *testing.T) {
	subs, err := SplitByPrefix("2001:db8::/48", 50)
	if err != nil {
		t.Fatalf("SplitByPrefix error: %v", err)
	}
	if len(subs) != 4 {
		t.Fatalf("got %d subnets, want 4", len(subs))
	}
	if subs[0].String() != "2001:db8::/50" {
		t.Errorf("subnet[0] = %q, want 2001:db8::/50", subs[0].String())
	}
}

func TestSplitByPrefixErrors(t *testing.T) {
	if _, err := SplitByPrefix("bad-cidr", 26); err == nil {
		t.Error("expected error for invalid CIDR")
	}
	if _, err := SplitByPrefix("192.168.0.0/24", 20); err == nil {
		t.Error("expected error when new prefix smaller than base")
	}
	if _, err := SplitByPrefix("192.168.0.0/24", 33); err == nil {
		t.Error("expected error when new prefix exceeds address width")
	}
	if _, err := SplitByPrefix("10.0.0.0/8", 32); err == nil {
		t.Error("expected error for too many subnets")
	}
}

func TestSplitCount(t *testing.T) {
	tests := []struct {
		count     int
		wantLen   int
		wantFirst string
	}{
		{1, 1, "10.0.0.0/24"},
		{2, 2, "10.0.0.0/25"},
		{3, 4, "10.0.0.0/26"},
		{4, 4, "10.0.0.0/26"},
		{5, 8, "10.0.0.0/27"},
	}
	for _, tt := range tests {
		subs, err := SplitCount("10.0.0.0/24", tt.count)
		if err != nil {
			t.Fatalf("SplitCount(%d) error: %v", tt.count, err)
		}
		if len(subs) != tt.wantLen {
			t.Errorf("SplitCount(%d) len = %d, want %d", tt.count, len(subs), tt.wantLen)
		}
		if subs[0].String() != tt.wantFirst {
			t.Errorf("SplitCount(%d) first = %q, want %q", tt.count, subs[0].String(), tt.wantFirst)
		}
	}
}

func TestSplitCountErrors(t *testing.T) {
	if _, err := SplitCount("10.0.0.0/24", 0); err == nil {
		t.Error("expected error for count < 1")
	}
	if _, err := SplitCount("bad-cidr", 2); err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestPlanVLSM(t *testing.T) {
	reqs := []Requirement{
		{Name: "Sales", Hosts: 50},
		{Name: "Engineering", Hosts: 100},
		{Name: "Guest", Hosts: 10},
		{Name: "PointToPoint", Hosts: 2},
	}
	allocs, err := PlanVLSM("192.168.1.0/24", reqs)
	if err != nil {
		t.Fatalf("PlanVLSM error: %v", err)
	}
	if len(allocs) != len(reqs) {
		t.Fatalf("got %d allocations, want %d", len(allocs), len(reqs))
	}

	// Order must match input order, not allocation order.
	for i, a := range allocs {
		if a.Name != reqs[i].Name {
			t.Errorf("allocation[%d].Name = %q, want %q", i, a.Name, reqs[i].Name)
		}
		if a.UsableHosts.Cmp(big.NewInt(int64(reqs[i].Hosts))) < 0 {
			t.Errorf("allocation %q usable hosts %s < requested %d", a.Name, a.UsableHosts.String(), reqs[i].Hosts)
		}
	}

	// Engineering (100 hosts) needs a /25 (126 usable), should start at the
	// base network address since it is allocated first (largest block).
	if allocs[1].CIDR != "192.168.1.0/25" {
		t.Errorf("Engineering CIDR = %q, want 192.168.1.0/25", allocs[1].CIDR)
	}
	// Sales (50 hosts) needs a /26 (62 usable), next aligned block after /25.
	if allocs[0].CIDR != "192.168.1.128/26" {
		t.Errorf("Sales CIDR = %q, want 192.168.1.128/26", allocs[0].CIDR)
	}
}

func TestPlanVLSMOutOfSpace(t *testing.T) {
	reqs := []Requirement{
		{Name: "Huge", Hosts: 1000},
	}
	if _, err := PlanVLSM("192.168.1.0/24", reqs); err == nil {
		t.Error("expected error when requirement exceeds base network capacity")
	}

	reqs2 := []Requirement{
		{Name: "A", Hosts: 100},
		{Name: "B", Hosts: 100},
		{Name: "C", Hosts: 100},
	}
	if _, err := PlanVLSM("192.168.1.0/24", reqs2); err == nil {
		t.Error("expected error when combined requirements exceed base network capacity")
	}
}

func TestPlanVLSMInvalid(t *testing.T) {
	if _, err := PlanVLSM("bad-cidr", []Requirement{{Name: "A", Hosts: 1}}); err == nil {
		t.Error("expected error for invalid CIDR")
	}
	if _, err := PlanVLSM("10.0.0.0/24", []Requirement{{Name: "A", Hosts: 0}}); err == nil {
		t.Error("expected error for zero hosts requirement")
	}
	if _, err := PlanVLSM("10.0.0.0/24", []Requirement{{Name: "A", Hosts: -5}}); err == nil {
		t.Error("expected error for negative hosts requirement")
	}
}

func TestPlanVLSMIPv6(t *testing.T) {
	reqs := []Requirement{
		{Name: "Servers", Hosts: 500},
		{Name: "Clients", Hosts: 20},
	}
	allocs, err := PlanVLSM("2001:db8::/56", reqs)
	if err != nil {
		t.Fatalf("PlanVLSM error: %v", err)
	}
	for _, a := range allocs {
		if !strings.Contains(a.CIDR, "2001:db8:") {
			t.Errorf("allocation CIDR %q does not look like it is within the base network", a.CIDR)
		}
	}
}

func TestPlanVLSMEmpty(t *testing.T) {
	allocs, err := PlanVLSM("10.0.0.0/24", nil)
	if err != nil {
		t.Fatalf("PlanVLSM error: %v", err)
	}
	if len(allocs) != 0 {
		t.Errorf("got %d allocations, want 0", len(allocs))
	}
}

func TestHostBitsNeeded(t *testing.T) {
	tests := []struct {
		hosts  int
		isIPv6 bool
		want   int
	}{
		{1, false, 0},
		{2, false, 1},
		{3, false, 3},
		{6, false, 3},
		{14, false, 4},
		{0, false, 0},
		{-1, false, 0},
		{1, true, 0},
		{2, true, 1},
		{3, true, 2},
	}
	for _, tt := range tests {
		got := hostBitsNeeded(tt.hosts, tt.isIPv6)
		if got != tt.want {
			t.Errorf("hostBitsNeeded(%d, %v) = %d, want %d", tt.hosts, tt.isIPv6, got, tt.want)
		}
	}
}

func TestBitsForCount(t *testing.T) {
	tests := []struct {
		count int
		want  int
	}{
		{1, 0},
		{2, 1},
		{3, 2},
		{4, 2},
		{5, 3},
		{8, 3},
		{9, 4},
	}
	for _, tt := range tests {
		got := bitsForCount(tt.count)
		if got != tt.want {
			t.Errorf("bitsForCount(%d) = %d, want %d", tt.count, got, tt.want)
		}
	}
}

func TestUsableHostsForBits(t *testing.T) {
	if usableHostsForBits(0, false).Cmp(big.NewInt(1)) != 0 {
		t.Error("usableHostsForBits(0, false) want 1")
	}
	if usableHostsForBits(1, false).Cmp(big.NewInt(2)) != 0 {
		t.Error("usableHostsForBits(1, false) want 2")
	}
	if usableHostsForBits(8, false).Cmp(big.NewInt(254)) != 0 {
		t.Error("usableHostsForBits(8, false) want 254")
	}
	if usableHostsForBits(8, true).Cmp(big.NewInt(256)) != 0 {
		t.Error("usableHostsForBits(8, true) want 256")
	}
}

func TestAddrBigIntRoundTrip(t *testing.T) {
	d, err := GetDetails("203.0.113.5/32")
	if err != nil {
		t.Fatal(err)
	}
	if d.Network.String() != "203.0.113.5" {
		t.Errorf("Network = %q, want 203.0.113.5", d.Network.String())
	}

	d6, err := GetDetails("2001:db8::abcd/128")
	if err != nil {
		t.Fatal(err)
	}
	if d6.Network.String() != "2001:db8::abcd" {
		t.Errorf("Network = %q, want 2001:db8::abcd", d6.Network.String())
	}
}
