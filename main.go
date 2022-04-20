package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {

	runServer(1234)
}

func runServer(port int) {
	// TODO: split your code up mate

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 20 // is this hostname or IP? Dunno, yolo
	transport.IdleConnTimeout = time.Minute
	transport.MaxIdleConns = 20

	// shared client to get http connection keep alive
	client := &http.Client{}
	client.Transport = transport

	// default behaviour if nil is to follow up to 10 redirects
	client.CheckRedirect = nil

	proxyRequest := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		host := r.Host
		if host == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("must provide Host header :("))
			return
		}

		newReq := r.Clone(r.Context())

		// if host == "gst.prod.dl.playstation.net" {
		// 	// TODO: remove me after testing. This is the specificlaly bad host that gives 302s
		// 	host = "69.164.0.128"
		// }
		newReq.URL.Host = host
		newReq.URL.Scheme = "http"
		newReq.RequestURI = ""

		response, err := client.Do(newReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("oh no got an error: " + err.Error()))
			return
		}
		defer response.Body.Close()

		for k, v := range response.Header {
			if len(v) > 0 {
				w.Header().Set(k, v[0])
			}
		}
		w.WriteHeader(response.StatusCode)
		_, err = io.Copy(w, response.Body)
		if err != nil {
			// too late to set status code now
			fmt.Println("got error while responding: ", err)
			return
		}
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      http.HandlerFunc(proxyRequest),
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		IdleTimeout:  time.Minute,
	}

	// not that `proxy_pass` in nginx will use them with how we
	// configure it
	server.SetKeepAlivesEnabled(true)

	err := server.ListenAndServe()
	fmt.Println("Stopped listening because: ", err.Error())
}
