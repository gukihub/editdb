all:
	go build -o index.cgi
	cp index.cgi /srv/jqgrid/grid3

clean:
	rm index.cgi
