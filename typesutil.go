package main

import (
	"net"
)

func udpaddrequal(a1,a2 *net.UDPAddr) bool{
	return (a1.Port==a2.Port && a1.IP.Equal(a2.IP))
}

func (s AckRumorSlice) Len() int{
	return len(s)
	}

func (s AckRumorSlice) Less(i, j int) bool{
	return s[i].ID < s[j].ID
	}

func (s AckRumorSlice) Swap(i, j int){
	s[i],s[j]=s[j],s[i]
	}
