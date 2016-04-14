package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"text/template"
)

func handler_index(c *Context) {

	type st_tabledef struct {
		Tnum  int
		Tname string
		Model string
	}

	type st_tmpdef struct {
		Tabledef   []st_tabledef
		Jqgridoper string
		Jqgridid   string
		Progname   string
		Version    string
		Dbname     string
	}

	var td st_tmpdef

	td.Dbname = c.Dbi.Name
	td.Progname = progname
	td.Jqgridoper = jqgridoper
	td.Jqgridid = jqgridid
	td.Version = version

	tables, err := table_list(c)
	if err != nil {
		panic(err.Error())
	}

	tmplfile := "index.tmpl"
	t := template.New("t")
	t1 := template.Must(t.ParseFiles(tmplfile))

	i := 1
	// loop on the db table list
	for _, table := range tables {
		td.Tabledef = append(td.Tabledef,
			st_tabledef{
				Tname: table,
				Tnum:  i,
				Model: describe_cols(c, table)})
		i++
	}

	err = t1.ExecuteTemplate(c.W, tmplfile, td)
	if err != nil {
		fmt.Println("executing template:", err)
	}
}
