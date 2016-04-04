//
//   lsdb.go - display a mysql database table contents
//
//   Guillaume Kielwasser - 17/02/2016
//

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gukihub/editdb/lib"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
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

//
//   Variables definitions
//

var (
	http_port int = 8081
)

const (
	nullstr string = "nil"
)

//
//   Functions
//

func handler(c *libdb.Context) {
	fmt.Fprintf(c.W, "Database: %s\n\n", c.Dbi.Name)

	tables, err := table_list(c)
	if err != nil {
		panic(err.Error())
	}

	// loop on the db table list
	for _, table := range tables {
		fmt.Fprintf(c.W, "Table name: %s\n", table)
		describe_table(c, table)
		cntr := list_constraints(c, table)
		display_table_content(c, table, cntr)
	}
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

//
//   main
//

func main() {
	var c libdb.Context
	libdb.Dbinit(&c)
	libdb.Dbconnect(&c)

	// trap INT and TERM signals and run the trapexit function
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		<-ch
		trapexit(&c)
		os.Exit(0)
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c.W = w
		c.R = r
		handler(&c)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", http_port), nil))
}

//
//   end
//
