package tcgplayer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoadSYPNamesAnExpiredTicket pins that a ticket the store no longer
// honours is reported as such. The store answers with a redirect to its logon
// page, the client follows it, and the CSV reader finds nothing in the page -
// which used to read as an empty list rather than a list never served.
func TestLoadSYPNamesAnExpiredTicket(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/direct/ExportSYPList", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/account/logon?ReturnUrl=%2fadmin%2fdirect%2fExportSYPList", http.StatusFound)
	})
	mux.HandleFunc("/admin/account/logon", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body>Log On</body></html>")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := loadSYPFrom(context.Background(), srv.URL+"/admin/direct/ExportSYPList", 1, "stale")
	if !errors.Is(err, ErrSYPNotAuthenticated) {
		t.Fatalf("got %v, want ErrSYPNotAuthenticated", err)
	}
}

// TestLoadSYPReadsTheList pins the other side: a served list is read.
func TestLoadSYPReadsTheList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		fmt.Fprint(w, "skuId,a,b,c,d,e,f,marketPrice,qty\n123,x,x,x,x,x,x,1.25,3\n")
	}))
	defer srv.Close()

	got, err := loadSYPFrom(context.Background(), srv.URL, 1, "ticket")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SkuID != 123 {
		t.Fatalf("got %+v, want the one sku", got)
	}
}
