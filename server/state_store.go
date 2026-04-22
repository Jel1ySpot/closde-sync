package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
)

const credentialsStateKey = "credentials"

var claudeSettingsSyncKeys = []string{
	"userID",
	"firstStartTime",
	"oauthAccount",
	"claudeCodeFirstTokenDate",
	"groveConfigCache",
	"passesEligibilityCache",
	"overageCreditGrantCache",
	"claudeAiMcpEverConnected",
}

type State struct {
	Tokens []string       `json:"tokens"`
	State  map[string]any `json:"state"`
}

type stateStore struct {
	mu       sync.Mutex
	filePath string
}

func newStateStore(filePath string) *stateStore {
	return &stateStore{filePath: filePath}
}

func (s *stateStore) Load() (State, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Tokens: []string{}, State: map[string]any{}}, nil
		}
		return State{}, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	state.Tokens = normalizeTokens(state.Tokens)
	state.State = normalizeJSONObject(state.State)
	return state, nil
}

func (s *stateStore) Save(state State) error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}

	state.Tokens = normalizeTokens(state.Tokens)
	state.State = normalizeJSONObject(state.State)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, append(data, '\n'), 0o644)
}

func (s *stateStore) AddToken(token string, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.Load()
	if err != nil {
		return "", err
	}

	resolvedToken := strings.TrimSpace(token)
	resolvedName := strings.TrimSpace(name)
	if resolvedToken == "" {
		resolvedToken, err = generateToken()
		if err != nil {
			return "", err
		}
	}
	if resolvedName != "" {
		resolvedToken = resolvedName + "-" + resolvedToken
	}

	for _, existingToken := range state.Tokens {
		if existingToken == resolvedToken {
			return resolvedToken, nil
		}
	}

	state.Tokens = append(state.Tokens, resolvedToken)
	if err := s.Save(state); err != nil {
		return "", err
	}
	return resolvedToken, nil
}

func (s *stateStore) ReplaceState(nextState map[string]any) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.Load()
	if err != nil {
		return State{}, err
	}
	state.State = normalizeJSONObject(nextState)
	if err := s.Save(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *stateStore) SyncStateFromClaudeFiles(claudeSettingsFile string, claudeCredentialsFile string) (State, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	configFileExists, err := pathExists(s.filePath)
	if err != nil {
		return State{}, false, err
	}
	state, err := s.Load()
	if err != nil {
		return State{}, false, err
	}

	mergedState, err := loadMergedClaudeState(claudeSettingsFile, claudeCredentialsFile)
	if err != nil {
		return State{}, false, err
	}

	stateChanged := !reflect.DeepEqual(state.State, mergedState)
	state.State = mergedState
	if !stateChanged && configFileExists {
		return state, false, nil
	}
	if err := s.Save(state); err != nil {
		return State{}, false, err
	}
	return state, true, nil
}

func loadMergedClaudeState(claudeSettingsFile string, claudeCredentialsFile string) (map[string]any, error) {
	settings, err := readJSONObjectFile(claudeSettingsFile)
	if err != nil {
		return nil, err
	}
	credentials, err := readJSONObjectFile(claudeCredentialsFile)
	if err != nil {
		return nil, err
	}

	mergedState := make(map[string]any)
	for _, key := range claudeSettingsSyncKeys {
		if value, ok := settings[key]; ok {
			mergedState[key] = normalizeJSONValue(value)
		}
	}
	if len(credentials) > 0 {
		mergedState[credentialsStateKey] = normalizeJSONObject(credentials)
	}
	return normalizeJSONObject(mergedState), nil
}

func pathExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func readJSONObjectFile(filePath string) (map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return map[string]any{}, nil
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}

	object, err := ensureJSONObject(value)
	if err != nil {
		return nil, errors.New(filePath + " must contain a JSON object")
	}
	return object, nil
}

func writeClaudeStateFiles(claudeSettingsFile string, claudeCredentialsFile string, nextState map[string]any) error {
	currentSettings, err := readJSONObjectFile(claudeSettingsFile)
	if err != nil {
		return err
	}

	nextSettings := cloneJSONObject(currentSettings)
	for _, key := range claudeSettingsSyncKeys {
		if value, ok := nextState[key]; ok {
			nextSettings[key] = cloneJSONValue(value)
		} else {
			delete(nextSettings, key)
		}
	}

	nextCredentials := map[string]any{}
	if rawCredentials, ok := nextState[credentialsStateKey].(map[string]any); ok {
		nextCredentials = cloneJSONObject(rawCredentials)
	}

	if err := writeJSONObjectFile(claudeSettingsFile, nextSettings); err != nil {
		return err
	}
	return writeJSONObjectFile(claudeCredentialsFile, nextCredentials)
}

func writeJSONObjectFile(filePath string, value map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(normalizeJSONObject(value), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, append(data, '\n'), 0o644)
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func normalizeTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(tokens))
	normalized := make([]string, 0, len(tokens))
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func normalizeJSONObject(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	normalized := make(map[string]any, len(value))
	for key, item := range value {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey != "" {
			normalized[trimmedKey] = normalizeJSONValue(item)
		}
	}
	return normalized
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeJSONObject(typed)
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeJSONValue(item))
		}
		return normalized
	default:
		return typed
	}
}

func ensureJSONObject(value any) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("JSON body must be an object")
	}
	return normalizeJSONObject(object), nil
}
