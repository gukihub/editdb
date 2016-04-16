//
//   editdb.go - easily edit a mysql database tables
//
//   Guillaume Kielwasser - 17/02/2016
//

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	//"gopkg.in/gcfg.v1"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

//
//   Structure definitions
//

type constraint struct {
	table      string
	column     string
	ref_table  string
	ref_column string
}

type tblcol struct {
	table  string
	column string
}

//
//   Variables definitions
//

const (
	jqgridoper string = "jqGrid:oper"
	jqgridid   string = "jqGrid:id"
	progname   string = "EditDB"
	version    string = "1.0"
)

var (
	// session map
	smap map[string]*Context
	// channel map
	cmap map[string]chan string
)

//
//   Functions
//

func handler(w http.ResponseWriter, r *http.Request) {
	urlquery := r.URL.Query()

	if urlquery.Get("app") == "dologin" {
		handler_dologin(w, r)
		return
	}

	sid := urlquery.Get("sid")
	if sid == "" {
		handler_login(w, r)
		return
	}

	c := smap[sid]
	if c == nil {
		http.Redirect(w, r, "?session=expired", 301)
		return
	}

	c.W = w
	c.R = r
	fmt.Fprintf(os.Stdout, "c: %v\n", c)

	switch urlquery.Get("app") {
	case "lstable":
		handler_lstable(c)
	case "edtable":
		handler_edtable(c)
	case "lsopts":
		handler_lsopts(c)
	case "form":
		handler_form(c)
	//case "login":
	//handler_login(c)
	//case "dologin":
	//handler_dologin(c)
	default:
		handler_index(c)
	}
}

func mkeditopt_url(c *Context, table string, idx string) (r string) {

	colnames := tablecol(c, table)
	disp := colnames[1]

	r = fmt.Sprintf("%8sdataUrl: '?app=lsopts&table=%s&idx=%s&disp=%s'",
		" ", table, idx, disp)

	return r
}

func mkeditopt_value(c *Context, table string, idx string) (r string) {

	colnames := tablecol(c, table)
	disp := colnames[1]

	query := fmt.Sprintf("select %s,%s from %s", idx, disp, table)
	rows, err := Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	var res []string
	for _, val := range rows {
		res = append(res, fmt.Sprintf("          '%s':'%s'",
			val[idx], val[disp]))
	}

	r = fmt.Sprintf(`        value: {
%s
        }`, strings.Join(res, ",\n"))

	return r
}

// return a string with the jqGrid colModel option
func describe_cols(c *Context, table string) (r string) {

	cntr := mapcon(c, table)
	fmt.Fprintf(os.Stderr, "mapcon(%s): %v\n", table, cntr)

	query := fmt.Sprintf(`
		select COLUMN_NAME
		from INFORMATION_SCHEMA.COLUMNS
		where TABLE_NAME='%s'
		and TABLE_SCHEMA='%s'
		`, table, c.Dbi.Name)

	rows, err := c.Dbh.Query(query)
	if err != nil {
		panic(err.Error())
	}

	var res []string
	for rows.Next() {
		var colName sql.NullString

		if err := rows.Scan(&colName); err != nil {
			panic(err.Error())
		}
		if cntr[colName.String].column != "" {
			res = append(res,
				fmt.Sprintf(
					`    {
      name: '%s',
      editable: true,
      edittype: 'select',
      editoptions:{
%s
      }
    }`,
					colName.String,
					mkeditopt_url(
						c,
						cntr[colName.String].table,
						cntr[colName.String].column)))
		} else {
			res = append(res,
				fmt.Sprintf("    { name: '%s', editable: true, edittype:'text' }", colName.String))
		}
	}
	r = strings.Join(res, ",\n")
	//fmt.Println(r)
	return r
}

// construct the select query that will display all fields
// of the given table and resolve the foreign keys constraints
func mkquery2(ctx *Context, table string) (query string) {

	// slice of the table constraints
	conslice := list_constraints(ctx, table)

	// tables list to query from
	var t []string
	// list of conditions
	var c []string
	// query slice
	var q []string
	// and slice
	var a []string
	// field slice
	var f []string

	// the actual table is always queried
	t = append(t, table)

	for _, value := range conslice {
		t = append(t, value.ref_table)
		c = append(c, fmt.Sprintf("%s.%s=%s.%s",
			value.table, value.column,
			value.ref_table, value.ref_column))
	}
	//for _, field := range get_table_desc(ctx, table) {
	cons := mapcon(ctx, table)
	for _, field := range get_col_names(ctx, table) {
		if cons[field].table != "" {
			ref_table_cols := get_col_names(ctx, cons[field].table)
			f = append(f, fmt.Sprintf("%s.%s as '%s.%s'",
				cons[field].table, ref_table_cols[1],
				table, field))

		} else {
			f = append(f, fmt.Sprintf("%s.%s as '%s.%s'",
				table, field, table, field))
		}
	}
	q = append(q, "select")
	q = append(q, strings.Join(f, ","))
	q = append(q, "from")
	q = append(q, strings.Join(t, ","))

	if len(c) > 0 {
		q = append(q, "where")

		for _, value := range c {
			a = append(a, value)
		}
	}

	q = append(q, strings.Join(a, " and "))
	query = strings.Join(q, " ")
	return query
}

// construct the select query that will display all fields
// of the given table. In addition, it will add all fields from the
// foreign tables by analysing its constraints
func mkquery(ctx *Context, table string,
	conslice []constraint) (query string) {

	// tables list to query from
	var t []string
	// list of conditions
	var c []string
	// query slice
	var q []string
	// and slice
	var a []string
	// field slice
	var f []string

	// the actual table is always queried
	t = append(t, table)

	// map of the field we don't want to display, eg foreign key idx
	n := make(map[string]int)

	for _, value := range conslice {
		t = append(t, value.ref_table)
		c = append(c, fmt.Sprintf("%s.%s=%s.%s",
			value.table, value.column,
			value.ref_table, value.ref_column))

		n[fmt.Sprintf("%s.%s", value.table, value.column)] = 1
		n[fmt.Sprintf("%s.%s", value.ref_table, value.ref_column)] = 1
	}
	for _, val := range t {
		for _, field := range get_table_desc(ctx, val) {
			if n[field] != 1 {
				f = append(f, fmt.Sprintf("%s as '%s'",
					field, field))
			}
		}
	}
	q = append(q, "select")
	q = append(q, strings.Join(f, ","))
	q = append(q, "from")
	q = append(q, strings.Join(t, ","))

	if len(c) > 0 {
		q = append(q, "where")

		for _, value := range c {
			a = append(a, value)
		}
	}

	q = append(q, strings.Join(a, " and "))
	query = strings.Join(q, " ")
	return query
}

// display the content of the given table. The output is formatted
// and there is a nice header :)
func display_table_content(c *Context,
	table string, conslice []constraint) {

	sql := mkquery(c, table, conslice)

	fmt.Fprintf(c.W, "query: %s\n", sql)

	rows, err := Query(c.Dbh, sql)
	if err != nil {
		panic(err.Error())
	}
	// fmt.Fprintf(w, "%#+v\n", rows)

	// slice of query fields
	var a []string
	for _, val := range rows {
		for key, _ := range val {
			a = append(a, fmt.Sprintf("%s", key))
		}
		break
	}
	sort.Strings(a)

	// map of maximum fields width
	b := make(map[string]int)
	// init b with the header field len
	for _, val := range a {
		b[val] = len(fmt.Sprintf("%s", val))
	}
	// get the max len of each row
	for _, val1 := range rows {
		for _, val2 := range a {
			if val1[val2] == nil {
				val1[val2] = Nullstr
			}
			if len(fmt.Sprintf("%s", val1[val2])) > b[val2] {
				b[val2] = len(fmt.Sprintf("%s", val1[val2]))
			}
		}
	}

	// headers
	fmt.Fprintf(c.W, "[HEADER] ")
	for _, val := range a {
		format := fmt.Sprintf("%%-%ds", b[val]+2)
		fmt.Fprintf(c.W, format, val)
	}
	fmt.Fprintln(c.W)

	// print rows
	for key, val1 := range rows {
		fmt.Fprintf(c.W, "[%06d] ", key)
		//fmt.Fprintf(w, "[value: %v]\n", val1)
		for _, val2 := range a {
			//fmt.Fprintf(w, "[%s: %s] ", val2, val1[val2])
			format := fmt.Sprintf("%%-%ds", b[val2]+2)
			fmt.Fprintf(c.W, format, val1[val2])
		}
		fmt.Fprintln(c.W)
	}
	fmt.Fprintln(c.W)
}

func trapexit(c *Context) {
	fmt.Println()
	Dbclose(c)
}

func handler_edtable(c *Context) {
	// log sql request to the wbe serveur log
	w := os.Stderr

	urlquery := c.R.URL.Query()
	table := urlquery.Get("table")
	cols := get_col_names(c, table)

	c.R.ParseForm()
	form := c.R.Form
	oper := form.Get(jqgridoper)

	switch oper {

	case "edit":
		id := form.Get(jqgridid)
		updateval := make([]string, 0)
		for _, col := range cols {
			val := form.Get(col)
			if val != "" {
				updateval = append(updateval,
					fmt.Sprintf("%s=\"%s\"", col, val))
			}
		}
		sql := fmt.Sprintf("update %s set %s where %s=%s",
			table, strings.Join(updateval, ","), cols[0], id)
		fmt.Fprintf(w, "%s\n", sql)
		_, err := c.Dbh.Query(sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			http.Error(c.W, fmt.Sprintf("%s\n", err),
				http.StatusInternalServerError)
		}

	case "add":
		colslice := make([]string, 0)
		valslice := make([]string, 0)

		for _, col := range cols {
			val := form.Get(col)
			if val != "" {
				colslice = append(colslice,
					fmt.Sprintf("%s", col))
				valslice = append(valslice,
					fmt.Sprintf("\"%s\"", val))
			}
		}
		sql := fmt.Sprintf("insert into %s (%s) values(%s)",
			table, strings.Join(colslice, ","),
			strings.Join(valslice, ","))
		fmt.Fprintf(w, "%s\n", sql)
		_, err := c.Dbh.Query(sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			http.Error(c.W, fmt.Sprintf("%s\n", err),
				http.StatusInternalServerError)
		}

	case "del":
		id := form.Get(jqgridid)
		sql := fmt.Sprintf("delete from %s where %s=%s",
			table, cols[0], id)
		fmt.Fprintf(w, "%s\n", sql)
		_, err := c.Dbh.Query(sql)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			http.Error(c.W, fmt.Sprintf("%s\n", err),
				http.StatusInternalServerError)
		}

	default:
		handler_edtable_to_file(c)

	}
}

func handler_edtable_to_file(c *Context) {
	f, err := os.Create("editdb.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := f
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
}

func handler_lstable(c *Context) {
	header := c.W.Header()
	header.Set("Content-type", "text/xml;charset=utf-8")

	urlquery := c.R.URL.Query()
	table := urlquery.Get("table")
	page, _ := strconv.Atoi(urlquery.Get("page"))
	limit, _ := strconv.Atoi(urlquery.Get("rows"))
	sidx := urlquery.Get("sidx")
	sord := urlquery.Get("sord")

	// if we not pass at first time index use the first
	// column for the index or what you want
	if sidx == "" {
		sidx = "1"
	}

	// calculate the number of rows for the query.
	// We need this for paging the result
	query := fmt.Sprintf("select count(*) as count from %s", table)
	rows, err := Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}
	tmpmap := rows[0]
	count, _ := strconv.Atoi(fmt.Sprintf("%s", tmpmap["count"]))

	// calculate the total pages for the query
	var total int
	if count > 0 && limit > 0 {
		total = int(math.Ceil(float64(count) / float64(limit)))
	} else {
		total = 0
	}

	// if for some reasons the requested page is greater than the total
	// set the requested page to total page
	if page > total {
		page = total
	}

	// calculate the starting position of the rows
	start := limit*page - limit

	// if for some reasons start position is negative set it to 0
	// typical case is that the user type 0 for the requested page
	if start < 0 {
		start = 0
	}

	fmt.Fprintf(c.W, "<?xml version='1.0' encoding='utf-8'?>\n")

	fmt.Fprintf(c.W, "<rows>\n")
	fmt.Fprintf(c.W, "  <page>%d</page>\n", page)
	fmt.Fprintf(c.W, "  <total>%d</total>\n", total)
	fmt.Fprintf(c.W, "  <record>%d</record>\n", count)

	//query = fmt.Sprintf("select * from %s order by %s %s limit %d,%d",
	//	table, sidx, sord, start, limit)
	query = fmt.Sprintf("%s order by %s %s limit %d,%d",
		mkquery2(c, table), sidx, sord, start, limit)
	fmt.Fprintf(os.Stderr, "%s\n", query)
	rows, err = Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	//col_names := get_col_names(c, table)
	col_names := get_table_desc(c, table)
	for _, row := range rows {
		fmt.Fprintf(c.W, "  <row id=\"%s\">\n", row[col_names[0]])
		for _, val := range col_names {
			fmt.Fprintf(c.W, "    <cell>%s</cell>\n", row[val])
		}
		fmt.Fprintf(c.W, "  </row>\n")
	}
	fmt.Fprintf(c.W, "</rows>\n")
}

func handler_lsopts(c *Context) {
	urlquery := c.R.URL.Query()

	table := urlquery.Get("table")
	idx := urlquery.Get("idx")
	disp := urlquery.Get("disp")

	query := fmt.Sprintf("select %s,%s from %s", idx, disp, table)
	fmt.Fprintf(os.Stderr, "%s\n", query)
	rows, err := Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	fmt.Fprintf(c.W, "<select>\n")
	for _, val := range rows {
		fmt.Fprintf(c.W, "  <option value='%s'>%s</option>\n",
			val[idx], val[disp])
	}
	fmt.Fprintf(c.W, "</select>\n")
}

func handler_form(c *Context) {
	var id int
	urlquery := c.R.URL.Query()

	table := urlquery.Get("table")
	field := urlquery.Get("field")
	key := urlquery.Get("key")
	idstr := urlquery.Get("id")
	if idstr == "" {
		id = -1
	} else {
		id, _ = strconv.Atoi(idstr)
	}

	//fmt.Fprintf(c.W, "field: %s<br>\n", field)

	query := fmt.Sprintf(`
	select
		TABLE_NAME,COLUMN_NAME,REFERENCED_TABLE_NAME,
		REFERENCED_COLUMN_NAME
	from
		information_schema.key_column_usage
	where
		REFERENCED_TABLE_NAME is not NULL
		and
		CONSTRAINT_SCHEMA='%s'
		and
		REFERENCED_TABLE_NAME='%s'
	`, c.Dbi.Name, table)

	//fmt.Fprintf(os.Stderr, "%s\n", query)
	rows, err := Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	for _, row := range rows {
		for col, val := range row {
			fmt.Fprintf(c.W, "%s: %s<br>\n", col, val)
		}
		fmt.Fprintf(c.W, "<br>\n")
	}

	fmt.Fprintf(c.W, "<br>\n")
	//fmt.Fprintf(c.W, "key: %s<br>\n", key)

	// next row:
	// select * from application where id>'3' limit 1;
	// previous row:
	// select * from application where id<'3' limit 1;

	// determine the first id
	query = fmt.Sprintf("select %s from %s limit 1", key, table)
	fmt.Fprintf(os.Stderr, "%s\n", query)
	rows, err = Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}
	row := rows[0]
	firstid, _ := strconv.Atoi(fmt.Sprintf("%s", row[key]))
	//fmt.Fprintf(c.W, "firstid: %d<br/>\n", firstid)

	// determine the last id
	query = fmt.Sprintf("select %s from %s order by %s desc limit 1",
		key, table, key)
	fmt.Fprintf(os.Stderr, "%s\n", query)
	rows, err = Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}
	row = rows[0]
	lastid, _ := strconv.Atoi(fmt.Sprintf("%s", row[key]))
	//fmt.Fprintf(c.W, "lastid: %d<br/>\n", lastid)

	// id out of range, then use the first record
	if (id < firstid) || (id > lastid) {
		id = firstid
	}
	//fmt.Fprintf(c.W, "id: %d<br/>\n", id)

	// form request
	fmt.Fprintf(c.W, "<br>\n")
	query = fmt.Sprintf("select * from %s where %s='%d'", table, key, id)
	rows, err = Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	for _, row := range rows {
		fmt.Fprintf(c.W, "<h2>%s %s: %s</h2><br>\n",
			table, field, row[field])
		for col, val := range row {
			fmt.Fprintf(c.W, "%s: %s<br>\n", col, val)
		}
		fmt.Fprintf(c.W, "<br>\n")
	}

	fmt.Fprintf(c.W, "<br>\n")

	// next link
	if id < lastid {
		query = fmt.Sprintf("select %s from %s where %s>'%d' limit 1",
			key, table, key, id)
		fmt.Fprintf(os.Stderr, "%s\n", query)
		rows, err = Query(c.Dbh, query)
		if err != nil {
			panic(err.Error())
		}
		row = rows[0]
		nextid, _ := strconv.Atoi(fmt.Sprintf("%s", row[key]))
		fmt.Fprintf(c.W, "<a href=\"?app=form&table=%s&field=%s&key=%s&id=%d\">next</a>\n",
			table, field, key, nextid)
	}

	// previous link
	if id > firstid {
		query = fmt.Sprintf("select %s from %s where %s<'%d' order by %s desc limit 1",
			key, table, key, id, key)
		fmt.Fprintf(os.Stderr, "%s\n", query)
		rows, err = Query(c.Dbh, query)
		if err != nil {
			panic(err.Error())
		}
		row = rows[0]
		previd, _ := strconv.Atoi(fmt.Sprintf("%s", row[key]))
		fmt.Fprintf(c.W, "<a href=\"?app=form&table=%s&field=%s&key=%s&id=%d\">previous</a><br/>\n",
			table, field, key, previd)
	}
}

func tolog(s string) {
	log.Println(s)
}

//
//   main
//

func main() {
	tolog("start")
	// config
	docroot := "/home/gui/docker/containers/jqgrid/grid3"
	port := "8083"

	/*
		var c Context
		err := gcfg.ReadFileInto(&c, "editdb.cfg")
		if err != nil {
			log.Panic(err.Error())
		}
	*/

	smap = make(map[string]*Context)
	cmap = make(map[string]chan string)

	//Dbconnect(&c)
	//defer Dbclose(&c)

	// trap SIGINT and SIGTERM then close the DB connexion and quit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		<-ch
		fmt.Fprintf(os.Stderr, "\n")
		for sid, msg := range cmap {
			fmt.Fprintf(os.Stderr, "Closing session %s\n", sid)
			// send the quit signal to the goroutine
			msg <- "quit"
			// wait for the goroutine to complete
			<-msg
		}
		tolog("stop")
		os.Exit(0)
	}()

	http.HandleFunc("/", handler)

	http.Handle("/css/",
		http.FileServer(
			http.Dir(docroot)))
	http.Handle("/js/",
		http.FileServer(
			http.Dir(docroot)))

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

//
//   end
//
