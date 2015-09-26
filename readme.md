#Watch
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/influx6/watch)
[![Travis](https://travis-ci.org/influx6/watch.svg?branch=master)](https://travis-ci.org/influx6/watch)

A simple filewatcher with basic extension filtering and watching systems (built of [skelterjoin/rerun](https://github.com/skelterjohn/rerun))

##Usage

    watch [--import] <import path> [--ext] <extensions>
    [--build] [--bin] <bin path to store> [--cmd] <cmd_to_rerun> --dir

##Examples

  - Build project directory on any change:

     ```
      watch  --import github.com/influx6/todo
     ```

  - Build project directory and subdirectories files on any change:

     ```
      watch  --import github.com/influx6/todo --dir
     ```

  - Run command on any change within current directory

     ```
      watch  --cmd "echo 'dude'"
     ```

  - Build project directory on any change,run command and only watch files in extensions:

     ```
      watch  --import github.com/influx6/todo --cmd "echo 'dude'" --ext ".go .tmpl .js"
     ```
