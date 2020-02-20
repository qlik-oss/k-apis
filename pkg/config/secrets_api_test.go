package config

import (
	"os"
	"os/user"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateK8sSecret(t *testing.T) {
	//Disabling this test by default, since tested methods make k8s API calls
	t.SkipNow()

	usr, err := user.Current()
	assert.NoError(t, err)

	kubeConfigFile := path.Join(usr.HomeDir, ".kube/config")
	_, err = os.Stat(kubeConfigFile)
	if err != nil {
		assert.FailNow(t, "unable to find kube config file in the default location", err)
	}

	testNamespace := "t1"

	secretsClient, err := getSecretsClient(kubeConfigFile, testNamespace)
	assert.NoError(t, err)

	_ = secretsClient.Delete("sec1", &metaV1.DeleteOptions{})
	_ = secretsClient.Delete("sec3", &metaV1.DeleteOptions{})

	secretsClient, err = getSecretsClient(kubeConfigFile, defaultNamespace)
	assert.NoError(t, err)
	_ = secretsClient.Delete("sec2", &metaV1.DeleteOptions{})

	type args struct {
		namespace      string
		kubeconfigPath string
		secretName     string
		dataMap        map[string][]byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid create case",
			args: args{
				namespace:      testNamespace,
				kubeconfigPath: kubeConfigFile,
				secretName:     "sec1",
				dataMap: map[string][]byte{
					"data1": []byte("data1"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid update case",
			args: args{
				namespace:      testNamespace,
				kubeconfigPath: kubeConfigFile,
				secretName:     "sec1",
				dataMap: map[string][]byte{
					"data1": []byte("data2"),
				},
			},
			wantErr: false,
		},
		{
			name: "no namespace",
			args: args{
				namespace:      "",
				kubeconfigPath: kubeConfigFile,
				secretName:     "sec2",
				dataMap: map[string][]byte{
					"data1": []byte("data1"),
				},
			},
			wantErr: false,
		},
		{
			name: "no data",
			args: args{
				namespace:      testNamespace,
				kubeconfigPath: kubeConfigFile,
				secretName:     "sec3",
				dataMap:        map[string][]byte{},
			},
			wantErr: false,
		},
		{
			name: "empty secret name",
			args: args{
				namespace:      testNamespace,
				kubeconfigPath: kubeConfigFile,
				secretName:     "",
				dataMap:        map[string][]byte{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateK8sSecret(tt.args.namespace, tt.args.kubeconfigPath, tt.args.secretName, tt.args.dataMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateK8sSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sec, err := GetK8sSecretData(tt.args.namespace, tt.args.kubeconfigPath, tt.args.secretName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetK8sSecretData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sec == nil {
				sec = map[string][]byte{}
			}
			if !reflect.DeepEqual(sec, tt.args.dataMap) {
				t.Errorf("GetK8sSecretData() retrieved data did not match expected data= %v, want %v", sec, tt.args.dataMap)
				return
			}

		})
	}
}
