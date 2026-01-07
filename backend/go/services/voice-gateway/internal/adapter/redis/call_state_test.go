package redis

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	redigo "github.com/redis/go-redis/v9"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/voice-gateway/internal/domain/call"
)

func newTestRepo(t *testing.T) (*CallStateRepository, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	client := redigo.NewClient(&redigo.Options{Addr: mr.Addr()})
	repo := NewCallStateRepository(client, time.Minute)
	return repo, mr
}

func TestCallStateRepositorySaveAndGet(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.ConversationID = uuid.New()

	if err := repo.Save(ctx, c); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := repo.Get(ctx, c.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if loaded.ID != c.ID || loaded.ChannelID != c.ChannelID {
		t.Fatalf("loaded call mismatch: %+v", loaded)
	}

	byChannel, err := repo.GetByChannelID(ctx, c.ChannelID)
	if err != nil {
		t.Fatalf("get by channel failed: %v", err)
	}
	if byChannel.ID != c.ID {
		t.Fatalf("expected call %s, got %s", c.ID, byChannel.ID)
	}

	list, err := repo.ListByTenant(ctx, c.TenantID)
	if err != nil {
		t.Fatalf("list by tenant failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("unexpected list result: %+v", list)
	}

	count, err := repo.CountActive(ctx)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected active count > 0")
	}
}

func TestCallStateRepositoryDelete(t *testing.T) {
	repo, _ := newTestRepo(t)
	ctx := context.Background()

	c := call.NewCall(uuid.New(), call.DirectionInbound, "+100", "+200")
	c.ChannelID = "chan-1"
	c.ConversationID = uuid.New()

	if err := repo.Save(ctx, c); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := repo.Get(ctx, c.ID); err == nil {
		t.Fatalf("expected get to fail after delete")
	}
	if _, err := repo.GetByChannelID(ctx, c.ChannelID); err == nil {
		t.Fatalf("expected get by channel to fail after delete")
	}
	list, err := repo.ListByTenant(ctx, c.TenantID)
	if err != nil {
		t.Fatalf("list by tenant failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no calls after delete, got %d", len(list))
	}
}
