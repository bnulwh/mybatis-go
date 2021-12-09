package utils

import (
	log "github.com/bnulwh/logrus"
	"os"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func MakeDirAll(dir string) error {
	ok, err := PathExists(dir)
	if err != nil {
		log.Errorf("check dir %s failed.%v", dir, err)
		return err
	}
	if !ok {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Errorf("create dir %s failed. %v", dir, err)
			return err
		}
	}
	return nil
}
