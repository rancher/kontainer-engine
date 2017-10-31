package store

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rancher/netes-machine/cluster"
	"github.com/rancher/netes-machine/utils"
)

func GetAllClusterFromStore() (map[string]cluster.Cluster, error) {
	homeDir := filepath.Join(utils.HomeDir(), ".netes", "clusters")
	dir, err := ioutil.ReadDir(homeDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	clusters := map[string]cluster.Cluster{}
	// looks for config.json
	for _, file := range dir {
		if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
			subDir, err := ioutil.ReadDir(filepath.Join(homeDir, file.Name()))
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for _, subFile := range subDir {
				if !subFile.IsDir() && strings.HasSuffix(subFile.Name(), "config.json") {
					cls := cluster.Cluster{}
					data, err := ioutil.ReadFile(filepath.Join(homeDir, file.Name(), subFile.Name()))
					if err != nil {
						return nil, err
					}
					if err := json.Unmarshal(data, &cls); err != nil {
						return nil, err
					}
					clusters[cls.Name] = cls
				}
			}
		}
	}
	return clusters, nil
}
