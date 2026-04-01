package main

import (
	"encoding/json"
	"os"
	"time"
)

const contextStateFile = "context-state.json"

type contextState struct {
	UpdatedAt time.Time          `json:"updated_at"`
	Skills    *activeSkillState  `json:"skills,omitempty"`
	Memory    *activeMemoryState `json:"memory,omitempty"`
}

func contextStatePath() string {
	return coderPath(contextStateFile)
}

func loadContextState() (*contextState, error) {
	data, err := os.ReadFile(contextStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return buildContextStateFromActiveFiles()
		}
		return nil, err
	}

	var state contextState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func buildContextStateFromActiveFiles() (*contextState, error) {
	skills, err := loadActiveSkillState()
	if err != nil {
		return nil, err
	}
	memory, err := loadActiveMemoryState()
	if err != nil {
		return nil, err
	}
	if skills == nil && memory == nil {
		return nil, nil
	}

	return &contextState{
		UpdatedAt: time.Now(),
		Skills:    skills,
		Memory:    memory,
	}, nil
}

func saveContextState(state *contextState) error {
	if err := os.MkdirAll(coderDir, 0755); err != nil {
		return err
	}
	if state == nil {
		state = &contextState{}
	}
	state.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(contextStatePath(), data, 0644)
}

func syncContextState(skills *activeSkillState, memory *activeMemoryState) error {
	state, err := loadContextState()
	if err != nil {
		return err
	}
	if state == nil {
		state = &contextState{}
	}
	if skills != nil {
		state.Skills = skills
	}
	if memory != nil {
		state.Memory = memory
	}
	return saveContextState(state)
}
