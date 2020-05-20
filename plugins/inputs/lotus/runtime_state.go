package lotus

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type lotusState struct {
	LastProcessedHeight int `json:"last_processed_height"`
}

const stateFilename = "state.json"

func (l *lotus) statePath(inclFilename bool) string {
	statePath := l.Config_DataPath + "-telegraf"
	if inclFilename {
		return filepath.Join(statePath, stateFilename)
	}
	return statePath
}

func (l *lotus) inflateState() error {
	b, err := ioutil.ReadFile(l.statePath(true))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	state := new(lotusState)
	if err := json.Unmarshal(b, state); err != nil {
		return err
	}

	l.lastHeight = state.LastProcessedHeight
	return nil
}

func (l *lotus) persistState() error {
	if err := os.MkdirAll(l.statePath(false), os.ModePerm); err != nil {
		return err
	}

	state := &lotusState{
		LastProcessedHeight: l.lastHeight,
	}

	serState, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(l.statePath(true), serState, 0644); err != nil {
		return err
	}
	return nil
}
