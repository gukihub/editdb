package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
	"time"
)

func pause(s string) {
	log.Println(s)
	time.Sleep(2 * time.Second)
}

func mkSessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func handler_login(w http.ResponseWriter, r *http.Request) {
	urlquery := r.URL.Query()

	type st_tmpl struct {
		Session string
	}

	tmpl := st_tmpl{Session: urlquery.Get("session")}

	tmplfile := "login.tmpl"
	t := template.New("t")
	t1 := template.Must(t.ParseFiles(tmplfile))
	err := t1.ExecuteTemplate(w, tmplfile, tmpl)
	if err != nil {
		fmt.Println("executing template:", err)
	}
}

func handler_dologin(w http.ResponseWriter, r *http.Request) {
	/*
		w := c.W
		r := c.R

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
			fmt.Fprintln(w, "Cookie",
				cookie.Name+":", cookie.Value, cookie.Path,
				cookie.Domain, cookie.RawExpires)
		}
	*/

	var c Context

	//c.W = w
	//c.R = r

	sid := mkSessionId()
	c.Sid = sid

	r.ParseForm()
	form := r.Form
	c.Dbi.Host = form.Get("dbhost")
	c.Dbi.Name = form.Get("dbname")
	c.Dbi.User = form.Get("dbuser")
	c.Dbi.Pass = form.Get("dbpass")
	c.Dbi.Port = "3306"

	smap[sid] = &c

	cmap[sid] = make(chan string)

	go dbConRoutine(&c, cmap[sid])

	switch <-cmap[sid] {
	case "error":
		delete(smap, sid)
		delete(cmap, sid)
		http.Redirect(w, r, "?session=confailed", 301)
	case "ok":
		http.Redirect(w, r, "?sid="+sid, 301)
	default:
	}

	//handler_index(c)
}

func dbConRoutine(c *Context, msg chan string) {
	err := Dbconnect(c)
	if err != nil {
		msg <- "error"
		return
	}
	msg <- "ok"
	for {
		switch <-msg {
		case "quit":
			Dbclose(c)
			msg <- "ok"
			return
		default:
		}
	}
}
