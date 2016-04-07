package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

// return a map with the constraints of the given table
func mapcon(c *Context, table string) (r map[string]tblcol) {
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
func list_constraints(c *Context, table string) (conslice []constraint) {
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
		/*
		   fmt.Fprintf(c.W, "Constraint from %s.%s to %s.%s\n",
		           table.String, column.String, ref_table.String,
		           ref_column.String)
		*/
		cons.table = table.String
		cons.column = column.String
		cons.ref_table = ref_table.String
		cons.ref_column = ref_column.String
		conslice = append(conslice, cons)
	}

	return (conslice)
}

// return a slice with table.column of the given table
func get_table_desc(c *Context, table string) (table_desc []string) {

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

// return a slice with the columns of the given table
func get_col_names(c *Context, table string) (r []string) {

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

// print the table's columns to w
func describe_table(c *Context, table string) {

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

// return a slice with the db tables list
func table_list(c *Context) (tables []string, err error) {

	// tables type to return. We display views first.
	table_type := []string{"VIEW", "BASE TABLE"}

	for _, t := range table_type {
		query := fmt.Sprintf(`
                        SELECT TABLE_NAME
                        FROM INFORMATION_SCHEMA.TABLES
                        WHERE TABLE_TYPE='%s' AND TABLE_SCHEMA='%s'
                        `, t, c.Dbi.Name)
		rows, err := c.Dbh.Query(query)
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
	}

	return tables, err
}

// return a slice with the table's column names
func tablecol(c *Context, table string) (r []string) {
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
