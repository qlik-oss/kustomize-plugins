package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/qlik-oss/kustomize-plugins/kustomize/utils"

	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	DataSource map[string]interface{} `json:"dataSource,omitempty" yaml:"dataSource,omitempty"`
	Pwd        string
	ldr        ifc.Loader
	rf         *resmap.Factory
}

//nolint: go-lint noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

var logger *log.Logger

func init() {
	logger = utils.GetLogger("Gomplate")
}

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) (err error) {
	p.ldr = ldr
	p.rf = rf
	p.Pwd = ldr.Root()
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	var env []string
	var vaultAddressPath, vaultTokenPath string
	var vaultAddress, vaultToken, ejsonKey string
	if p.DataSource["vault"] != nil {
		vaultAddressPath = fmt.Sprintf("%s", p.DataSource["vault"].(map[string]interface{})["addressPath"])
		vaultTokenPath = fmt.Sprintf("%s", p.DataSource["vault"].(map[string]interface{})["tokenPath"])

		if _, err := os.Stat(vaultAddressPath); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(vaultAddressPath)
			if err != nil {
				logger.Printf("error reading vault address file: %v, error: %v\n", vaultAddressPath, err)
				return err
			}
			vaultAddress = fmt.Sprintf("VAULT_ADDR=%s", string(readBytes))
			env = append(env, vaultAddress)
		} else if err != nil {
			logger.Printf("error executing stat on vault address file: %v, error: %v\n", vaultAddressPath, err)
		}

		if _, err := os.Stat(vaultTokenPath); os.IsNotExist(err) {
			readBytes, err := ioutil.ReadFile(vaultTokenPath)
			if err != nil {
				logger.Printf("error reading vault token file: %v, error: %v\n", vaultTokenPath, err)
				return err
			}
			vaultToken = fmt.Sprintf("VAULT_TOKEN=%s", string(readBytes))
			env = append(env, vaultToken)
		} else if err != nil {
			logger.Printf("error executing stat on vault token file: %v, error: %v\n", vaultTokenPath, err)
		}
	}

	var ejsonPrivateKeyPath string
	if p.DataSource["ejson"] != nil {
		ejsonPrivateKeyPath = fmt.Sprintf("%s", p.DataSource["ejson"].(map[string]interface{})["privateKeyPath"])
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
		dataSource = fmt.Sprintf("%s", p.DataSource["ejson"].(map[string]interface{})["filePath"])
	} else if vaultAddress != "" && vaultToken != "" {
		dataSource = fmt.Sprintf("%s", p.DataSource["vault"].(map[string]interface{})["secretPath"])
	} else if p.DataSource["file"] != nil {
		dataSource = fmt.Sprintf("%s", p.DataSource["file"].(map[string]interface{})["path"])
	} else {
		logger.Print("returning error exit 1\n")
		return errors.New("exit 1")
	}

	for _, r := range m.Resources() {
		yamlByte, err := r.AsYAML()
		if err != nil {
			logger.Printf("error getting resource as yaml: %v, error: %v\n", r.GetName(), err)
			return err
		}
		output, err := utils.RunGomplate(dataSource, p.Pwd, env, string(yamlByte), logger)
		if err != nil {
			logger.Printf("error executing runGomplate() on dataSource: %v, in directory: %v, error: %v\n", dataSource, p.Pwd, err)
			return err
		}
		res, err := p.rf.RF().FromBytes(output)
		if err != nil {
			logger.Printf("error unmarshalling resource from bytes: %v\n", err)
			return err
		}
		r.SetMap(res.Map())
	}
	return nil
}
