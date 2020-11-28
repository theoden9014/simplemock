package simplemock

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"

	"golang.org/x/tools/go/packages"
	goimports "golang.org/x/tools/imports"

	"github.com/rogpeppe/go-internal/testenv"
)

type GoFile struct {
	*bytes.Buffer

	Package string
	Import  *Import
}

func NewGoFile() *GoFile {
	return &GoFile{
		Buffer: bytes.NewBuffer(nil),
		Import: &Import{},
	}
}

func (f *GoFile) Generate() error {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintln(buf, `package `, f.Package)
	if err := f.Import.WriteTo(buf); err != nil {
		return err
	}
	if _, err := f.WriteTo(buf); err != nil {
		return err
	}
	f.Buffer = buf
	return nil
}

// Format source code by format and goimports
func (f *GoFile) Format() error {
	b, err := format.Source(f.Buffer.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format by fomat: %w", err)
	}
	// remove not used packages in imported
	b, err = goimports.Process("", b, &goimports.Options{})
	if err != nil {
		return fmt.Errorf("failed to format by goimports: %w", err)
	}
	f.Buffer.Reset()
	f.Buffer = bytes.NewBuffer(b)

	return nil
}

// Check source code by govet
func (f *GoFile) Check() error {
	tmpfile, err := ioutil.TempFile("", ".go")
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	buf := bytes.NewBuffer(nil)
	r := io.TeeReader(f, buf)
	if _, err := io.Copy(tmpfile, r); err != nil {
		return err
	}
	f.Buffer = buf

	cmdGoPath, err := testenv.GoTool()
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdGoPath, "vet", tmpfile.Name())
	cmd.Env = os.Environ()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()
	if err := cmd.Start(); err != nil {
		buf := bytes.NewBuffer(nil)
		if _, err := buf.ReadFrom(stderr); err != nil {
			return err
		}
		return fmt.Errorf("%s: %w", buf.String(), err)
	}

	return err
}

type Import struct {
	importsCheck map[string]struct{}
	imports      []string
}

func (im *Import) WriteTo(w io.Writer) error {
	// Sort package
	sort.Sort(im)
	fmt.Fprintln(w, `import (`)
	for i := 0; i < im.Len(); i++ {
		fmt.Fprintln(w, `"`+im.At(i)+`"`)
	}
	fmt.Fprintln(w, `)`)
	return nil
}

func (im *Import) Add(pkg string) {
	if im.imports == nil {
		im.importsCheck = make(map[string]struct{})
	}
	if _, ok := im.importsCheck[pkg]; ok {
		im.importsCheck[pkg] = struct{}{}
		im.imports = append(im.imports, pkg)
	}
}

func (im *Import) AddPackages(imports map[string]*packages.Package) {
	for name := range imports {
		im.Add(name)
	}
}

func (im *Import) At(i int) string {
	return im.imports[i]
}

func (im *Import) Len() int {
	return len(im.imports)
}

func (im *Import) Swap(i, j int) {
	im.imports[i], im.imports[j] = im.imports[j], im.imports[i]
}

func (im *Import) Less(i, j int) bool {
	return im.imports[i] < im.imports[j]
}
