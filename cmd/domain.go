package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/qri-io/dataset"
)

func GetDomainList() Domains {
	return Domains{LocalDomain{}}
}

type Domain interface {
	DatasetForAddress(dataset.Address) (*dataset.Dataset, error)
}

type Domains []Domain

func (domains Domains) DatasetForAddress(adr dataset.Address) (ds *dataset.Dataset, err error) {
	for _, domain := range domains {
		if ds, err = domain.DatasetForAddress(adr); err == nil && ds != nil {
			return
		}
	}

	return nil, fmt.Errorf("couldn't find a dataset for address: %s", adr.String())
}

type LocalDomain struct{}

func (l LocalDomain) DatasetForAddress(adr dataset.Address) (ds *dataset.Dataset, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(filepath.Join(dir, "dataset.json")); err == nil {
		bytes, err := ioutil.ReadFile(filepath.Join(dir, "dataset.json"))
		if err != nil {
			return nil, err
		}

		ds = &dataset.Dataset{}
		if err = json.Unmarshal(bytes, ds); err != nil {
			return nil, err
		}

		if d, err := ds.DatasetForAddress(adr); err == nil && d != nil {
			if data, err := d.FetchBytes(dir); err != nil {
				return nil, err
			} else {
				d.Data = data
			}
			return d, err
		}
	}

	return
}
