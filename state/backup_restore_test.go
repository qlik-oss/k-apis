package state

import (
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackupRestore(t *testing.T) {

	//Disabling this test by default, since tested methods make k8s API calls
	t.SkipNow()

	usr, err := user.Current()
	assert.NoError(t, err)

	sourceDir, err := os.Getwd()
	assert.NoError(t, err)

	targetDir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	defer os.RemoveAll(targetDir)

	err = Backup(path.Join(usr.HomeDir, ".kube/config"), "test", "", []BackupDir{{ConfigmapKey: "whatever", Directory: sourceDir}})
	assert.NoError(t, err)

	err = Restore(path.Join(usr.HomeDir, ".kube/config"), "test", "", []BackupDir{{ConfigmapKey: "whatever", Directory: targetDir}})
	assert.NoError(t, err)

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
	assert.True(t, reflect.DeepEqual(sourceMap, targetMap))
}
