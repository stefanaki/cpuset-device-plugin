package plugin

import (
	"encoding/json"
	"os"
)

type State struct {
	Allocations []Allocation `json:"allocations"`
}

func LoadFromFile(filename string) (*State, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	state := &State{}
	err = json.Unmarshal(data, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *State) SaveToFile(filename string) error {
	stateJSON, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, stateJSON, 0644)
}
