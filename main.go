package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mailgun/groupcache/v2"
)

const (
	pathVar = "element"
)

func main() {
	portFlag := flag.Int("port", 8080, "port to run the server on")
	peersFlag := flag.String("peers", "", "peer list to use")

	flag.Parse()

	mux := http.NewServeMux()

	// groupcache
	pool := groupcache.NewHTTPPoolOpts(fmt.Sprintf("http://localhost:%d", *portFlag), &groupcache.HTTPPoolOptions{})
	pool.Set(strings.Split(*peersFlag, ",")...)

	group := groupcache.NewGroup("things", 3000, groupcache.GetterFunc(
		func(_ context.Context, id string, dest groupcache.Sink) error {
			result := []byte(id)

			// Set the result in the groupcache to expire after 5 minutes
			return dest.SetBytes(result, time.Now().Add(5*time.Minute))
		},
	))

	// simple getter endpoint
	mux.Handle(fmt.Sprintf("GET /things/{%s}", pathVar), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var result []byte
		err := group.Get(r.Context(), r.PathValue(pathVar), groupcache.AllocatingByteSliceSink(&result))
		if err != nil {
			fmt.Printf("error getting object: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := struct {
			Result string `json:"result"`
		}{
			Result: string(result),
		}

		body, err := json.Marshal(resp)
		if err != nil {
			fmt.Printf("error marshaling response: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(body)
	}))

	mux.Handle("POST /peers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading peers: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		peers := strings.Split(string(b), ",")

		fmt.Printf("updating peers to %s\n", peers)
		pool.Set(peers...)
	}))

	mux.Handle("/_groupcache/", pool)

	fmt.Printf("listening on port %d\n", *portFlag)
	http.ListenAndServe(fmt.Sprintf("localhost:%d", *portFlag), mux)
}
