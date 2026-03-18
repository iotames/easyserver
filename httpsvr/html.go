package httpsvr

import (
	"fmt"
	"io"

	"os"
	"path/filepath"

	"text/template"
)

var (
	tplLeftDelim  = "{{"
	tplRightDelim = "}}"
)

// SetTplDelims 设置模板的左右边界符。不设置默认为 {{ 和 }}。
//
// Example:
//
//	httpsvr.SetTplDelims("<%{", "}%>")
func SetTplDelims(left, right string) {
	tplLeftDelim = left
	tplRightDelim = right
}

func newTpl(name string) *template.Template {
	tplFuncs := template.FuncMap{}
	// return template.New(name).Funcs(tplFuncs).Delims("<%{", "}%>")
	return template.New(name).Funcs(tplFuncs).Delims(tplLeftDelim, tplRightDelim)
}

// SetContentByTplFile 从模板文件中设置内容
//
// Example1:
//
//	f, err := os.OpenFile(targetFilepath, os.O_CREATE|os.O_WRONLY, 0o755)
//	SetContentByTplFile(tplFilepath, f, data)
//	f.Close()
//
// Example2:
//
//	var bf bytes.Buffer
//	var data = map[string]any{"name": "Tom"}
//	SetContentByTplFile(tplFilepath, &bf, data)
//	SetContentByTplFile(tplFilepath, os.Stdout, data)
func SetContentByTplFile(tplFilepath string, wr io.Writer, data any) error {
	// t, err := template.ParseFiles(tplFilepath)
	t, err := parseFiles(tplFilepath)
	if err != nil {
		return err
	}
	return t.Execute(wr, data)
}

func parseFiles(filenames ...string) (*template.Template, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}

	var t *template.Template
	for _, filename := range filenames {
		name, b, err := readFileOS(filename)
		if err != nil {
			return nil, err
		}
		s := string(b)
		var tmpl *template.Template
		if t == nil {
			t = newTpl(name)
		}
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name) // .Funcs(tplFuncs)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func readFileOS(file string) (name string, b []byte, err error) {
	name = filepath.Base(file)
	b, err = os.ReadFile(file)
	return
}
