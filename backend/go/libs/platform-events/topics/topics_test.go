package topics

import (
	"reflect"
	"testing"
)

func TestGetTopicsByGroup(t *testing.T) {
	got := GetTopicsByGroup("auth")
	want := []string{
		UserCreated,
		UserUpdated,
		UserDeleted,
		UserLoggedIn,
		UserLoggedOut,
		PasswordChanged,
		PasswordReset,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("auth topics mismatch: got %v, want %v", got, want)
	}
}

func TestGetTopicsByGroupRAG(t *testing.T) {
	got := GetTopicsByGroup("rag")
	want := []string{RAGIngestionRequested}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rag topics mismatch: got %v, want %v", got, want)
	}
}

func TestGetTopicsByGroupUnknown(t *testing.T) {
	got := GetTopicsByGroup("unknown")
	if got != nil && len(got) != 0 {
		t.Fatalf("expected nil/empty for unknown group, got %v", got)
	}
}

func TestAllTopicsContainsAllGroups(t *testing.T) {
	all := AllTopics()
	seen := make(map[string]bool)
	for _, topic := range all {
		if seen[topic] {
			t.Fatalf("duplicate topic in AllTopics: %s", topic)
		}
		seen[topic] = true
	}

	expectedCount := 0
	for _, groupTopics := range TopicGroups {
		expectedCount += len(groupTopics)
		for _, topic := range groupTopics {
			if !seen[topic] {
				t.Fatalf("topic missing from AllTopics: %s", topic)
			}
		}
	}

	if len(all) != expectedCount {
		t.Fatalf("unexpected topic count; got %d want %d", len(all), expectedCount)
	}
}
