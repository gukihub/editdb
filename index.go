package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"text/template"
)

func handler_index(c *Context) {

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
	s2 := fmt.Sprintf(`
$("#grid{{.Tnum}}").jqGrid({
  url: "?app=lstable&table={{.Tname}}",
  datatype: "xml",
  mtype: "GET",
  colModel: [
{{.Model}}
  ],
  prmNames: { 'oper': '%s', 'id':'%s' },
  cellEdit: true,
  cellsubmit: 'remote',
  // url used when editing a record
  cellurl: '?app=edtable&table={{.Tname}}',
  // url used when adding a record
  editurl: '?app=edtable&table={{.Tname}}',
  pager: "#pager{{.Tnum}}",
  height: 'auto',
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
`, jqgridoper, jqgridid)

	const s3 = `
}); 
</script>
 
<style>
html
{
    height:100%;
}

body
{
    margin:0px;
    font-family:verdana;
    font-size:12px;
    position:absolute; top:0; bottom:0; right:0; left:0;
}

#cadre
{
/*
    margin-top:20px;
    margin-left:40px;
    margin-right:40px;
    margin-bottom:20px;
*/
    margin-top:0px;
    margin-left:0px;
    margin-right:0px;
    margin-bottom:0px;
    border:solid 1px black;
    position:absolute;
    top:0px;
    left:0px;
    right:0px;
    bottom:0px;
}

#header {
    background-color:black;
    color:white;
    text-align:center;
    padding:5px;
    height:60px;
}

#nav {
    line-height:20px;
    background-color:#eeeeee;
    float:left;
    padding:5px;
    overflow:auto;
    position:absolute;
    width:120px;
    top:70px;
    left:0px;
    bottom:0px;
}

#main {
    padding:10px;
    overflow:auto;
    position:absolute;
    top:70px;
    left:130px;
    right:0px;
    bottom:24px;
}
#footer {
    background-color:black;
    color:white;
    clear:both;
    text-align:center;
    padding:5px;
    position:absolute;
    left:0px;
    right:0px;
    bottom:0px;
    height:14px
}
</style>

</head>
<body>
  <div id="cadre">
    <div id="header">
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
	var t2 = template.Must(template.New("t2").Parse(s4))

	type tblidx struct {
		index int
		table string
	}

	navslice := make([]tblidx, 0)
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
		navslice = append(navslice, tblidx{index: i, table: table})
		i++
	}

	fmt.Fprintf(c.W, "%s\n", s3)

	// in div header
	fmt.Fprintf(c.W, "  <h2>Database: %s</h2>\n", c.Dbi.Name)

	fmt.Fprintf(c.W, ` </div>

        <div id="nav">
                <!--<br/><br/>-->`)

	for _, t := range navslice {
		// we point the link to gbox_grid# instead of grid#
		// because it would point to low in the table
		fmt.Fprintf(c.W, "<a href=\"#gbox_grid%d\">%s</a></br>\n",
			t.index, t.table)
	}

	fmt.Fprintf(c.W, `
                <!--<br/><br/>-->
        </div>

        <div id="main">
                <!--<br/>-->

`)

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

	fmt.Fprintf(c.W, ` </div>
        <div id="footer">
                EditDB 1.0
        </div>
  </div>
`)
	fmt.Fprintf(c.W, "%s\n", s5)
}
