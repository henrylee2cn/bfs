package main

import (
	"bfs/libs/errors"
	"bfs/store/needle"
	"bfs/store/volume"
	log "github.com/golang/glog"
	"net/http"
	"strconv"
	"time"
)

// StartAdmin start admin http listen.
func StartAdmin(addr string, s *Store) {
	go func() {
		var (
			err      error
			serveMux = http.NewServeMux()
		)
		serveMux.Handle("/probe", httpProbeHandler{s: s})
		serveMux.Handle("/bulk_volume", httpBulkVolumeHandler{s: s})
		serveMux.Handle("/compact_volume", httpCompactVolumeHandler{s: s})
		serveMux.Handle("/add_volume", httpAddVolumeHandler{s: s})
		serveMux.Handle("/add_free_volume", httpAddFreeVolumeHandler{s: s})
		if err = http.ListenAndServe(addr, serveMux); err != nil {
			log.Errorf("http.ListenAndServe(\"%s\") error(%v)", addr, err)
			return
		}
	}()
	return
}

// httpProbeHandler http upload a file.
type httpProbeHandler struct {
	s *Store
}

func (h httpProbeHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var (
		v      *volume.Volume
		n      *needle.Needle
		err    error
		vid    int64
		ret    = http.StatusOK
		params = r.URL.Query()
		now    = time.Now()
	)
	if r.Method != "GET" && r.Method != "HEAD" {
		ret = http.StatusMethodNotAllowed
		http.Error(wr, "method not allowed", ret)
		return
	}
	defer HttpGetWriter(r, wr, now, &err, &ret)
	if vid, err = strconv.ParseInt(params.Get("vid"), 10, 32); err != nil {
		log.Errorf("strconv.ParseInt(\"%s\") error(%v)", params.Get("vid"), err)
		ret = http.StatusBadRequest
		return
	}
	n = h.s.Needle()
	if v = h.s.Volumes[int32(vid)]; v != nil {
		if err = v.Probe(n); err != nil {
			if err == errors.ErrNeedleDeleted || err == errors.ErrNeedleNotExist {
				ret = http.StatusNotFound
			} else {
				ret = http.StatusInternalServerError
			}
		}
	} else {
		ret = http.StatusNotFound
		err = errors.ErrVolumeNotExist
	}
	if err == nil {
		if r.Method == "GET" {
			if _, err = wr.Write(n.Data); err != nil {
				log.Errorf("wr.Write() error(%v)", err)
				ret = http.StatusInternalServerError
			}
		}
		if log.V(1) {
			log.Infof("get a needle: %v", n)
		}
	}
	h.s.FreeNeedle(n)
	return
}

// httpBulkVolumeHandler http bulk block.
type httpBulkVolumeHandler struct {
	s *Store
}

func (h httpBulkVolumeHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var (
		err          error
		vid          int64
		bfile, ifile string
		res          = map[string]interface{}{}
	)
	if r.Method != "POST" {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer HttpPostWriter(r, wr, time.Now(), &err, res)
	bfile = r.FormValue("bfile")
	ifile = r.FormValue("ifile")
	if vid, err = strconv.ParseInt(r.FormValue("vid"), 10, 32); err != nil {
		log.Errorf("strconv.ParseInt(\"%s\") error(%v)", r.FormValue("vid"), err)
		err = errors.ErrParam
		return
	}
	go func() {
		log.Infof("bulk volume: %d start", vid)
		err = h.s.BulkVolume(int32(vid), bfile, ifile)
		log.Infof("bulk volume: %d stop", vid)
	}()
	return
}

// httpCompactVolumeHandler http compact block.
type httpCompactVolumeHandler struct {
	s *Store
}

func (h httpCompactVolumeHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var (
		err error
		vid int64
		res = map[string]interface{}{}
	)
	if r.Method != "POST" {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer HttpPostWriter(r, wr, time.Now(), &err, res)
	if vid, err = strconv.ParseInt(r.FormValue("vid"), 10, 32); err != nil {
		log.Errorf("strconv.ParseInt(\"%s\") error(%v)", r.FormValue("vid"), err)
		err = errors.ErrParam
		return
	}
	// long time processing, not block, we can from info stat api get status.
	go func() {
		log.Infof("compact volume: %d start", vid)
		if err = h.s.CompactVolume(int32(vid)); err != nil {
			log.Errorf("s.CompactVolume() error(%v)", err)
		}
		log.Infof("compact volume: %d stop", vid)
	}()
	return
}

// httpAddVolumeHandler http compact block.
type httpAddVolumeHandler struct {
	s *Store
}

func (h httpAddVolumeHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var (
		err error
		vid int64
		res = map[string]interface{}{}
	)
	if r.Method != "POST" {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer HttpPostWriter(r, wr, time.Now(), &err, res)
	if vid, err = strconv.ParseInt(r.FormValue("vid"), 10, 32); err != nil {
		log.Errorf("strconv.ParseInt(\"%s\") error(%v)", r.FormValue("vid"), err)
		err = errors.ErrParam
		return
	}
	log.Infof("add volume: %d", vid)
	_, err = h.s.AddVolume(int32(vid))
	return
}

// httpAddFreeVolumeHandler http compact block.
type httpAddFreeVolumeHandler struct {
	s *Store
}

func (h httpAddFreeVolumeHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	var (
		err        error
		sn         int
		n          int64
		bdir, idir string
		res        = map[string]interface{}{}
	)
	if r.Method != "POST" {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer HttpPostWriter(r, wr, time.Now(), &err, res)
	bdir, idir = r.FormValue("bdir"), r.FormValue("idir")
	if n, err = strconv.ParseInt(r.FormValue("n"), 10, 32); err != nil {
		log.Errorf("strconv.ParseInt(\"%s\") error(%v)", r.FormValue("vid"), err)
		err = errors.ErrParam
		return
	}
	log.Infof("add free volume: %d", n)
	sn, err = h.s.AddFreeVolume(int(n), bdir, idir)
	res["succeed"] = sn
	return
}
