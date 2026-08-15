package agent

import (
	"net"
	"testing"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestSettingsMaskKeepOnEmptyAndClear(t *testing.T) {
	orig := lookupIP
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("1.1.1.1")}, nil
	}
	t.Cleanup(func() { lookupIP = orig })
	store := NewMemoryStore()
	st, err := PutSettings(store, 9, SettingsPut{
		BaseURL: "https://api.x.ai/v1",
		Model:   "grok-4.5",
		ApiKey:  "sk-secret-1234",
	})
	require.NoError(t, err)
	require.True(t, ModelConfigured(st))
	view := ViewSettings(st)
	require.True(t, view.ApiKeySet)
	require.Equal(t, "****1234", view.ApiKeyMasked)
	require.NotContains(t, view.ApiKeyMasked, "secret")
	require.True(t, view.ModelConfigured)

	st2, err := PutSettings(store, 9, SettingsPut{
		BaseURL: "https://api.x.ai/v1",
		Model:   "grok-4.5",
		ApiKey:  "",
	})
	require.NoError(t, err)
	require.Equal(t, "sk-secret-1234", st2.ApiKey)
	require.True(t, ModelConfigured(st2))

	st3, err := PutSettings(store, 9, SettingsPut{ApiKeyClear: true})
	require.NoError(t, err)
	require.Empty(t, st3.ApiKey)
	require.False(t, ModelConfigured(st3))
	require.False(t, ViewSettings(st3).ApiKeySet)
}

func TestModelConfiguredRequiresApiKeyOnly(t *testing.T) {
	require.False(t, ModelConfigured(nil))
	require.False(t, ModelConfigured(&models.AgentUserSettings{BaseURL: "https://api.x.ai/v1", Model: "m"}))
	require.True(t, ModelConfigured(&models.AgentUserSettings{ApiKey: "k"}))
}

func TestEffectiveFillsCodeDefaults(t *testing.T) {
	eff := Effective(&models.AgentUserSettings{ApiKey: "k"})
	require.Equal(t, DefaultBaseURL, eff.BaseURL)
	require.Equal(t, DefaultModel, eff.Model)
	require.Equal(t, DefaultMaxTurns, eff.MaxTurns)
	require.Equal(t, DefaultWriteMode, eff.WriteMode)
	require.NotNil(t, eff.Temperature)
	require.Equal(t, DefaultTemperature, *eff.Temperature)
}

func TestPutSettingsRequiresBaseURLAndModel(t *testing.T) {
	orig := lookupIP
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("1.1.1.1")}, nil
	}
	t.Cleanup(func() { lookupIP = orig })
	store := NewMemoryStore()
	_, err := PutSettings(store, 5, SettingsPut{ApiKey: "k"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "baseUrl")
	_, err = PutSettings(store, 5, SettingsPut{BaseURL: "https://api.openai.com/v1", ApiKey: "k"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")
}

func TestPutSettingsAcceptsCustomReasoningEffort(t *testing.T) {
	orig := lookupIP
	lookupIP = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("1.1.1.1")}, nil
	}
	t.Cleanup(func() { lookupIP = orig })
	store := NewMemoryStore()
	st, err := PutSettings(store, 3, SettingsPut{
		BaseURL:         "https://api.openai.com/v1",
		Model:           "gpt-4o-mini",
		ReasoningEffort: "xhigh",
	})
	require.NoError(t, err)
	require.Equal(t, "xhigh", st.ReasoningEffort)
	_, err = PutSettings(store, 3, SettingsPut{ReasoningEffort: "not valid"})
	require.Error(t, err)
}

func TestPutSettingsAllowsPrivateURLWhenFlagSet(t *testing.T) {
	store := NewMemoryStore()
	allow := true
	st, err := PutSettings(store, 2, SettingsPut{
		BaseURL:         "http://127.0.0.1:11434/v1",
		Model:           "llama",
		ApiKey:          "local",
		AllowPrivateLLM: &allow,
	})
	require.NoError(t, err)
	require.True(t, st.AllowPrivateLLM)
	_, err = PutSettings(store, 2, SettingsPut{
		BaseURL:         "http://127.0.0.1:11434/v1",
		AllowPrivateLLM: new(bool),
	})
	require.Error(t, err)
}
