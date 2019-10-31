package main

import (
	"fmt"
	"net"
	"math"
	"math/rand"
	"os"
	"crypto/sha256"
	"io/ioutil"
	"bytes"
	"strconv"
	"time"
)

func (g *Gossiper) randomPeer(indexNot int)(int , *net.UDPAddr){
	pei :=indexNot
	var addr *net.UDPAddr
	addr=nil
	limit:=0

		g.peersLock.Lock()
		l:=len(g.peers)
		if(l>0){
			for pei==indexNot && limit<5{
				pei = rand.Intn(l)
				limit++
			}
			addr=g.peers[pei]
		}
		g.peersLock.Unlock()

	return pei, addr
}

//lock originstatLock rumorlistLock
func (g *Gossiper) addRumorPeer(rumor *RumorMessage) (bool){

	g.addCheckOrigin(rumor.Origin)

	g.originstatLock.RLock()
	stat,vu:=g.originstat[rumor.Origin]
	if(!vu){
		fmt.Printf("Error saving data\n")
	}
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()
	_,got:=stat.rumorinlist[rumor.ID]
	if(!got){
		if(rumor.Text==""){
			g.routerumlistLock.Lock()
			stat.rumorinlist[rumor.ID]=Rumorlistindextuple{Enum_routerumlist,len(g.routerumlist)}
			g.routerumlist=append(g.routerumlist,rumor)
			g.routerumlistLock.Unlock()
		}else{
			g.rumorlistLock.Lock()
			stat.rumorinlist[rumor.ID]=Rumorlistindextuple{Enum_rumorlist,len(g.rumorlist)}
			g.rumorlist=append(g.rumorlist,rumor)
			g.rumorlistLock.Unlock()
		}
	} else{
		stat.stateLock.Unlock()
		return false
	}

	_,got=stat.rumorinlist[rumor.ID]
	if(!got){
		fmt.Printf("Error saving data\n")
	}

	if(rumor.ID== stat.next){
		nextid:=rumor.ID+1
		_,got=stat.rumorinlist[nextid]
		for got{
			nextid++
			_,got=stat.rumorinlist[nextid]
		}
		stat.next=nextid
	}

	stat.stateLock.Unlock()

	return true

}


//lock rumorlistLock, originrumorLock , originnextLock
func (g *Gossiper) addRumorClient(rumor *RumorMessage){

	g.originstatLock.RLock()
	stat:=g.originstat[rumor.Origin]
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()

		g.rumorlistLock.Lock()
		stat.rumorinlist[rumor.ID]=Rumorlistindextuple{Enum_rumorlist,len(g.rumorlist)}
		g.rumorlist=append(g.rumorlist,rumor)
		g.rumorlistLock.Unlock()


	_,got:=stat.rumorinlist[rumor.ID]
	if(!got){
		fmt.Printf("Error saving data\n")
	}

	stat.next=g.seq+1

	stat.stateLock.Unlock()

}

//lock peersLock, ackwaitLock
func (g *Gossiper) addPeer(addr *net.UDPAddr)(int){

	g.peersLock.Lock()
	ind := -1
	for i,u:=range g.peers{
		if(udpaddrequal(addr,u)){
			ind=i
			break
		}
	}

	if(ind<0){
		ind=len(g.peers)
		g.peers=append(g.peers,addr)
		if(ind==0){
			g.peerstring=g.peerstring+addr.String()
		} else {
			g.peerstring=g.peerstring+","+addr.String()
		}
		g.ackwaitLock.Lock()
		g.ackwaiting[addr.String()]=make(map[string]AckRumorSlice)
		g.ackwaitLock.Unlock()
	}

	g.peersLock.Unlock()

	return ind
}


//lock originrumorLock originnextLock
func (g *Gossiper) addCheckOrigin(origin string){

	g.originstatLock.Lock()
	_,exist:=g.originstat[origin]
	if(!exist){
		var ostat OriginState
		ostat.rumorinlist=make(map[uint32]Rumorlistindextuple)
		ostat.next=1
		g.originstat[origin]=&ostat
	}
	g.originstatLock.Unlock()

}

//lock hashreadyLock listfileLock
func (g *Gossiper) FileSharePrepare(name string)(*Fileintern,error){
	var uname string
	if(name[0]=='/'){
		uname=name
	}else{
		uname="_SharedFiles/"+name
	}

	f,err:=os.Open(uname)
	if(err!=nil){
		return nil,err
	}

	statf,err:=f.Stat()
	if(err!=nil){
		return nil,err
	}

	internname:="_SharedFiles/"+statf.Name()

	fs,err:=os.OpenFile(internname,os.O_RDWR|os.O_CREATE, 0644)
	if(err!=nil){
		return nil,err
	}


	fi:=&Fileintern{name:statf.Name(),size:statf.Size()}

	chunkn:=int64(math.Ceil(float64(fi.size)/float64(1<<13)))
	metafilebuf:=make([]byte, sha256.Size*chunkn)
	chunk:=make([]byte, 1<<13)
	readytoadd:=make(map[string]string)


	for i := int64(0); i < chunkn; i++ {
		n,_:=f.Read(chunk)
		b:=sha256.Sum256(chunk)
		fname:=internname+".chunk"+strconv.FormatInt(i, 10)
		copy(metafilebuf[i*sha256.Size:],b[:])
		cf,_:=os.Create(fname)
		cf.Write(chunk[:n])
		cf.Close()
		fs.Write(chunk[:n])
		readytoadd[string(b[:])]=fname
	}

	fi.metafile=metafilebuf
	fi.metahash=sha256.Sum256(metafilebuf)

	mf,_:=os.Create(internname+".meta")
	mf.Write(metafilebuf)
	mf.Close()
	readytoadd[string(fi.metahash[:])]=internname+".meta"

	g.hashreadyLock.Lock()
	for k,v:=range readytoadd{
		g.hashready[k]=v
	}
	g.hashreadyLock.Unlock()

	g.listfileLock.Lock()
	g.listfile=append(g.listfile, fi)
	g.listfileLock.Unlock()

	return fi,nil

}

//lock hashrequestedLock
func (g *Gossiper) FileDownloadMetaStart(filename string,dest string, hash []byte){
	rm:=&DataRequest{g.name,dest,Def_hoplimit,hash}

	fip:=&fileindown{name:filename,pendingreply:1}

	hrt:=hashrequesttuple{"_SharedFiles/"+filename+".meta",time.NewTicker(g.reqtimeout),fip}

		go func(){
			for _ = range hrt.Ticker.C {
				g.sendDirect(&GossipPacket{DataRequest:rm}, dest)
			}
		}()

	go g.sendDirect(&GossipPacket{DataRequest:rm}, dest)

	g.hashrequestedLock.Lock()
	g.hashrequested[string(hash)]=hrt
	g.hashrequestedLock.Unlock()

}


//lock fileindown.lock
func (g *Gossiper) Filemetaprocess(fi *fileindown, metacontent []byte,dest string){
	chunkn:=len(metacontent)/sha256.Size

	if(chunkn*sha256.Size==len(metacontent)){
		//check the meta is not buggy with length (can have a correct hash of meta but meta faulty at generation)

		fi.lock.Lock()
		fi.pendingreply=fi.pendingreply+chunkn
		fi.lock.Unlock()

		go g.sendAllchunkRequest("_SharedFiles/"+fi.name, metacontent,chunkn, dest, fi)
		}
}

//lock listfileLock
func (g *Gossiper) ReconstructFile(name string){
	f,err:=os.OpenFile("_Downloads/"+name,os.O_RDWR|os.O_CREATE, 0644)
	if(err!=nil){
		return
	}

	metacontent,err:=ioutil.ReadFile(name+".meta")
	chunkn:=len(metacontent)/sha256.Size

	for i := 0; i < chunkn; i++ {
		chunk,_:=ioutil.ReadFile(name+".chunk"+strconv.Itoa(i))

		hverif:=sha256.Sum256(chunk)
		if(bytes.Equal(metacontent[i*sha256.Size:(i+1)*sha256.Size], hverif[:])){
			f.Write(chunk)
		}else{
			return
		}
	}

	statf,_:=f.Stat()

	fi:=&Fileintern{name:name,size:statf.Size(),metafile:metacontent,metahash:sha256.Sum256(metacontent)}


	g.listfileLock.Lock()
	g.listfile=append(g.listfile, fi)
	g.listfileLock.Unlock()

	fmt.Printf("RECONSTRUCTED file %s\n",name)

}
