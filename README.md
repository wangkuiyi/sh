# SH

[![Build Status](https://travis-ci.org/wangkuiyi/sh.png?branch=master)](https://travis-ci.org/wangkuiyi/sh) [![GoDoc](https://godoc.org/github.com/wangkuiyi/sh?status.svg)](https://godoc.org/github.com/wangkuiyi/sh)

SH is a package that mimics the Shell programming in Go.  Here are
some examples:

1. Get the most recent version of CoreOS in the stable channel:

    Shell version:

    ```
    curl -s https://stable.release.core-os.net/amd64-usr/current/version.txt | \
	  grep "COREOS_VERSION=" | \
	  cut -f 2 -d '='
    ```

    SH version:

    ```
    <-sh.Cut(
	    sh.Grep(
          sh.Run("curl", "-s", "https://stable.release.core-os.net/amd64-usr/current/version.txt"),
		  "COREOS_VERSION="),
        2, "=")
    ```

1. Find out all Go source files in the current directory and recursively sub-directories that contains the string "func For":

    Shell version:

    ```
    for i in $(du -a | cut -f 2 | grep '\.go$'); do
        if grep "func For" $i > /dev/null; then 
            echo $i
        fi
    done
    ```

    SH version:

    ```
    fmt.Println(Wc(For(Grep(Du("."), "\\.go$"), func(x string, out chan string) {
		if ToFile(Grep(Cat(x), "func For"), os.DevNull) > 0 {
			out <- x
		}
	})))
    ```
