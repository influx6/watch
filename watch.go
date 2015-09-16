//watch handles reloading of a command by watching a directory and if supplied a set of given extensions for change
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/howeyc/fsnotify"
)

var multispaces = regexp.MustCompile(`\s+`)

func goDeps(targetdir string) (bool, error) {
	cmdline := []string{"go", "get"}

	cmdline = append(cmdline, targetdir)

	//setup the executor and use a shard buffer
	cmd := exec.Command("go", cmdline[1:]...)
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	cmd.Stderr = buf

	err := cmd.Run()

	if buf.Len() > 0 {
		return false, fmt.Errorf("go install failed: %s: %s", buf.String(), err.Error())
	}

	return true, nil
}

//goRun runs the runs a command
func goRun(cmd string) string {
	var cmdline []string
	com := strings.Split(cmd, " ")

	if len(com) < 0 {
		return ""
	}

	if len(com) == 1 {
		cmdline = append(cmdline, com...)
	} else {
		cmdline = append(cmdline, com[0])
		cmdline = append(cmdline, com[1:]...)
	}

	//setup the executor and use a shard buffer
	cmdo := exec.Command(cmdline[0], cmdline[1:]...)
	buf := bytes.NewBuffer([]byte{})
	cmdo.Stdout = buf
	cmdo.Stderr = buf

	_ = cmdo.Run()

	return buf.String()
}

//gobuild runs the build process and returns true/false and an error
func gobuild(dir, name string) (bool, error) {
	cmdline := []string{"go", "build"}

	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s.exe", name)
	}

	target := filepath.Join(dir, name)
	cmdline = append(cmdline, "-o", target)

	//setup the executor and use a shard buffer
	cmd := exec.Command("go", cmdline[1:]...)
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	cmd.Stderr = buf

	err := cmd.Run()

	if buf.Len() > 0 {
		return false, fmt.Errorf("go build failed: %s: %s", buf.String(), err.Error())
	}

	return true, nil
}

// runBin runs the generated bin file with the arguments expected
func runBin(bindir, bin string, args []string) chan bool {
	var relunch = make(chan bool)
	go func() {
		binfile := fmt.Sprintf("%s/%s", bindir, bin)
		// cmdline := append([]string{bin}, args...)
		var proc *os.Process

		for dosig := range relunch {
			if proc != nil {
				if err := proc.Signal(os.Interrupt); err != nil {
					log.Printf("Error in sending signal %s", err)
					proc.Kill()
				}
				proc.Wait()
			}

			if !dosig {
				continue
			}

			cmd := exec.Command(binfile, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				log.Printf("Error starting process: %s", err)
			}

			proc = cmd.Process
		}
	}()
	return relunch
}

func buildPkgWatcher(pkpath string, assets map[string]bool) (*fsnotify.Watcher, error) {
	ws, err := fsnotify.NewWatcher()
	add2Watcher(ws, pkpath, assets)
	return ws, err
}

func buildWatcher(pkpath string) (*fsnotify.Watcher, error) {
	ws, err := fsnotify.NewWatcher()
	ws.Watch(pkpath)
	return ws, err
}

func hasIn(paths []string, dt string) bool {
	for _, so := range paths {
		if strings.Contains(so, dt) || so == dt {
			return true
		}
	}
	return false
}

func watchDir(ws *fsnotify.Watcher, dir string, assets map[string]bool, skip []string) {

	mo, err := os.Stat(dir)

	if err != nil {
		return
	}

	if !mo.IsDir() {
		return
	}

	filepath.Walk(filepath.ToSlash(dir), func(path string, info os.FileInfo, err error) error {

		if strings.Contains(path, ".git") {
			return nil
		}

		if info == nil {
			return nil
		}

		if hasIn(skip, path) {
			return nil
		}

		// if !info.IsDir() {
		// 	return nil
		// }

		if assets[path] {
			return nil
		}

		ws.Watch(path)
		assets[path] = true
		return nil
	})
}

func add2Watcher(ws *fsnotify.Watcher, pkgpath string, assets map[string]bool) {
	pkg, err := build.Import(pkgpath, "", 0)

	if err != nil {
		return
	}

	if pkg.Goroot {
		return
	}

	ws.Watch(pkg.Dir)
	assets[pkgpath] = true

	for _, imp := range pkg.Imports {
		if !assets[imp] {
			add2Watcher(ws, imp, assets)
		}
	}
}

func watch(command, importable, bin, exts string, dobuild, withdir bool, args []string) error {
	log.Printf("Command: %s %s %s %t", command, importable, bin, dobuild)

	extcls := multispaces.ReplaceAllString(exts, " ")
	extens := multispaces.Split(extcls, -1)

	if len(extens) == 1 && extens[0] == "" {
		extens = extens[:0]
	}

	var buildName string
	var ubin string

	var buildHandler = func() error {
		var pkgs *build.Package
		var err error
		pkgs, err = build.Import(importable, "", 0)

		if err != nil {
			return err
		}

		_, buildName = path.Split(pkgs.ImportPath)
		// _, buildName := path.Split("./")

		wd, _ := os.Getwd()
		if bin != "" {
			ubin = filepath.ToSlash(filepath.Join(wd, bin))
		} else {
			ubin = pkgs.BinDir
		}

		// lets install
		_, err = goDeps("./")

		if err != nil {
			return err
		}

		log.Printf("Building Pkg(%s) bin to %s \nUsing name: %s", pkgs.ImportPath, ubin, buildName)

		done, err := gobuild(ubin, buildName)

		if err != nil {
			return err
		}

		_ = done
		return nil
	}

	var buildWatch = func() (*fsnotify.Watcher, error) {
		var err error
		var watch *fsnotify.Watcher

		added := make(map[string]bool)

		if dobuild {
			watch, err = buildPkgWatcher(importable, added)
		} else {
			if !added["./"] {
				watch, err = buildWatcher("./")
			}
		}

		//lets watch the current directory also if allowed
		if withdir && err == nil {
			wod, ex := os.Getwd()

			if ex == nil && wod != "" {
				watchDir(watch, wod, added, []string{ubin})
			}
		}

		return watch, err
	}

	var err error
	var watch *fsnotify.Watcher
	var binRun bool
	var binChan chan bool

	//lets build if we are allowed
	if dobuild {
		if err = buildHandler(); err != nil {
			return err
		}

		binChan = runBin(ubin, buildName, args)
		binRun = true
		binChan <- true
	}

	log.Printf("Building dir watchers.....")
	watch, err = buildWatch()

	if err != nil {
		log.Printf("Unable to build err %s", err.Error())
		return err
	}

	for {

		//should we watch
		we, _ := <-watch.Event

		exo := filepath.Ext(we.Name)

		//if its a .git directory skip it
		if strings.Contains(filepath.ToSlash(we.Name), ".git") {
			continue
		}

		//if its our bin directory skip it
		if filepath.ToSlash(we.Name) == filepath.ToSlash(ubin) {
			continue
		}

		log.Printf("Watch: %s -> %s with extensions: %s", exo, we.Name, extens)

		if len(extens) > 0 {
			var found bool
			for _, mo := range extens {
				if exo == mo {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		log.Printf("Watcher notified change: %s", we.Name)

		watch.Close()

		go func(evs chan *fsnotify.FileEvent) {
			for _ = range evs {
			}
		}(watch.Event)

		log.Printf("Re-initiating watch scans .....")

		if command != "" {
			log.Printf("Running cmd '%s' with result: '%s'", command, goRun(command))
		}

		if dobuild {
			if err = buildHandler(); err != nil {
				return err
			}

			if binRun {
				binChan <- true
			}
		}

		watch, err = buildWatch()

		if err != nil {
			return err
		}

		go func(errors chan error) {
			for _ = range errors {
			}
		}(watch.Error)

	}
}

func usage() {
	fmt.Printf(`Watch:
    About: provides a simple but combined go dir builder and file watcher
    Version: %s
    Usage: watch [--import] <import path> [--cmd] <cmd_to_rerun> [--ext] <extensions> [--bin] <bin path to store> --dir
    `, version)
}

var version = "0.0.1"

func main() {
	exts := flag.String("ext", "", "a space seperated string of extensions to watch")
	cmd := flag.String("cmd", "", "Command to run instead on every change")
	withdir := flag.Bool("dir", false, "This sets the current directories and subdirectories to be watched")
	bindir := flag.String("bin", "./bin", "The build directory for storing the build file")
	importdir := flag.String("import", "", "Command to run instead on every change")

	flag.Parse()

	if *cmd == "" && *importdir == "" {
		usage()
		return
	}

	build := (*importdir != "")

	err := watch(*cmd, *importdir, *bindir, *exts, build, *withdir, flag.Args())

	if err != nil {
		log.Printf("Errored: %s", err.Error())
	}
}
