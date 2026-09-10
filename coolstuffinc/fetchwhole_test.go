package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestFetchWholeResumesATruncatedBody pins that a sell list cut mid-stream is
// finished rather than abandoned. The storefront answers 200 with a short
// body and no transport error about one fetch in three, so the decode was
// the only thing that noticed and three bad attempts in a row failed the
// run outright.
func TestFetchWholeResumesATruncatedBody(t *testing.T) {
	full := strings.Repeat("abcdefghij", 500)

	for _, tt := range []struct {
		desc     string
		cut      int  // bytes served before the body is cut, 0 for none
		noRanges bool // the server ignores Range and starts over
		want     string
		wantErr  bool
		serves   int
	}{
		{desc: "a whole body is taken as it comes", want: full, serves: 1},
		{
			desc: "a cut body is finished by asking for the rest",
			cut:  1234, want: full, serves: 2,
		},
		{
			desc:     "a server that will not resume still answers in full",
			cut:      1234,
			noRanges: true, want: full, serves: 2,
		},
		{
			desc: "a body that stops arriving is an error, not a loop",
			cut:  0, want: "", wantErr: true, serves: 2,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var served int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				served++
				body := full
				status := http.StatusOK
				// Honour a range unless this case says the server won't.
				rng := r.Header.Get("Range")
				if rng != "" && !tt.noRanges {
					var from int
					fmt.Sscanf(rng, "bytes=%d-", &from)
					body = full[from:]
					status = http.StatusPartialContent
					w.Header().Set("Content-Range",
						fmt.Sprintf("bytes %d-%d/%d", from, len(full)-1, len(full)))
				}
				// The first pass is the one that gets cut; a resume is
				// served whole, which is what the real server does.
				send := body
				if served == 1 && (tt.cut > 0 || tt.wantErr) {
					send = body[:tt.cut]
				}
				// Content-Length is what makes the shortfall visible, so
				// state the length of what SHOULD arrive.
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				w.WriteHeader(status)
				_, _ = w.Write([]byte(send))
			}))
			defer srv.Close()

			got, err := fetchWhole(context.Background(), srv.URL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("fetchWhole() = %d bytes, want an error", len(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("fetchWhole(): %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %d bytes, want %d", len(got), len(tt.want))
			}
			if served != tt.serves {
				t.Errorf("server saw %d requests, want %d", served, tt.serves)
			}
		})
	}
}
