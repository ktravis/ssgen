package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
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
					k := strings.ReplaceAll(strings.TrimLeft(dir, "/"), "/", ".")
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
	Metadata map[string]any
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
	d, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	f := &file{
		Metadata: make(map[string]any),
	}
	buf := bytes.NewBuffer(d)
	ctx := parser.NewContext()
	html, err := MarkdownToHtml(ctx, buf.String())
	if err != nil {
		return nil, err
	}
	fm := frontmatter.Get(ctx)
	if fm != nil {
		if err := fm.Decode(&f.Metadata); err != nil {
			return nil, fmt.Errorf("could not decode file metadata: %w", err)
		}
	} else {
		fmt.Printf("No metadata for file: %v\n", path)
	}
	f.Content = html
	f.src = path
	f.Path = strings.TrimRight(fromBase, filepath.Ext(path))
	dir := filepath.Dir(f.Path)
	if s, ok := f.Metadata["name"].(string); ok {
		f.Path = filepath.Join(dir, slugify(s))
	}
	if s, ok := f.Metadata["slug"].(string); ok {
		f.Path = filepath.Join(dir, s)
	}
	if p, ok := f.Metadata["published"]; ok {
		switch t := p.(type) {
		case string:
		case time.Time:
			f.Metadata["published"] = t.Format("2006/01/02")
		default:
			return nil, fmt.Errorf("unexpected type %T for metadata field 'published'", p)
		}
	} else {
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
		switch n {
		case "/":
			n = strings.SplitN(filepath.Base(f.Path), ".", 2)[0]
		case ".":
			n = f.Path
		}
		if v, ok := f.Metadata["template"].(string); ok {
			n = v
		}
		if n == "/" {
			n = "main"
		} else {
			n += ".html"
		}

		o := f.Path
		if s, ok := f.Metadata["slug"].(string); ok {
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
		err = t.Execute(w, map[string]any{
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
