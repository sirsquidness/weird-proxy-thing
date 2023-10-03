package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
        "net"
        "context"
)

func main() {

	runServer(1234)
}


func runServer(port int) {
	// TODO: split your code up mate

	transport := http.DefaultTransport.(*http.Transport).Clone()

        //TODO: split up code even more mate
        var (
          dnsResolverIP        = "1.1.1.1:53" // Google DNS resolver.
          dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
          dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
        )

        dialer := &net.Dialer{
          Resolver: &net.Resolver{
            PreferGo: true,
            Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
              d := net.Dialer{
                Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
              }
              return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
            },
          },
        }

        dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
          return dialer.DialContext(ctx, network, addr)
        }

        transport.DialContext = dialContext

	transport.MaxConnsPerHost = 20 // is this hostname or IP? Dunno, yolo
	transport.IdleConnTimeout = time.Minute
	transport.MaxIdleConns = 20

	// shared client to get http connection keep alive
	client := &http.Client{}
	client.Transport = transport

	// if this CheckRedirect is nil, it will follow up to 10 redirects...
	// but we'd like to log to know that we saved the day, so we override the func
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			// dunno if via can ever be len()==0, but safety first right?
			return nil
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		fmt.Printf("Saving the day for a 302 from: %s\n", via[0].URL.String())
		return nil
	}

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
		fmt.Println("Handled ", r.URL.String())
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
