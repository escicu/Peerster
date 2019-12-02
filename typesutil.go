package main

import (
	"net"
	"strings"
)

func udpaddrequal(a1, a2 *net.UDPAddr) bool {
	return (a1.Port == a2.Port && a1.IP.Equal(a2.IP))
}

func (s AckGossipingSlice) Len() int {
	return len(s)
}

func (s AckGossipingSlice) Less(i, j int) bool {
	return s[i].ID < s[j].ID
}

func (s AckGossipingSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func searchrequestequal(a1, a2 *SearchRequest) bool {
	return (a1.Origin == a2.Origin && strings.Join(a1.Keywords, "") == strings.Join(a2.Keywords, ""))
}

func (s AckTLCSlice) Len() int {
	return len(s)
}

func (s AckTLCSlice) Less(i, j int) bool {
	return s[i].ID < s[j].ID
}

func (s AckTLCSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
