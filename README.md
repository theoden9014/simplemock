# simplemock
[![Go Report Card](https://goreportcard.com/badge/github.com/theoden9014/simplemock)](https://goreportcard.com/report/github.com/theoden9014/simplemock)

This is a code generator for mock from interfaces.

## Installation
```
go get -u github.com/theoden9014/simplemock/cmd/simplemockgen
```

## Usage
```
Usage: ./simplemock [options...] path1, path2, ...
  -out string
    	output file, default output to stdout
  -pkgname string
    	output package name for mock
```

## Example
```go
// example.go
package example

type Reader interface {
	Read(p []byte) (n int, err error)
}
```

```shell
$ simplemockgen ./example.go
```

Then the following code will be generated.
```go
package example

type ReaderMock struct {
	ReadFunc func(p []byte) (n int, err error)
}

func (m *ReaderMock) Read(p []byte) (n int, err error) {
	if m.ReadFunc != nil {
		return m.ReadFunc(p)
	}
	return 0, nil
}
```
