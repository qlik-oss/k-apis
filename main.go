package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"

	"github.com/qlik-oss/k-apis/pkg/state"
)

func main() {
	//Disabling this test by default, since tested methods make k8s API calls
	//t.SkipNow()

	usr, err := user.Current()
	// assert.NoError(t, err)

	sourceDir, err := os.Getwd()
	// assert.NoError(t, err)

	targetDir, err := ioutil.TempDir("", "")
	// assert.NoError(t, err)
	defer os.RemoveAll(targetDir)
	fmt.Println(err)

	err = state.Backup(path.Join(usr.HomeDir, ".kube/config"), "test", "", "", []state.BackupDir{{ConfigmapKey: "whatever", Directory: sourceDir}})
	// assert.NoError(t, err)
	fmt.Println(err)
	err = state.Restore(path.Join(usr.HomeDir, ".kube/config"), "test", "", []state.BackupDir{{ConfigmapKey: "whatever", Directory: targetDir}})
	// assert.NoError(t, err)
	fmt.Println(err)

	sourceMap := make(map[string]bool)
	filepath.Walk(sourceDir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fpath != sourceDir {
			sourceMap[path.Base(fpath)] = true
		}
		return nil
	})

	targetMap := make(map[string]bool)
	filepath.Walk(targetDir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fpath != targetDir {
			targetMap[path.Base(fpath)] = true
		}
		return nil
	})
	fmt.Println(reflect.DeepEqual(sourceMap, targetMap))
	// assert.True(t, reflect.DeepEqual(sourceMap, targetMap))

}
