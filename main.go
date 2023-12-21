package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/at-wat/ebml-go"
	"github.com/at-wat/ebml-go/webm"
)

type wHeader struct {
	Header  webm.EBMLHeader `ebml:"EBML"`
	Segment struct {
		SeekHead webm.SeekHead `ebml:"SeekHead"`
		Info     webm.Info     `ebml:"Info"`
		Tracks   webm.Tracks   `ebml:"Tracks,stop"`
	} `ebml:"Segment,size=unknown"`
}

func main() {
	http.Handle("/", corsHandler(http.FileServer(http.Dir("public")).ServeHTTP))

	// http.HandleFunc("/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
	// 	w.WriteHeader(200)
	// }))

	http.HandleFunc("/record/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(404)
			return
		}

		// NOTE: start time must be provided from client, this is just for POC
		startTime := time.Now().UTC()
		// src is io.Reader of webm bytestream format coming from client
		src := r.Body

		// extract initial segment, search webm bytestreamformat for explanation
		wh := new(wHeader)
		wh.Segment.Info.DateUTC = startTime
		err := ebml.Unmarshal(src, wh, ebml.WithIgnoreUnknown(true))
		if err != nil && err != ebml.ErrReadStopped {
			log.Println("unmarshal err", err)
			w.WriteHeader(400)
			return
		}

		// write rest of the bytes (media segments) to a temp file (or any io.Writer...)

		// this POC uses tmp file for simplicity
		b, err := os.CreateTemp("", "*-.webm")
		defer func() { os.Remove(b.Name()) }()
		if err != nil {
			log.Println("tmp err", err)
			w.WriteHeader(500)
			return
		}

		// copy all bytes until stream closes, this POC does not handle http2 stream close error (page refresh or restarting stream, etc)
		_, err = io.Copy(b, src)
		if err != nil {
			log.Println("copy err", err)
			w.WriteHeader(500)
			return
		}

		// calculate recording duration as wall clock time since it's started, see NOTE above for startTime value
		wh.Segment.Info.Duration = float64(time.Now().UTC().Sub(startTime).Milliseconds())

		// write initial segment bytes with calculated duration to writer, which is a temp file in this POC
		h, err := os.CreateTemp("", "*-.webm")
		defer func() { os.Remove(h.Name()) }()
		if err != nil {
			log.Println("tmp err", err)
			w.WriteHeader(500)
			return
		}
		err = ebml.Marshal(wh, h)
		if err != nil {
			log.Println("ebml marshall err", err)
			w.WriteHeader(500)
			return
		}

		// all bytes written to tmp files, merge them into playable webm
		recordName := strings.TrimPrefix(r.URL.Path, "/record/")
		d := mustOpenFile("public/recordings", recordName+".webm", os.O_CREATE|os.O_RDWR)
		defer d.Close()
		_, err1 := h.Seek(0, io.SeekStart)
		_, err2 := b.Seek(0, io.SeekStart)
		_, err3 := d.ReadFrom(h)
		_, err4 := d.ReadFrom(b)
		err = errors.Join(err1, err2, err3, err4)
		if err != nil {
			log.Println("merge err", err)
			w.WriteHeader(500)
			return
		}
		log.Println("recording done", d.Name())
	}))

	fmt.Println("Starting the server on :8080...")

	if os.Getenv("APP_ENV") == "development" {
		log.Fatal(http.ListenAndServeTLS(":8080", "certs/cert.pem", "certs/key.pem", nil))
	} else {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
}

func corsHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == http.MethodOptions || r.Method == http.MethodHead {
			return
		}
		h(w, r)
	}
}

func mustOpenFile(dir, name string, flag int) *os.File {
	fname := filepath.Join(string(dir), name)
	if err := os.MkdirAll(filepath.Dir(fname), os.ModePerm); err != nil {
		panic(err)
	}

	file, _ := os.OpenFile(fname, flag, os.ModePerm)
	return file
}
