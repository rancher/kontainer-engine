package utils

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

func WriteToFile(data []byte, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ioutil.WriteFile(file, data, 0644)
	}

	tmpfi, err := ioutil.TempFile("", "file.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfi.Name())

	if err = ioutil.WriteFile(tmpfi.Name(), data, 0644); err != nil {
		return err
	}

	if err = tmpfi.Close(); err != nil {
		return err
	}

	if err = os.Remove(file); err != nil {
		return err
	}

	err = os.Rename(tmpfi.Name(), file)
	return err
}

func HomeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}

func DecodePem(data, types string) ([]byte, error) {
	capem, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	capemBlock, _ := pem.Decode(capem)
	if capemBlock == nil || capemBlock.Type != types {
		return nil, errors.New("failed to decode ca.pem")
	}
	return capemBlock.Bytes, nil
}
