// Package subnet implements IPv4 and IPv6 subnet-planning arithmetic:
// inspecting a network, splitting it into equal-sized subnets, and
// allocating variable-length subnets (VLSM) to a list of host requirements.
package subnet

import (
	"fmt"
	"math/big"
	"net/netip"
	"sort"
)

// Details describes a single network/subnet.
type Details struct {
	CIDR           string
	Network        netip.Addr
	Prefix         int
	Bits           int // 32 for IPv4, 128 for IPv6
	IsIPv6         bool
	FirstUsable    netip.Addr
	LastUsable     netip.Addr
	Broadcast      netip.Addr // IPv4 only; zero value for IPv6
	TotalAddresses *big.Int
	UsableHosts    *big.Int
}

// Requirement is a named request for a subnet capable of holding at least
// Hosts usable host addresses.
type Requirement struct {
	Name  string
	Hosts int
}

// Allocation is the result of fitting a Requirement into a slice of a base
// network.
type Allocation struct {
	Name           string
	CIDR           string
	Network        netip.Addr
	Prefix         int
	HostsRequested int
	UsableHosts    *big.Int
}

func addrToBigInt(a netip.Addr) *big.Int {
	if a.Is4() {
		b := a.As4()
		return new(big.Int).SetBytes(b[:])
	}
	b := a.As16()
	return new(big.Int).SetBytes(b[:])
}

func bigIntToAddr(v *big.Int, isIPv6 bool) netip.Addr {
	size := 4
	if isIPv6 {
		size = 16
	}
	buf := make([]byte, size)
	vb := v.Bytes() // minimal big-endian form; always <= size for in-range values
	copy(buf[size-len(vb):], vb)
	if isIPv6 {
		var a16 [16]byte
		copy(a16[:], buf)
		return netip.AddrFrom16(a16)
	}
	var a4 [4]byte
	copy(a4[:], buf)
	return netip.AddrFrom4(a4)
}

// powerOfTwo returns 2^n as a *big.Int.
func powerOfTwo(n int) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), uint(n))
}

// GetDetails parses a CIDR string and returns its network details.
func GetDetails(cidr string) (Details, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return Details{}, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	network := prefix.Masked()
	addr := network.Addr()
	bits := 32
	if addr.Is6() && !addr.Is4In6() {
		bits = 128
	}
	isIPv6 := bits == 128
	hostBits := bits - network.Bits()
	total := powerOfTwo(hostBits)

	networkInt := addrToBigInt(addr)
	lastInt := new(big.Int).Add(networkInt, total)
	lastInt.Sub(lastInt, big.NewInt(1))
	lastAddr := bigIntToAddr(lastInt, isIPv6)

	d := Details{
		CIDR:           network.String(),
		Network:        addr,
		Prefix:         network.Bits(),
		Bits:           bits,
		IsIPv6:         isIPv6,
		TotalAddresses: total,
	}

	if !isIPv6 {
		d.Broadcast = lastAddr
		switch {
		case hostBits == 0: // /32
			d.FirstUsable = addr
			d.LastUsable = addr
			d.UsableHosts = big.NewInt(1)
		case hostBits == 1: // /31, RFC 3021 point-to-point
			d.FirstUsable = addr
			d.LastUsable = lastAddr
			d.UsableHosts = big.NewInt(2)
		default:
			firstInt := new(big.Int).Add(networkInt, big.NewInt(1))
			usableLastInt := new(big.Int).Sub(lastInt, big.NewInt(1))
			d.FirstUsable = bigIntToAddr(firstInt, false)
			d.LastUsable = bigIntToAddr(usableLastInt, false)
			d.UsableHosts = new(big.Int).Sub(total, big.NewInt(2))
		}
	} else {
		// IPv6 has no broadcast address; every address in the subnet,
		// including the network address, is usable.
		d.FirstUsable = addr
		d.LastUsable = lastAddr
		d.UsableHosts = new(big.Int).Set(total)
	}

	return d, nil
}

// SplitByPrefix splits a base CIDR into all subnets of the given new prefix
// length. newPrefix must be >= the base prefix length and <= the address
// family width.
func SplitByPrefix(cidr string, newPrefix int) ([]netip.Prefix, error) {
	base, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	base = base.Masked()
	bits := 32
	if base.Addr().Is6() && !base.Addr().Is4In6() {
		bits = 128
	}
	isIPv6 := bits == 128
	if newPrefix < base.Bits() {
		return nil, fmt.Errorf("new prefix /%d is larger than base prefix /%d", newPrefix, base.Bits())
	}
	if newPrefix > bits {
		return nil, fmt.Errorf("new prefix /%d exceeds address width /%d", newPrefix, bits)
	}

	subnetHostBits := bits - newPrefix
	blockSize := powerOfTwo(subnetHostBits)
	baseHostBits := bits - base.Bits()
	count := new(big.Int).Lsh(big.NewInt(1), uint(baseHostBits-subnetHostBits))

	if !count.IsInt64() || count.Int64() > 1_000_000 {
		return nil, fmt.Errorf("split would produce too many subnets (%s)", count.String())
	}

	n := int(count.Int64())
	result := make([]netip.Prefix, 0, n)
	cur := addrToBigInt(base.Addr())
	for i := 0; i < n; i++ {
		a := bigIntToAddr(cur, isIPv6)
		result = append(result, netip.PrefixFrom(a, newPrefix))
		cur = new(big.Int).Add(cur, blockSize)
	}
	return result, nil
}

// SplitCount splits a base CIDR into at least `count` equal-sized subnets,
// rounding up to the next power of two.
func SplitCount(cidr string, count int) ([]netip.Prefix, error) {
	if count < 1 {
		return nil, fmt.Errorf("count must be at least 1, got %d", count)
	}
	base, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	extraBits := bitsForCount(count)
	return SplitByPrefix(base.String(), base.Masked().Bits()+extraBits)
}

// bitsForCount returns the smallest n such that 2^n >= count.
func bitsForCount(count int) int {
	n := 0
	for (int64(1) << uint(n)) < int64(count) {
		n++
	}
	return n
}

// hostBitsNeeded returns the smallest number of host bits such that a
// subnet of that size can provide `hosts` usable addresses for the given
// address family.
func hostBitsNeeded(hosts int, isIPv6 bool) int {
	if hosts < 1 {
		hosts = 1
	}
	n := 0
	for {
		total := int64(1) << uint(n)
		var usable int64
		if isIPv6 {
			usable = total
		} else {
			switch {
			case n == 0:
				usable = 1
			case n == 1:
				usable = 2
			default:
				usable = total - 2
			}
		}
		if usable >= int64(hosts) {
			return n
		}
		n++
	}
}

// PlanVLSM allocates a slice of the base network to each Requirement using
// variable-length subnet masking: requirements are packed largest-first so
// that each block naturally falls on a correctly aligned boundary, but the
// returned allocations preserve the original input order.
func PlanVLSM(cidr string, reqs []Requirement) ([]Allocation, error) {
	base, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	base = base.Masked()
	bits := 32
	if base.Addr().Is6() && !base.Addr().Is4In6() {
		bits = 128
	}
	isIPv6 := bits == 128

	type indexed struct {
		idx      int
		req      Requirement
		hostBits int
	}
	items := make([]indexed, len(reqs))
	for i, r := range reqs {
		if r.Hosts < 1 {
			return nil, fmt.Errorf("requirement %q: hosts must be >= 1", r.Name)
		}
		items[i] = indexed{idx: i, req: r, hostBits: hostBitsNeeded(r.Hosts, isIPv6)}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].hostBits > items[j].hostBits
	})

	networkStart := addrToBigInt(base.Addr())
	hostBitsAvail := bits - base.Bits()
	networkSize := powerOfTwo(hostBitsAvail)
	networkEnd := new(big.Int).Add(networkStart, networkSize) // exclusive

	// items are sorted by hostBits descending, so blockSize is non-increasing
	// and each successive block start (always a running sum of larger,
	// power-of-two-aligned blocks) is already a multiple of the next block's
	// size — no extra alignment padding is ever needed.
	cur := new(big.Int).Set(networkStart)
	results := make([]Allocation, len(reqs))
	for _, it := range items {
		blockSize := powerOfTwo(it.hostBits)
		blockEnd := new(big.Int).Add(cur, blockSize)
		if blockEnd.Cmp(networkEnd) > 0 {
			return nil, fmt.Errorf("network %s is not large enough to fit requirement %q (%d hosts)", base.String(), it.req.Name, it.req.Hosts)
		}

		newPrefix := bits - it.hostBits
		allocAddr := bigIntToAddr(cur, isIPv6)
		usable := usableHostsForBits(it.hostBits, isIPv6)
		results[it.idx] = Allocation{
			Name:           it.req.Name,
			CIDR:           netip.PrefixFrom(allocAddr, newPrefix).String(),
			Network:        allocAddr,
			Prefix:         newPrefix,
			HostsRequested: it.req.Hosts,
			UsableHosts:    usable,
		}
		cur = blockEnd
	}

	return results, nil
}

func usableHostsForBits(hostBits int, isIPv6 bool) *big.Int {
	total := powerOfTwo(hostBits)
	if isIPv6 {
		return total
	}
	switch hostBits {
	case 0:
		return big.NewInt(1)
	case 1:
		return big.NewInt(2)
	default:
		return new(big.Int).Sub(total, big.NewInt(2))
	}
}
