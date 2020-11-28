package simplemock

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"os"
)

type Command struct {
	Stdout io.Writer
	Stderr io.Writer
}

const (
	StatusOK  int = 0
	StatusErr     = -1
)

func (c *Command) Run(args ...string) int {
	flags := flag.NewFlagSet("simplemockgen", flag.ContinueOnError)
	var (
		outpath string
		pkgname string
	)
	flags.SetOutput(c.Stderr)
	flags.StringVar(&outpath, "out", "", "output file, default output to stdout")
	flags.StringVar(&pkgname, "pkgname", "", "output package name for mock")
	flags.Usage = func() {
		fmt.Fprintf(c.Stderr, "Usage: %s [options...] path1, path2, ...\n", os.Args[0])
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		c.error(err)
		return StatusErr
	}

	// default config
	conf := struct {
		outpath string
		output  io.Writer
		pkgname string
	}{
		outpath: outpath,
		output:  c.Stdout,
		pkgname: pkgname,
	}

	patterns := flags.Args()

	gofile := NewGoFile()

	err := load(patterns, func(pkgname string, node ast.Node, info typeInfo, err error) error {
		if err != nil {
			return err
		}
		if len(conf.pkgname) == 0 {
			conf.pkgname = pkgname
		}
		walk(node, info, func(iface string, ifaceType *types.Interface, err error) error {
			if err != nil {
				return err
			}
			mockname := iface + "Mock"
			mock, err := NewSimpleMock(mockname, ifaceType)
			if err != nil {
				return fmt.Errorf("SimpleMock: %w", err)
			}
			return mock.WriteTo(gofile)
			return nil
		})
		return nil
	})
	if err != nil {
		c.error(err)
		return StatusErr
	}

	gofile.Package = conf.pkgname
	if err := gofile.Generate(); err != nil {
		c.errorf("generate source code: %w", err)
	}
	if err := gofile.Format(); err != nil {
		c.errorf("format source code: %w", err)
	}
	if err := gofile.Check(); err != nil {
		c.errorf("check source code: %w", err)
	}

	if len(conf.outpath) == 0 {
		io.Copy(conf.output, gofile)
		return StatusOK
	}
	f, err := os.OpenFile(conf.outpath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.error(err)
	}
	defer f.Close()
	io.Copy(f, gofile)

	return StatusOK
}

func (c *Command) errorf(format string, a ...interface{}) {
	c.error(fmt.Errorf(format, a...))
}

func (c *Command) error(a ...interface{}) {
	fmt.Fprintln(c.Stderr, a...)
}
