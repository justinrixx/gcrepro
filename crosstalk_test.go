package gcrepro

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/golang/groupcache/groupcachepb"
	"github.com/golang/protobuf/proto"
	"github.com/mailgun/groupcache/v2"
)

func Test_handleCrosstalk(t *testing.T) {
	t.Run("should NOT propagate to multiple peers", func(t *testing.T) {
		const key = "foo"
		const val = "bar"

		groupcache.NewGroup("test", 1, groupcache.GetterFunc(
			func(_ context.Context, k string, dest groupcache.Sink) error {
				if k == key {
					dest.SetBytes([]byte(val), time.Now().Add(10*time.Minute))
				}
				return nil
			},
		))

		peer2Called := false
		// peer2 just needs to return an error and record if called
		peer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer2Called = true
			w.WriteHeader(http.StatusInternalServerError)
		}))

		pool := groupcache.NewHTTPPoolOpts(
			"http://will-not-match-peer-list.com",
			&groupcache.HTTPPoolOptions{
				Context:   crosstalkIngressContextFromRequest,
				Transport: crosstalkEgressTransport, // comment out this line to break the test
			},
		)

		peer1 := httptest.NewServer(http.HandlerFunc(handleCrosstalk(pool)))
		pool.Set(peer2.URL) // ensure that peer2 is always picked

		// make a peer request
		res, err := http.DefaultClient.Get(peer1.URL + "/_groupcache/test/" + key)
		if err != nil {
			t.Fatal(fmt.Errorf("error talking to crosstalk endpoint: %v", err))
		}
		defer res.Body.Close()

		// process the protocol buffer response
		d, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(fmt.Errorf("error reading crosstalk response body: %v", err))
		}
		pbRes := pb.GetResponse{}
		if err := proto.Unmarshal(d, &pbRes); err != nil {
			t.Fatal(fmt.Errorf("error unmarshaling protocol butffer response body: %v", err))
		}

		// assert expectations
		// peer1 should have looked up the value
		if string(pbRes.Value) != val {
			t.Fatal("missing expected cache value")
		}
		// peer1 should not have called peer2
		if peer2Called {
			t.Fatal("crosstalk unexpectedly propagated")
		}
	})
}
