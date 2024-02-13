package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestIntegration(tt *testing.T) {
	api := NewAPI()

	for _, tc := range []struct {
		name   string
		path   string
		points int
	}{
		{
			name:   "example morning receipt",
			path:   "testdata/morning-receipt.json",
			points: 15,
		},
		{
			name:   "example simple receipt",
			path:   "testdata/simple-receipt.json",
			points: 31,
		},
		{
			name:   "readme target receipt",
			path:   "testdata/readme-target-receipt.json",
			points: 28,
		},
		{
			name:   "readme m&m corner market receipt",
			path:   "testdata/readme-corner-market-receipt.json",
			points: 109,
		},
	} {
		tt.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("failed to open receipt file, got %v, want no error", err)
			}
			defer f.Close()

			var id string

			{
				rw := httptest.NewRecorder()
				req := httptest.NewRequest("POST", "/receipts/process", f)

				api.ServeHTTP(rw, req)

				if rw.Code != http.StatusOK {
					t.Fatalf("failed to process receipt, got %d status code, want 200", rw.Code)
				}

				var got ProcessReceiptResponse
				if err := json.NewDecoder(rw.Body).Decode(&got); err != nil {
					t.Fatalf("failed to parse receipt response, got %v, want no error", err)
				}

				if got.ID == "" {
					t.Fatal("process receipt response returned empty ID, want valid UUID")
				}

				id = got.ID
			}

			{
				rw := httptest.NewRecorder()
				req := httptest.NewRequest("GET", fmt.Sprintf("/receipts/%s/points", id), nil)

				api.ServeHTTP(rw, req)

				if rw.Code != http.StatusOK {
					t.Fatalf("failed to get points, got %d status code, want 200", rw.Code)
				}

				var got GetPointsResponse
				if err := json.NewDecoder(rw.Body).Decode(&got); err != nil {
					t.Fatalf("failed to parse points response, got %v, want no error", err)
				}

				if points := got.Points; points != tc.points {
					t.Fatalf("receipt points do not match, got %d, want %d", points, tc.points)
				}
			}
		})
	}
}
