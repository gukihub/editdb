// go build -ldflags "-s -w" -o index.cgi cgi.go

// sopt.go
// not used in the project but kept as reference for cgi program

package main

import (
	"fmt"
	"github.com/gukihub/editdb/lib"
	"net/http"
	"net/http/cgi"
	"strings"
)

func lsconenv(c *libdb.Context) {
	w := c.W
	r := c.R

	header := w.Header()
	header.Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Method:", r.Method)
	fmt.Fprintln(w, "URL:", r.URL.String())
	query := r.URL.Query()
	for k := range query {
		fmt.Fprintln(w, "Query", k+":", query.Get(k))
	}
	r.ParseForm()
	form := r.Form
	for k := range form {
		fmt.Fprintln(w, "Form", k+":", form.Get(k))
	}
	post := r.PostForm
	for k := range post {
		fmt.Fprintln(w, "PostForm", k+":", post.Get(k))
	}
	fmt.Fprintln(w, "RemoteAddr:", r.RemoteAddr)
	if referer := r.Referer(); len(referer) > 0 {
		fmt.Fprintln(w, "Referer:", referer)
	}
	if ua := r.UserAgent(); len(ua) > 0 {
		fmt.Fprintln(w, "UserAgent:", ua)
	}
	for _, cookie := range r.Cookies() {
		fmt.Fprintln(w, "Cookie", cookie.Name+":",
			cookie.Value, cookie.Path, cookie.Domain,
			cookie.RawExpires)
	}
}
func handler(c *libdb.Context) {
	urlquery := c.R.URL.Query()

	table := urlquery.Get("table")
	idx := urlquery.Get("idx")
	disp := urlquery.Get("disp")

	/*
		fmt.Fprintf(c.W, "GET table: %s<br/>\n", table)
		fmt.Fprintf(c.W, "GET idx: %s\n<br/>", idx)
		fmt.Fprintf(c.W, "GET disp: %s\n<br/>", disp)
	*/

	query := fmt.Sprintf("select %s,%s from %s", idx, disp, table)
	rows, err := libdb.Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	var res []string
	fmt.Fprintf(c.W, "{\n")
	for _, val := range rows {
		res = append(res, fmt.Sprintf("  '%s':'%s'",
			val[idx], val[disp]))
	}
	fmt.Fprintf(c.W, strings.Join(res, ",\n"))
	fmt.Fprintf(c.W, "\n}\n")
}

func main() {
	var c libdb.Context
	libdb.Dbinit(&c)
	libdb.Dbconnect(&c)

	err := cgi.Serve(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				c.W = w
				c.R = r
				handler(&c)
			}))
	if err != nil {
		fmt.Println(err)
	}

	libdb.Dbclose(&c)
}
