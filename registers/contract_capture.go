package registers

import (
	"github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
	"strings"
)

type CaptureContractWrapper struct {
	contracts map[string]map[string]string
	directory string

	log zerolog.Logger
}

var _ RegisterGetWrapper = &CaptureContractWrapper{}

func NewCaptureContractWrapper(directory string, log zerolog.Logger) *CaptureContractWrapper {
	return &CaptureContractWrapper{
		directory: directory,
		contracts: make(map[string]map[string]string),
		log:       log,
	}
}

func (c *CaptureContractWrapper) Wrap(inner RegisterGetRegisterFunc) RegisterGetRegisterFunc {
	return func(owner string, key string) (flow.RegisterValue, error) {
		val, err := inner(owner, key)
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(key, "code.") {
			address := flow.BytesToAddress([]byte(owner)).HexWithPrefix()
			contractName := strings.TrimPrefix(key, "code.")

			if _, ok := c.contracts[address]; !ok {
				c.contracts[address] = make(map[string]string)
			}

			c.contracts[address][contractName] = string(val)
		}

		return val, nil
	}
}

func (c *CaptureContractWrapper) Close() error {
	for account, contracts := range c.contracts {
		for name, code := range contracts {
			filename := filepath.Join(c.directory, account, name+".cdc")
			err := os.MkdirAll(filepath.Dir(filename), os.ModePerm)
			if err != nil {
				return err
			}
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			_, err = file.WriteString(code)
			if err != nil {
				return err
			}
		}

	}
	return nil
}
