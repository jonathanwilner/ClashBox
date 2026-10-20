package main

import "net/netip"

// Older ClashBox settings omit the newer core's LAN ACL. A nil list means
// unspecified; retain the core default. An explicit empty list still denies all.
func defaultLanAllowedIPs(prefixes []netip.Prefix) []netip.Prefix {
	if prefixes == nil {
		return []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")}
	}
	return prefixes
}
