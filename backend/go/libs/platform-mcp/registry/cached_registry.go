package registry

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// Loader exposes tool loading operations used by CachedRegistry.
type Loader interface {
	ListTools(ctx context.Context, tenantID string) ([]protocol.Tool, error)
	DescribeTool(ctx context.Context, tenantID, name string) (protocol.Tool, error)
}

// CachedRegistry wraps a loader with an in-memory cache per tenant with TTL and ETag helpers.
type CachedRegistry struct {
	loader Loader
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
	now   func() time.Time
}

type cacheEntry struct {
	tools   []protocol.Tool
	etag    string
	fetched time.Time
}

// NewCachedRegistry builds a registry that caches per-tenant tool lists with TTL.
func NewCachedRegistry(loader Loader, ttl time.Duration) *CachedRegistry {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedRegistry{
		loader: loader,
		ttl:    ttl,
		cache:  make(map[string]cacheEntry),
		now:    time.Now,
	}
}

// ListTools returns cached tools when fresh, otherwise refreshes via the loader.
func (c *CachedRegistry) ListTools(ctx context.Context, tenantID string) ([]protocol.Tool, error) {
	_, tools, _, err := c.ListToolsWithETag(ctx, tenantID, "")
	return tools, err
}

// DescribeTool returns a tool, leveraging the cached list when available.
func (c *CachedRegistry) DescribeTool(ctx context.Context, tenantID, name string) (protocol.Tool, error) {
	_, tool, _, err := c.DescribeToolWithETag(ctx, tenantID, name, "")
	return tool, err
}

// ListToolsWithETag returns tools and an aggregate ETag; notModified is true when If-None-Match hits the cache.
func (c *CachedRegistry) ListToolsWithETag(ctx context.Context, tenantID, ifNoneMatch string) (string, []protocol.Tool, bool, error) {
	entry, err := c.ensureEntry(ctx, tenantID)
	if err != nil {
		return "", nil, false, err
	}
	if entry.etag != "" && ifNoneMatch != "" && entry.etag == ifNoneMatch {
		return entry.etag, nil, true, nil
	}
	return entry.etag, entry.tools, false, nil
}

// DescribeToolWithETag returns a tool and its ETag; notModified is true when If-None-Match matches the cached ETag.
func (c *CachedRegistry) DescribeToolWithETag(ctx context.Context, tenantID, name, ifNoneMatch string) (string, protocol.Tool, bool, error) {
	entry, err := c.ensureEntry(ctx, tenantID)
	if err != nil {
		return "", protocol.Tool{}, false, err
	}

	tool, found := findTool(entry.tools, name)
	if !found {
		tool, err = c.loader.DescribeTool(ctx, tenantID, name)
		if err != nil {
			return "", protocol.Tool{}, false, err
		}
		entry.tools = append(entry.tools, tool)
		entry.etag = listETag(entry.tools)
		entry.fetched = c.now()
		c.setEntry(tenantID, entry)
	}

	etag := tool.ETag
	if etag != "" && ifNoneMatch != "" && etag == ifNoneMatch {
		return etag, protocol.Tool{}, true, nil
	}
	return etag, tool, false, nil
}

func (c *CachedRegistry) ensureEntry(ctx context.Context, tenantID string) (cacheEntry, error) {
	now := c.now()

	c.mu.RLock()
	entry, ok := c.cache[tenantID]
	if ok && now.Sub(entry.fetched) < c.ttl {
		c.mu.RUnlock()
		return entry, nil
	}
	c.mu.RUnlock()

	tools, err := c.loader.ListTools(ctx, tenantID)
	if err != nil {
		return cacheEntry{}, err
	}
	entry = cacheEntry{tools: tools, etag: listETag(tools), fetched: now}
	c.setEntry(tenantID, entry)
	return entry, nil
}

func (c *CachedRegistry) setEntry(tenantID string, entry cacheEntry) {
	c.mu.Lock()
	c.cache[tenantID] = entry
	c.mu.Unlock()
}

func findTool(tools []protocol.Tool, name string) (protocol.Tool, bool) {
	for _, t := range tools {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return protocol.Tool{}, false
}
