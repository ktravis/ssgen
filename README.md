# ssgen

A dead-simple static site generator that takes [go text
templates](https://godoc.org/text/template) and a markdown content structure,
producing HTML.

Included is a development server that will watch the sources and refresh output after
they change (i.e. `ssgen -serve :8080`).

Usage:

```bash
~/s/g/k/ssgen $ ssgen -h
Usage of ssgen:
  -debug
    	print debug messages
  -in dir
    	dir for source markdown (default "src")
  -out dir
    	dir for output (default "build")
  -serve address
    	watch files, build and serve output at address
  -static dir
    	dir containing static files to be served at '/static' (default "static")
  -templates dir
    	dir for input templates (default "templates")
```
