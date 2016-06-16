default: check

check:
	go test

docs:
	godoc2md github.com/juju/mutex > README.md
	sed -i 's|\[godoc-link-here\]|[![GoDoc](https://godoc.org/github.com/juju/mutex?status.svg)](https://godoc.org/github.com/juju/mutex)|' README.md 


.PHONY: default check docs
