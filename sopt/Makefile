all:
	go build -o sopt.cgi sopt.go
	cp sopt.cgi /srv/jqgrid/grid3

.PHONY: install
install:
	cp sopt.cgi /home/gui/docker/containers/jqgrid/grid2

.PHONY: clean
clean:
	rm sopt.cgi
