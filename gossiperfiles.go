package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"time"
)

//lock hashreadyLock listfileLock
func (g *Gossiper) FileSharePrepare(name string) error {
	var uname string
	if name[0] == '/' {
		uname = name
	} else {
		uname = "_SharedFiles/" + name
	}

	f, err := os.Open(uname)
	if err != nil {
		return err
	}

	statf, err := f.Stat()
	if err != nil {
		return err
	}

	internname := "_SharedFiles/" + statf.Name()

	fs, err := os.OpenFile(internname, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	fi := &lockedSearchResult{searchresult: SearchResult{FileName: statf.Name()}}
	size := statf.Size()

	fi.searchresult.ChunkCount = uint64(math.Ceil(float64(size) / float64(1<<13)))
	metafilebuf := make([]byte, sha256.Size*fi.searchresult.ChunkCount)
	chunk := make([]byte, 1<<13)
	readytoadd := make(map[string]string)

	for i := uint64(0); i < fi.searchresult.ChunkCount; i++ {
		n, _ := f.Read(chunk)
		b := sha256.Sum256(chunk[:n])
		fname := internname + ".chunk" + strconv.FormatUint(i, 10)
		copy(metafilebuf[i*sha256.Size:], b[:])
		cf, _ := os.Create(fname)
		cf.Write(chunk[:n])
		cf.Close()
		fs.Write(chunk[:n])
		readytoadd[string(b[:])] = fname
		fi.searchresult.ChunkMap = append(fi.searchresult.ChunkMap, i+1)
	}

	h := sha256.Sum256(metafilebuf)
	fi.searchresult.MetafileHash = h[:]

	mf, _ := os.Create(internname + ".meta")
	mf.Write(metafilebuf)
	mf.Close()
	readytoadd[string(fi.searchresult.MetafileHash)] = internname + ".meta"

	g.hashreadyLock.Lock()
	for k, v := range readytoadd {
		g.hashready[k] = v
	}
	g.hashreadyLock.Unlock()

	g.filesLock.Lock()
	g.files = append(g.files, fi)
	g.filesLock.Unlock()

	s := fmt.Sprintf("metahash %s\n", hex.EncodeToString(fi.searchresult.MetafileHash))
	g.Printf(2, s)

	mess := &TLCMessage{g.name, g.seq, -1, BlockPublish{[32]byte{0}, TxPublish{statf.Name(), size, h[:]}}, nil, 0}

	g.addTLCClient(mess)
	g.seq++

	g.sendTLCunconfirmed(mess)

	return nil

}

//lock hashrequestedLock
func (g *Gossiper) FileDownloadMetaStart(filename string, dest string, hash []byte) {

	sendto := dest
	var rfs *returnedfilesstruct
	rfs = nil

	if sendto == "" {
		for _, val := range g.lastcompletedsearch.matchedfiles {
			var sl *SearchResult
			var nodename string
			for nn, x := range val.resultlist {
				sl = x
				nodename = nn
				break
			}
			if bytes.Equal(sl.MetafileHash, hash) {
				rfs = val
				sendto = nodename
				break
			}
		}

	}
	rm := &DataRequest{g.name, sendto, g.hoplimit, hash}

	fip := &lockedSearchResult{searchresult: SearchResult{FileName: filename, ChunkCount: 0}}

	hrt := hashrequesttuple{"_SharedFiles/" + filename + ".meta", time.NewTicker(g.reqtimeout),
		"DOWNLOADING metafile of " + filename + " from " + sendto, 0, fip, rfs}

	go func() {
		for _ = range hrt.Ticker.C {
			g.Printf(3, hrt.printstr+"\n")
			g.sendDirect(&GossipPacket{DataRequest: rm}, sendto)
		}
	}()

	g.Printf(3, hrt.printstr+"\n")
	go g.sendDirect(&GossipPacket{DataRequest: rm}, sendto)

	g.hashrequestedLock.Lock()
	g.hashrequested[string(hash)] = hrt
	g.hashrequestedLock.Unlock()
}

func (g *Gossiper) Filemetaprocess(fi *lockedSearchResult, metacontent []byte, metahash []byte, from string, rfs *returnedfilesstruct) {
	chunkn := len(metacontent) / sha256.Size

	if chunkn*sha256.Size == len(metacontent) {
		//check the meta is not buggy with length (can have a correct hash of meta but meta faulty at generation)

		fi.lock.Lock()
		fi.searchresult.ChunkCount = uint64(chunkn)
		fi.searchresult.MetafileHash = metahash
		fi.lock.Unlock()

		if rfs == nil {
			go g.sendAllchunkRequest(metacontent, chunkn, from, fi)
		} else {
			go g.sendAllchunkRequestFromSearch(rfs, metacontent, fi)
		}
	}
}

func (g *Gossiper) FileChunkUpdateAndReconstruct(fi *lockedSearchResult, chunknum uint64) {
	if fi == nil {
		return
	}

	l := 0
	c := uint64(1)
	name := ""

	fi.lock.Lock()
	i := sort.Search(len(fi.searchresult.ChunkMap), func(i int) bool { return fi.searchresult.ChunkMap[i] >= chunknum })
	if !(i < len(fi.searchresult.ChunkMap)) || fi.searchresult.ChunkMap[i] != chunknum {
		fi.searchresult.ChunkMap = append(fi.searchresult.ChunkMap, chunknum)
		sort.Slice(fi.searchresult.ChunkMap, func(i, j int) bool { return fi.searchresult.ChunkMap[i] < fi.searchresult.ChunkMap[j] })
	}
	l = len(fi.searchresult.ChunkMap)
	c = fi.searchresult.ChunkCount
	name = fi.searchresult.FileName
	fi.lock.Unlock()

	if uint64(l) == c {
		go g.ReconstructFile(name)
	}
}

//lock listfileLock
func (g *Gossiper) ReconstructFile(name string) {
	f, err := os.OpenFile("_Downloads/"+name, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}

	metacontent, err := ioutil.ReadFile("_SharedFiles/" + name + ".meta")
	chunkn := len(metacontent) / sha256.Size

	for i := 0; i < chunkn; i++ {
		chunk, _ := ioutil.ReadFile("_SharedFiles/" + name + ".chunk" + strconv.Itoa(i))

		hverif := sha256.Sum256(chunk)
		if bytes.Equal(metacontent[i*sha256.Size:(i+1)*sha256.Size], hverif[:]) {
			f.Write(chunk)
		} else {
			return
		}
	}

	s := fmt.Sprintf("RECONSTRUCTED file %s\n", name)
	g.Printf(3, s)

}
