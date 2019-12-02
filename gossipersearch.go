package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func (g *Gossiper) newSearch(sr *SearchRequest) *search {
	fmt.Printf("new search\n")

	ret := &search{}

	ret.returnedfiles = make(map[string]*returnedfilesstruct)
	ret.matchedfiles = make([]*returnedfilesstruct, 0)

	if sr.Budget > 0 {

		go g.sendRepartitSearchRequest(sr, -1)
		time.AfterFunc(time.Duration(1)*time.Second, g.endPendingSearch)

	} else {
		bu := uint64(2)
		sr.Budget = bu

		go g.sendRepartitSearchRequest(sr, -1)

		ret.ticker = time.NewTicker(time.Duration(Def_searchringtime) * time.Second)
		go func() {
			for _ = range ret.ticker.C {
				bu = bu * 2
				sr.Budget = bu
				fmt.Printf("tick %v\n", bu)
				g.sendRepartitSearchRequest(sr, -1)
				if bu >= Def_max_budget {
					time.AfterFunc(time.Duration(1)*time.Second, g.endPendingSearch)
				}
			}
		}()
	}
	return ret

}

func (g *Gossiper) Search(keywords []string) []*SearchResult {
	j := strings.Join(keywords, " | ")
	rep := strings.ReplaceAll(j, "*", ".*")
	re, _ := regexp.Compile(rep)
	res := make([]*SearchResult, 0)
	g.filesLock.Lock()
	for _, v := range g.files {
		v.lock.Lock()
		if re.MatchString(v.searchresult.FileName) {
			res = append(res, &v.searchresult)
		}
		v.lock.Unlock()
	}
	g.filesLock.Unlock()
	return res
}

func (g *Gossiper) searchRequestDuplicate(sr *SearchRequest) bool {
	g.searchduplicatelock.Lock()
	defer g.searchduplicatelock.Unlock()

	for _, r := range g.searchduplicate {
		if searchrequestequal(sr, r) {
			return true
		}
	}

	g.searchduplicate = append(g.searchduplicate, sr)

	time.AfterFunc(time.Duration(500)*time.Millisecond, func() {
		sr.Origin = ""
		for len(g.searchduplicate) > 0 && g.searchduplicate[0].Origin == "" {
			g.searchduplicate = g.searchduplicate[1:]
		}
	})

	return false
}

func (g *Gossiper) endPendingSearch() {
	if g.pendingsearch != nil {
		if g.pendingsearch.ticker != nil {
			g.pendingsearch.ticker.Stop()
		}
		g.lastcompletedsearch = g.pendingsearch
		g.pendingsearch = nil

		g.Printf(3, "SEARCH FINISHED\n")

	}
}
