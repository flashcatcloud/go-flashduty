package flashduty_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	flashduty "github.com/flashcatcloud/go-flashduty"
	"github.com/flashcatcloud/go-flashduty/retry"
)

// ExampleNewClient shows how to construct a client with a couple of options.
func ExampleNewClient() {
	client, err := flashduty.NewClient(
		"YOUR_APP_KEY",
		flashduty.WithTimeout(30*time.Second),
		flashduty.WithUserAgent("my-app/1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client

	fmt.Println("client ready")
}

// ExampleClient_Incidents lists open incidents and prints a short summary of each.
func ExampleClient_Incidents() {
	ctx := context.Background()

	client, err := flashduty.NewClient("YOUR_APP_KEY")
	if err != nil {
		log.Fatal(err)
	}

	req := &flashduty.ListIncidentsRequest{
		Progress:    "Triggered",
		ListOptions: flashduty.ListOptions{Limit: 20},
	}

	list, resp, err := client.Incidents.List(ctx, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("request_id=%s total=%d\n", resp.RequestID, list.Total)
	for _, inc := range list.Items {
		fmt.Printf("[%s] %s\n", inc.IncidentSeverity, inc.Title)
	}
}

// ExampleClient_Incidents_pagination walks every page using the search-after cursor.
func ExampleClient_Incidents_pagination() {
	ctx := context.Background()

	client, err := flashduty.NewClient("YOUR_APP_KEY")
	if err != nil {
		log.Fatal(err)
	}

	req := &flashduty.ListIncidentsRequest{
		ListOptions: flashduty.ListOptions{Limit: 50},
	}

	// Bound the loop so a misbehaving cursor can never spin forever.
	for page := 0; page < 100; page++ {
		list, resp, err := client.Incidents.List(ctx, req)
		if err != nil {
			log.Fatal(err)
		}

		for _, inc := range list.Items {
			fmt.Printf("%s: %s\n", inc.IncidentID, inc.Title)
		}

		if !resp.HasNextPage {
			break
		}
		// Advance the cursor for the next request.
		req.ListOptions.SearchAfterCtx = list.SearchAfterCtx
	}
}

// ExampleClient_errorHandling distinguishes API errors from rate-limit errors.
func ExampleClient_errorHandling() {
	ctx := context.Background()

	client, err := flashduty.NewClient("YOUR_APP_KEY")
	if err != nil {
		log.Fatal(err)
	}

	_, _, err = client.Incidents.Info(ctx, &flashduty.IncidentInfoRequest{
		IncidentID: "does-not-exist",
	})

	var rl *flashduty.RateLimitError
	if errors.As(err, &rl) {
		// Back off for the duration the server asked for, then retry.
		time.Sleep(rl.RetryAfter)
		return
	}

	var apiErr *flashduty.ErrorResponse
	if errors.As(err, &apiErr) {
		fmt.Printf("api error code=%s request_id=%s\n", apiErr.Code, apiErr.RequestID)
		return
	}

	if err != nil {
		log.Fatal(err)
	}
}

// ExampleWithTransport_retry composes the retry helper as the client transport.
func ExampleWithTransport_retry() {
	// retry.New returns an http.RoundTripper that transparently
	// retries safe requests on 429 and 5xx responses with backoff.
	var rt http.RoundTripper = retry.New()

	client, err := flashduty.NewClient(
		"YOUR_APP_KEY",
		flashduty.WithTransport(rt),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client

	fmt.Println("client with retry transport ready")
}
