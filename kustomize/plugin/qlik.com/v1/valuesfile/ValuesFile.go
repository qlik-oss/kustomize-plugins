package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/imdario/mergo"
	"github.com/qlik-trial/kustomize-plugins/kustomize/utils"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	DataSource map[string]interface{} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`
	ValuesFile string                 `json:"valuesFile,omitempty" yaml:"valuesFile,omitempty"`
	Root       string
	ldr        ifc.Loader
	rf         *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("ValuesFile")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	p.Root = ldr.Root()
	return yaml.Unmarshal(c, p)
}

func mergeFiles(orig map[string]interface{}, tmpl map[string]interface{}) (map[string]interface{}, error) {
	var mergedData = orig

	err := mergo.Merge(&mergedData, tmpl)
	if err != nil {
		logger.Printf("error executing Merge(), error: %v\n", err)
		return nil, err
	}

	return mergedData, nil
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var env []string
	var vaultAddressPath, vaultTokenPath string
	var vaultAddress, vaultToken, ejsonKey string
	if p.DataSource["vault"] != nil {
		vaultAddressPath = fmt.Sprintf("%v", p.DataSource["vault"].(map[string]interface{})["addressPath"])
		vaultTokenPath = fmt.Sprintf("%v", p.DataSource["vault"].(map[string]interface{})["tokenPath"])

		if _, err := os.Stat(vaultAddressPath); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(vaultAddressPath)
			if err != nil {
				logger.Printf("error reading file : %v, error: %v\n", vaultAddressPath, err)
				return err
			}
			vaultAddress = fmt.Sprintf("VAULT_ADDR=%s", string(readBytes))
			env = append(env, vaultAddress)
		} else if err != nil {
			logger.Printf("error executing stat on file: %v, error: %v\n", vaultAddressPath, err)
		}

		if _, err := os.Stat(vaultTokenPath); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(fmt.Sprintf("%v", vaultTokenPath))
			if err != nil {
				logger.Printf("error reading file: %v, error: %v\n", vaultTokenPath, err)
				return err
			}
			vaultToken = fmt.Sprintf("VAULT_TOKEN=%s", string(readBytes))
			env = append(env, vaultToken)
		} else if err != nil {
			logger.Printf("error executing stat on file: %v, error: %v\n", vaultTokenPath, err)
		}
	}

	var ejsonPrivateKeyPath string
	if p.DataSource["ejson"] != nil {
		ejsonPrivateKeyPath = fmt.Sprintf("%v", p.DataSource["ejson"].(map[string]interface{})["privateKeyPath"])
		if _, err := os.Stat(ejsonPrivateKeyPath); err == nil {
			readBytes, err := ioutil.ReadFile(ejsonPrivateKeyPath)
			if err != nil {
				logger.Printf("error reading ejson private key file: %v, error: %v\n", ejsonPrivateKeyPath, err)
				return err
			}
			ejsonKey = fmt.Sprintf("EJSON_KEY=%s", string(readBytes))
			env = append(env, ejsonKey)
		} else {
			logger.Printf("error executing stat on ejson private key file: %v, error: %v\n", ejsonPrivateKeyPath, err)
		}
	}
	if os.Getenv("EJSON_KEY") != "" && ejsonKey == "" {
		ejsonKey = fmt.Sprintf("EJSON_KEY=%s", os.Getenv("EJSON_KEY"))
		env = append(env, ejsonKey)
	}

	var dataSource string
	if ejsonKey != "" {
		dataSource = fmt.Sprintf("%v", p.DataSource["ejson"].(map[string]interface{})["filePath"])
	} else if vaultAddress != "" && vaultToken != "" {
		dataSource = fmt.Sprintf("%v", p.DataSource["vault"].(map[string]interface{})["secretPath"])
	} else {
		logger.Print("returning error exit 1\n")
		return errors.New("exit 1")
	}

	filePath := filepath.Join(p.Root, p.ValuesFile)
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Printf("error reading values.tml.yaml file: %v, error: %v\n", filePath, err)
		return errors.New("Error: values.tml.yaml is not found")
	}

	for _, r := range m.Resources() {
		// gomplate the initial values file first
		_, err := r.AsYAML()
		if err != nil {
			logger.Printf("error getting resource as yaml: %v, error: %v\n", r.GetName(), err)
			return errors.New("Error: Not a valid yaml file")
		}
		output, err := utils.RunGomplate(dataSource, p.Root, env, string(fileData), logger)
		if err != nil {
			logger.Printf("error executing runGomplate(), error: %v\n", err)
			return err
		}
		var Values map[string]interface{}
		err = yaml.Unmarshal(output, &Values)
		if err != nil {
			logger.Printf("error unmarshalling yaml, error: %v\n", err)
			return err
		}
		ValuePrefixed := map[string]interface{}{"values": Values}

		mergedFile, err := mergeFiles(r.Map(), ValuePrefixed)
		if err != nil {
			logger.Printf("error executing mergeFiles(), error: %v\n", err)
			return err
		}
		r.SetMap(mergedFile)
	}

	return nil
}
