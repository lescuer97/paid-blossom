package io

import (
	"fmt"
	"os"
	"ratasker/internal/utils"
)

type LocalFSHandler struct {
	DataPath string
}

func MakeFileSystemHandler() (LocalFSHandler, error) {
	var handler LocalFSHandler
	homeDir, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		return handler, fmt.Errorf("utils.GetRastaskerHomeDirectory(). %w", err)
	}

	pathToData := homeDir + "/" + "data"

	err = utils.MakeSureFilePathExists(pathToData, "")
	if err != nil {
		return handler, fmt.Errorf(`utils.MakeSureFilePathExists(pathToData, ""). %w`, err)
	}

	handler.DataPath = pathToData

	return handler, nil
}

func (l LocalFSHandler) WriteBlob(blob []byte) error {
	err := os.WriteFile(l.DataPath, blob, 0764)
	if err != nil {
		return fmt.Errorf(`os.WriteFile(l.DataPath, blob, 0764). %w`, err)
	}
	return nil
}

func (l LocalFSHandler) GetBlob(path string) ([]byte, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return fileBytes, fmt.Errorf(`os.ReadFile(path). %w`, err)
	}
	return fileBytes, nil
}

func (l LocalFSHandler) RemoveBlob(path string) error {
	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf(`os.Remove(path) %w`, err)
	}
	return nil

}

func (l LocalFSHandler) GetStoragePath() string {
	return l.DataPath
}
