#Watch
A simple filewatcher with basic extension filtering and watching systems (built of [skelterjoin/rerun](https://github.com/skelterjohn/rerun))

##Usage

    watch [--import] <import path> [--ext] <extensions>
    [--build] [--bin] <bin path to store> [--cmd] <cmd_to_rerun>

##examples

  - Build project directory on any change:

     ```watch  --import github.com/influx6/todo```

  - Run command on any change within current directory

     ```watch  --cmd "echo 'dude'"```

  - Build project directory on any change,run command and only watch files in extensions:

     ```watch  --import github.com/influx6/todo --cmd "echo 'dude'" --ext ".go .tmpl .js"```
