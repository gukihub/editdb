//
//   editdb.go - display a mysql database table contents
//
//   Guillaume Kielwasser - 17/02/2016
//

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gukihub/editdb/lib"
	"math"
	"net/http"
	"net/http/cgi"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
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
	nullstr string = "nil"
)

//
//   Functions
//

func handler(c *libdb.Context) {
	urlquery := c.R.URL.Query()
	switch urlquery.Get("app") {
	case "lstable":
		handler_lstable(c)
	default:
		handler_index(c)
	}
}

// return a map with the constraints of the given table
// and print them to w
func mapcon(c *libdb.Context, table string) (r map[string]tblcol) {
	r = make(map[string]tblcol)

	query := fmt.Sprintf(`
		select
			COLUMN_NAME,REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		from
			information_schema.key_column_usage
		where
			table_name = '%s'
			and
			REFERENCED_TABLE_NAME is not NULL
			and
			CONSTRAINT_SCHEMA = '%s'
		`, table, c.Dbi.Name)

	rows, err := c.Dbh.Query(query)

	if err != nil {
		panic(err.Error())
	}

	for rows.Next() {
		var column sql.NullString
		var ref_table sql.NullString
		var ref_column sql.NullString

		if err := rows.Scan(&column, &ref_table,
			&ref_column); err != nil {
			panic(err.Error())
		}
		r[column.String] = tblcol{
			table:  ref_table.String,
			column: ref_column.String}
	}

	return (r)
}

// return a slice with the constraints of the given table
// and print them to w
func list_constraints(c *libdb.Context, table string) (conslice []constraint) {
	var cons constraint
	conslice = make([]constraint, 0)

	query := fmt.Sprintf(`
		select
			TABLE_NAME,COLUMN_NAME,REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		from
			information_schema.key_column_usage
		where
			table_name = '%s'
			and
			REFERENCED_TABLE_NAME is not NULL
			and
			CONSTRAINT_SCHEMA = '%s'
		`, table, c.Dbi.Name)

	rows, err := c.Dbh.Query(query)

	if err != nil {
		panic(err.Error())
	}

	for rows.Next() {
		var table sql.NullString
		var column sql.NullString
		var ref_table sql.NullString
		var ref_column sql.NullString

		if err := rows.Scan(&table, &column, &ref_table,
			&ref_column); err != nil {
			panic(err.Error())
		}
		fmt.Fprintf(c.W, "Constraint from %s.%s to %s.%s\n",
			table.String, column.String, ref_table.String,
			ref_column.String)
		cons.table = table.String
		cons.column = column.String
		cons.ref_table = ref_table.String
		cons.ref_column = ref_column.String
		conslice = append(conslice, cons)
	}

	return (conslice)
}

// return a slice with the columns of the given table
func get_table_desc(c *libdb.Context, table string) (table_desc []string) {

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

	for rows.Next() {
		var colName sql.NullString

		if err := rows.Scan(&colName); err != nil {
			panic(err.Error())
		}
		table_desc = append(table_desc,
			fmt.Sprintf("%s.%s", table, colName.String))
	}

	return table_desc
}

func get_col_names(c *libdb.Context, table string) (r []string) {

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
		res = append(res, colName.String)
	}
	return res
}

func mkeditvalue(c *libdb.Context, table string, idx string) (r string) {

	colnames := tablecol(c, table)
	disp := colnames[1]

	query := fmt.Sprintf("select %s,%s from %s", idx, disp, table)
	rows, err := libdb.Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	var res []string
	for _, val := range rows {
		res = append(res, fmt.Sprintf("          '%s':'%s'",
			val[idx], val[disp]))
	}
	return (fmt.Sprintf(strings.Join(res, ",\n")))
}

// return a string with the jqGrid colModel option
func describe_cols(c *libdb.Context, table string) (r string) {

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
				fmt.Sprintf(`    {
      name: '%s',
      editable: true,
      edittype: 'select',
      formatter: 'select',
      editoptions:{
        value: {
%s
        }
      }
    }`, colName.String, mkeditvalue(c, cntr[colName.String].table, cntr[colName.String].column)))
		} else {
			res = append(res,
				fmt.Sprintf("    { name: '%s', editable: true, edittype:'text' }", colName.String))
		}
	}
	return strings.Join(res, ",\n")
}

// return a slice with the table's column names
func tablecol(c *libdb.Context, table string) (r []string) {
	r = make([]string, 0)

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

	for rows.Next() {
		var colName sql.NullString

		if err := rows.Scan(&colName); err != nil {
			panic(err.Error())
		}
		r = append(r, colName.String)
	}
	return (r)
}

// print the table's columns to w
func describe_table(c *libdb.Context, table string) {

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

	for rows.Next() {
		var colName sql.NullString

		if err := rows.Scan(&colName); err != nil {
			panic(err.Error())
		}
		fmt.Fprintf(c.W, "Column name: %s\n", colName.String)
	}
}

// construct the select query that will display all fields
// of the given table. In addition, it will add all fields from the
// foreign tables by analysing its constraints
func mkquery(ctx *libdb.Context, table string,
	conslice []constraint) (query string) {

	// tables list to query from
	var t []string
	// list of conditions
	var c []string
	// query slice
	var q []string
	// and slice
	var a []string
	// field slive
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
func display_table_content(c *libdb.Context,
	table string, conslice []constraint) {

	sql := mkquery(c, table, conslice)

	fmt.Fprintf(c.W, "query: %s\n", sql)

	rows, err := libdb.Query(c.Dbh, sql)
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
				val1[val2] = nullstr
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

// return a slice with the db tables list
func table_list(c *libdb.Context) (tables []string, err error) {

	rows, err := c.Dbh.Query(fmt.Sprintf(`
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA='%s'
		`, c.Dbi.Name))
	if err != nil {
		panic(err.Error())
	}

	for rows.Next() {
		var tableName sql.NullString

		if err := rows.Scan(&tableName); err != nil {
			panic(err.Error())
		}
		tables = append(tables, tableName.String)
	}

	return tables, err
}

func trapexit(c *libdb.Context) {
	fmt.Println()
	libdb.Dbclose(c)
}

func handler_lstable(c *libdb.Context) {
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
	rows, err := libdb.Query(c.Dbh, query)
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

	query = fmt.Sprintf("select * from %s order by %s %s limit %d,%d",
		table, sidx, sord, start, limit)
	rows, err = libdb.Query(c.Dbh, query)
	if err != nil {
		panic(err.Error())
	}

	col_names := get_col_names(c, table)
	for _, row := range rows {
		fmt.Fprintf(c.W, "  <row id=\"%s\">\n", row[col_names[0]])
		for _, val := range col_names {
			fmt.Fprintf(c.W, "    <cell>%s</cell>\n", row[val])
		}
		fmt.Fprintf(c.W, "  </row>\n")
	}
	fmt.Fprintf(c.W, "</rows>\n")
}

func handler_index(c *libdb.Context) {

	const s1 = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en" lang="en">
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<meta http-equiv="X-UA-Compatible" content="IE=edge" />
<title>EditDB</title>
 
<link rel="stylesheet" type="text/css" media="screen" href="css/ui-cupernito/jquery-ui.min.css" />
<link rel="stylesheet" type="text/css" media="screen" href="css/ui.jqgrid.css" />
 
<!--
<style type="text/css">
html, body {
    margin: 0;
    padding: 0;
    font-size: 75%;
}
</style>
-->
 
<script src="js/jquery-1.11.0.min.js" type="text/javascript"></script>
<script src="js/i18n/grid.locale-en.js" type="text/javascript"></script>
<script src="js/jquery.jqGrid.min.js" type="text/javascript"></script>
 
<script type="text/javascript">
$(function () {
`
	const s2 = `
$("#grid{{.Tnum}}").jqGrid({
  url: "?app=lstable&table={{.Tname}}",
  datatype: "xml",
  mtype: "GET",
  colModel: [
{{.Model}}
  ],
  prmNames: { 'oper': 'jqGrid:oper', 'id':'jqGrid:id' },
  cellEdit: true,
  cellsubmit: 'remote',
  cellurl: 'edit.php?table={{.Tname}}',
  editurl: 'edit.php?table={{.Tname}}',
  pager: "#pager{{.Tnum}}",
  height:'auto',
  rowNum: 10,
  rowList: [10, 20, 30],
  //sortname: "",
  sortorder: "asc",
  viewrecords: true,
  gridview: true,
  autoencode: true,
  caption: "Table {{.Tname}}"
}).navGrid(
  '#pager{{.Tnum}}',
  {edit:false,add:true,del:true,search:true},
  { }, { closeAfterAdd: true }
);
`

	const s3 = `
}); 
</script>
 
</head>
<body>
`

	s4 := `
  <table id="grid{{.Tnum}}"><tr><td></td></tr></table> 
  <div id="pager{{.Tnum}}"></div> 
  <br/>
     `

	s5 := `
</body>
</html>
`

	type tdef struct {
		Tnum  int
		Tname string
		Model string
	}
	var td tdef

	tables, err := table_list(c)
	if err != nil {
		panic(err.Error())
	}

	fmt.Fprintf(c.W, "%s\n", s1)

	var t1 = template.Must(template.New("t1").Parse(s2))
	var t2 = template.Must(template.New("t1").Parse(s4))

	i := 1
	// loop on the db table list
	for _, table := range tables {
		//fmt.Fprintf(c.W, "Table name: %s\n", table)
		//describe_table(c, table)
		//cntr := list_constraints(c, table)
		//display_table_content(c, table, cntr)
		td.Tname = table
		td.Tnum = i
		td.Model = describe_cols(c, table)
		//fmt.Fprintf(c.W, "%s\n", s2)
		err := t1.Execute(c.W, td)
		if err != nil {
			fmt.Println("executing template:", err)
		}
		i++
	}

	fmt.Fprintf(c.W, "%s\n", s3)

	fmt.Fprintf(c.W, "  <h2>Database: %s</h2>\n", c.Dbi.Name)

	i = 1
	for _, _ = range tables {
		td.Tnum = i
		//fmt.Fprintf(c.W, "%s\n", s4)
		//describe_table(c, table)
		err := t2.Execute(c.W, td)
		if err != nil {
			fmt.Println("executing template:", err)
		}
		i++
	}

	fmt.Fprintf(c.W, "%s\n", s5)
}

//
//   main
//

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

//
//   end
//
