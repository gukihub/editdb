//
//   libdb.go - lib for editdb
//
//   Guillaume Kielwasser - 02/04/2016
//

package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"os"
)

//
//   Structure definitions
//

type Dbinfo struct {
	Host string
	Name string
	Port string
	User string
	Pass string
}

type Wwwinfo struct {
	Port    string
	Docroot string
}

type Context struct {
	Dbi  Dbinfo
	Wwwi Wwwinfo
	Dbh  *sql.DB
	W    http.ResponseWriter
	R    *http.Request
	Sid  string
}

//
//   Variables definitions
//

const (
	Nullstr string = ""
)

//
//   Functions
//

/*
// Not used anymore, using a config file now
func Dbinit(ctx *Context) {
	ctx.Dbi.Host = "localhost"
	ctx.Dbi.Name = "db2"
	ctx.Dbi.Port = "3306"
	ctx.Dbi.User = "root"
	ctx.Dbi.Pass = "root"
}
*/

func Dbconnect(ctx *Context) {
	// connexion string
	constr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		ctx.Dbi.User, ctx.Dbi.Pass,
		ctx.Dbi.Host, ctx.Dbi.Port, ctx.Dbi.Name)
	fmt.Fprintf(os.Stderr, "Connection string: %s\n", constr)
	db, err := sql.Open("mysql", constr)
	if err != nil {
		panic(err.Error())
	}

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	fmt.Fprintf(os.Stderr, "Connection ok\n")

	ctx.Dbh = db
	return
}

// this function execute the given query to the DB and return the results
func Query(db *sql.DB, query string,
	args ...interface{}) ([]map[string]interface{}, error) {

	if db == nil {
		return nil, errors.New("db is nil")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	res := make([]map[string]interface{}, 0)

	for rows.Next() {
		container := make([]interface{}, len(cols))
		dest := make([]interface{}, len(cols))
		for i, _ := range container {
			dest[i] = &container[i]
		}
		rows.Scan(dest...)
		r := make(map[string]interface{})
		for i, colname := range cols {
			val := dest[i].(*interface{})
			if *val != nil {
				r[colname] = *val
			} else {
				r[colname] = Nullstr
			}
		}
		res = append(res, r)
	}

	return res, nil
}

func Dbclose(ctx *Context) {
	ctx.Dbh.Close()
	fmt.Fprintf(os.Stderr, "Connection closed\n")
}

//
//   end
//
