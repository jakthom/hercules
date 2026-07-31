package labels_test

import (
	"encoding/json"
	"testing"

	"github.com/jakthom/hercules/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironment(t *testing.T) {
	t.Setenv("HERCULES_LABEL", "resolved")
	resolved, err := (labels.Labels{"env": "$HERCULES_LABEL", "region": "east"}).ResolveEnvironment()
	require.NoError(t, err)
	assert.Equal(t, labels.Labels{"env": "resolved", "region": "east"}, resolved)
}

func TestResolveEnvironmentRejectsMissingAndEmptyReferences(t *testing.T) {
	_, err := (labels.Labels{"env": "$"}).ResolveEnvironment()
	require.Error(t, err)
	_, err = (labels.Labels{"env": "$HERCULES_DOES_NOT_EXIST"}).ResolveEnvironment()
	require.Error(t, err)
}

func TestLabelsAcceptLegacyList(t *testing.T) {
	var decoded labels.Labels
	require.NoError(t, json.Unmarshal([]byte(`[{"env":"test"},{"region":"east"}]`), &decoded))
	assert.Equal(t, labels.Labels{"env": "test", "region": "east"}, decoded)
}

func TestMergeStrictRejectsCollision(t *testing.T) {
	_, err := labels.MergeStrict(labels.Labels{"env": "a"}, labels.Labels{"env": "b"})
	require.Error(t, err)
}

func TestLabelNamesAreSortedAndValidated(t *testing.T) {
	configured := labels.Labels{"z": "1", "a": "2"}
	assert.Equal(t, []string{"a", "z"}, configured.LabelNames())
	assert.NoError(t, configured.ValidateNames())
	assert.Error(t, (labels.Labels{"not-valid": "x"}).ValidateNames())
}
