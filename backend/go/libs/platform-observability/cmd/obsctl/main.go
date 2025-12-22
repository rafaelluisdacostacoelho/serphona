package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// simple HTTP client reused across calls
var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	tempoURL := flag.String("tempo-url", "http://tempo:3200", "Tempo base URL (http[s]://host:port)")
	tenant := flag.String("tenant", "", "Tenant (X-Scope-OrgID)")
	authToken := flag.String("auth-token", "", "Bearer token for Tempo/Grafana APIs")
	service := flag.String("service", "", "Service name filter for search")
	limit := flag.Int("limit", 20, "Max traces to return on search")
	traceID := flag.String("trace-id", "", "Trace ID for get command")
	query := flag.String("query", "", "TraceQL query for search")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
	}

	cmd := args[0]
	switch cmd {
	case "search":
		q := strings.TrimSpace(*query)
		if q == "" && *service != "" {
			q = fmt.Sprintf("{ service.name = \"%s\" }", *service)
		}
		if q == "" {
			q = "{ service.name != \"\" }"
		}
		if err := searchTraces(*tempoURL, q, *tenant, *authToken, *limit); err != nil {
			fmt.Fprintf(os.Stderr, "search error: %v\n", err)
			os.Exit(1)
		}
	case "get":
		if *traceID == "" {
			fmt.Fprintln(os.Stderr, "--trace-id is required for get")
			usage()
		}
		if err := getTrace(*tempoURL, *traceID, *tenant, *authToken); err != nil {
			fmt.Fprintf(os.Stderr, "get error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "obsctl usage:")
	fmt.Fprintln(os.Stderr, "  obsctl [flags] search   --service <name> [--query <traceql>] [--limit N]")
	fmt.Fprintln(os.Stderr, "  obsctl [flags] get      --trace-id <id>")
	fmt.Fprintln(os.Stderr, "  Common flags: --tempo-url http://tempo:3200 --tenant <org> --auth-token <token>")
	os.Exit(2)
}

func searchTraces(tempoURL, traceql, tenant, token string, limit int) error {
	base := strings.TrimRight(tempoURL, "/")
	v := url.Values{}
	v.Set("query", traceql)
	if limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", limit))
	}

	req, err := http.NewRequest(http.MethodGet, base+"/api/search?"+v.Encode(), nil)
	if err != nil {
		return err
	}
	addHeaders(req, tenant, token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("search failed: %s: %s", resp.Status, string(body))
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return err
	}

	if len(sr.Traces) == 0 {
		fmt.Println("no traces found")
		return nil
	}

	fmt.Println("traceID\tservice\tduration_ms")
	for _, t := range sr.Traces {
		fmt.Printf("%s\t%s\t%d\n", t.TraceID, t.RootServiceName, t.DurationMs)
	}
	return nil
}

func getTrace(tempoURL, traceID, tenant, token string) error {
	base := strings.TrimRight(tempoURL, "/")
	req, err := http.NewRequest(http.MethodGet, base+"/api/traces/"+url.PathEscape(traceID), nil)
	if err != nil {
		return err
	}
	addHeaders(req, tenant, token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("get trace failed: %s: %s", resp.Status, string(body))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return err
	}

	fmt.Printf("trace %s fetched (%d bytes)\n", traceID, len(data))
	return nil
}

func addHeaders(req *http.Request, tenant, token string) {
	if tenant != "" {
		req.Header.Set("X-Scope-OrgID", tenant)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

type searchResponse struct {
	Traces []traceSummary `json:"traces"`
}

type traceSummary struct {
	TraceID         string `json:"traceID"`
	RootServiceName string `json:"rootServiceName"`
	DurationMs      int64  `json:"durationMs"`
}
