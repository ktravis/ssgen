package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/shurcooL/github_flavored_markdown"
)

type content struct {
	files     []*file
	tree      map[string][]*file
	templates map[string]*template.Template
}

func loadContent() (*content, error) {
	c := &content{
		tree:      make(map[string][]*file),
		templates: make(map[string]*template.Template),
	}
	err := filepath.Walk(*in, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			for _, e := range []string{".md", ".markdown"} {
				if e == ext {
					dbg("parsing '%s'...", path)
					f, err := parseMarkdownFile(path)
					if err != nil {
						return err
					}
					c.files = append(c.files, f)
					dir := filepath.Dir(f.Path)
					k := strings.Replace(strings.TrimLeft(dir, "/"), "/", ".", -1)
					if k != "" {
						c.tree[k] = append(c.tree[k], f)
					}
					return nil
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	c.templates, err = loadTemplates()
	if err != nil {
		return nil, err
	}
	return c, nil
}

type file struct {
	src string

	Path     string
	Metadata map[string]string
	Content  string
}

func parseMarkdownFile(path string) (*file, error) {
	fromBase := strings.TrimLeft(path, *in)
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	stat, err := src.Stat()
	if err != nil {
		return nil, err
	}
	d, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	f := &file{
		Metadata: make(map[string]string),
	}
	buf := bytes.NewBuffer(d)
	for {
		if b := buf.Bytes(); len(b) == 0 || b[0] != '@' {
			break
		}
		line, err := buf.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		parts := strings.SplitN(line[1:], "=", 2)
		k := strings.TrimSpace(parts[0])
		v := ""
		if len(parts) == 2 {
			v = strings.TrimSpace(parts[1])
		}
		f.Metadata[k] = v
	}
	f.Content = string(github_flavored_markdown.Markdown(buf.Bytes()))
	f.src = path
	f.Path = strings.TrimRight(fromBase, filepath.Ext(path))
	dir := filepath.Dir(f.Path)
	if s, ok := f.Metadata["name"]; ok {
		f.Path = filepath.Join(dir, slugify(s))
	}
	if s, ok := f.Metadata["slug"]; ok {
		f.Path = filepath.Join(dir, s)
	}
	if _, ok := f.Metadata["published"]; !ok {
		f.Metadata["published"] = stat.ModTime().Format("2006/01/02")
	}
	return f, nil
}

func (c *content) compile() error {
	for _, f := range c.files {
		if _, ok := f.Metadata["skip"]; ok {
			dbg("skipping output of file '%s'...", f.src)
			continue
		}
		d := filepath.Dir(f.Path)

		n := filepath.Base(d)
		if n == "/" {
			n = strings.SplitN(filepath.Base(f.Path), ".", 2)[0]
		} else if n == "." {
			n = f.Path
		}
		if v, ok := f.Metadata["template"]; ok {
			n = v
		}
		if n == "/" {
			n = "main"
		} else {
			n += ".html"
		}

		o := f.Path
		if s, ok := f.Metadata["slug"]; ok {
			o = filepath.Join(d, s)
		}
		index := filepath.Join(*out, o)
		if strings.HasSuffix(f.Path, "/index") {
			index += ".html"
		} else {
			index = filepath.Join(index, "index.html")
		}
		if err := os.MkdirAll(filepath.Join(*out, o), 0755); err != nil {
			return err
		}
		w, err := os.Create(index)
		if err != nil {
			return err
		}

		dbg("compiling '%s' against template '%s'...", index, n)
		t, ok := c.templates[n]
		if !ok {
			return fmt.Errorf("cannot find template '%s'", n)
		}
		err = t.Execute(w, map[string]interface{}{
			"file": f,
			"root": c.tree,
		})
		if err != nil {
			w.Close()
			return err
		}
		w.Close()
	}
	return nil
}
