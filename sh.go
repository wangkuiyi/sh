package sh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/topicai/candy"
)

// Echo splits arg by "\n" and outputs each line.
func Echo(arg string) chan string {
	out := make(chan string)
	go func() {
		for _, seg := range strings.Split(arg, "\n") {
			out <- seg
		}
		close(out)
	}()
	return out
}

// ToFile is like '>' in Shell.  It copies lines from in to a file.
func ToFile(in chan string, filename string) int {
	n := 0
	candy.WithCreated(filename, func(w io.Writer) {
		for l := range in {
			fmt.Fprintf(w, "%s\n", l)
			n++
		}
	})
	return n
}

// Cat reads from file named arg line-by-line.
func Cat(arg string) chan string {
	out := make(chan string)

	go candy.WithOpened(arg, func(r io.Reader) interface{} {
		s := bufio.NewScanner(bufio.NewReader(r))
		for s.Scan() {
			out <- s.Text()
		}
		if e := s.Err(); e != nil {
			panic(e)
		}
		close(out)
		return nil
	})

	return out
}

// Head reads the first line from in and writes it, while consuming
// and ignoring all rest lines.
func Head(in chan string, n int) chan string {
	out := make(chan string)
	go func() {
		for l := range in {
			if n > 0 {
				out <- l
				n--
			}
		}
		close(out)
	}()
	return out
}

// Wc consumes in and counts the number of lines in it.
func Wc(in chan string) int {
	n := 0
	for range in {
		n++
	}
	return n
}

func recursDu(dirname string, out chan string) {
	fis, e := ioutil.ReadDir(dirname)
	if e != nil {
		panic(e)
	}

	for _, fi := range fis {
		fullname := path.Join(dirname, fi.Name())
		if fi.IsDir() {
			recursDu(fullname, out)
		} else {
			out <- fullname
		}
	}
}

// Du takes a directory name and recursively lists all files (not
// sub-directories) in that directory.
func Du(dirname string) chan string {
	out := make(chan string)
	go func() {
		recursDu(dirname, out)
		close(out)
	}()
	return out
}

// Grep consumes all lines from in, and writes those contains pattern.
func Grep(in chan string, pattern string) chan string {
	out := make(chan string)
	go func() {
		r := regexp.MustCompile(pattern)
		for l := range in {
			if r.Find([]byte(l)) != nil {
				out <- l
			}
		}
		close(out)
	}()
	return out
}

// Run executes a command line and collects it stdout line-by-line.
func Run(name string, arg ...string) chan string {
	cmd := exec.Command(name, arg...)

	pr, pw := io.Pipe()
	cmd.Stdout = pw

	var err bytes.Buffer
	cmd.Stderr = &err

	out := make(chan string)

	go func() {
		s := bufio.NewScanner(pr)
		for s.Scan() {
			out <- s.Text()
		}
		close(out) // stops reader of out.
	}()

	go func() {
		if e := cmd.Run(); e != nil {
			// Don't panic here, because panics by a goroutine cannot be covered.
			log.Printf("Failed %s %v: %v, with output\n%s", name, arg, e, err.String())
		}
		pr.Close() // stops s.Scan().
		pw.Close()
	}()

	return out
}

// Cut cuts each line read from in by delim, and returns the field-th
// field if there are enough number of fields. NOTE: field is 1-based,
// as the shell command cut.
func Cut(in chan string, field int, delim string) chan string {
	p := strings.Fields
	if len(delim) > 0 {
		p = func(x string) []string {
			return strings.Split(x, delim)
		}
	}

	out := make(chan string)
	go func() {
		for x := range in {
			fs := p(x)
			if len(fs) >= field {
				out <- fs[field-1]
			}
		}
		close(out)
	}()
	return out
}

// For runs h for each line from in.  It copies outputs from all h
// invocations to its own output channel.
func For(in chan string, h interface{}) chan string {
	prototype := prototype(h)

	out := make(chan string)
	go func() {
		for x := range in {
			switch prototype {
			case 1:
				reflect.ValueOf(h).Call([]reflect.Value{reflect.ValueOf(x), reflect.ValueOf(out)})
			case 2:
				hout := reflect.ValueOf(h).Call([]reflect.Value{reflect.ValueOf(x)})[0]
				for y, ok := hout.Recv(); ok; y, ok = hout.Recv() {
					out <- y.String()
				}
			}
		}
		close(out)
	}()
	return out
}

func prototype(h interface{}) int {
	t := reflect.TypeOf(h)

	if t.Kind() != reflect.Func {
		log.Panicf("Parameter h of For must be a function.")
	}

	if t.NumIn() == 2 && t.In(0).Kind() == reflect.String && t.In(1).Kind() == reflect.Chan && t.In(1).Elem().Kind() == reflect.String && t.NumOut() == 0 {
		return 1 // h(x string, out chan string)
	}
	if t.NumIn() == 1 && t.In(0).Kind() == reflect.String && t.NumOut() == 1 && t.Out(0).Kind() == reflect.Chan && t.Out(0).Elem().Kind() == reflect.String {
		return 2 // h(x string) chan string
	}

	log.Panicf("Unsupported prototype of h.")
	return -1 // dummy
}
