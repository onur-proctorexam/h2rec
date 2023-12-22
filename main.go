package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/at-wat/ebml-go"
	"github.com/at-wat/ebml-go/webm"
)

type wContainer struct {
	Header  webm.EBMLHeader    `ebml:"EBML"`
	Segment webm.SegmentStream `ebml:"Segment,size=unknown"`
}

func main() {
	addr := flag.String("addr", ":8080", "http server listen addr")
	tlscert := flag.String("tlscert", "certs/cert.pem", "tls cert file")
	tlskey := flag.String("tlskey", "certs/key.pem", "tls key file")
	flag.Parse()

	http.Handle("/", corsHandler(http.FileServer(http.Dir("public")).ServeHTTP))

	http.HandleFunc("/record/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(404)
			return
		}

		b, err := os.CreateTemp("", "*-.webm")
		defer func() { b.Close() }()
		if err != nil {
			log.Println("tmp err", err)
			w.WriteHeader(500)
			return
		}

		wc := new(wContainer)
		err = ebml.Unmarshal(io.TeeReader(r.Body, b), wc)
		if err != nil && !isStreamError(err) {
			log.Println("unmarshal err", err)
			w.WriteHeader(500)
			return
		}

		lc := wc.Segment.Cluster[len(wc.Segment.Cluster)-1]
		fc := wc.Segment.Cluster[0]
		lt := lc.Timecode + uint64(lc.SimpleBlock[len(lc.SimpleBlock)-1].Timecode)
		ft := fc.Timecode + uint64(fc.SimpleBlock[0].Timecode)
		wc.Segment.Info.Duration = float64(lt - ft)

		recordName := strings.TrimPrefix(r.URL.Path, "/record/")
		d := mustOpenFile("public/recordings", recordName+".webm", os.O_CREATE|os.O_RDWR)
		defer d.Close()

		err = ebml.Marshal(wc, d)
		if err != nil {
			log.Println("marshal err", err)
			w.WriteHeader(500)
			return
		}

		err = os.Remove(b.Name())

		log.Println("recording done", d.Name(), err)
	}))

	fmt.Println("Starting the server on " + *addr)

	if *tlscert == "" || *tlskey == "" {
		log.Fatal(http.ListenAndServe(*addr, nil))
	} else {
		log.Fatal(http.ListenAndServeTLS(*addr, *tlscert, *tlskey, nil))
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

// http2 stream error, to catch canceled streams (refreshed browser page...)
var sError = &streamError{}

func isStreamError(err error) bool {
	return errors.As(err, sError)
}

type streamErrCode uint32

type streamError struct {
	StreamID uint32
	Code     streamErrCode
	Cause    error
}

func (e streamError) Error() string {
	return fmt.Sprintf("ID %v, code %v", e.StreamID, e.Code)
}
