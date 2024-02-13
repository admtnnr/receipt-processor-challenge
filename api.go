package fetch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// API represents the Fetch API server and its collective endpoints. The API
// stores submitted receipts in memory and are not persisted across restarts,
// but is safe for concurrent use.
type API struct {
	mux      *http.ServeMux
	mu       sync.RWMutex
	receipts map[string]*Receipt
}

// ProcessReceiptRequest is the request body that is submitted to the
// [ProcessReceipt] endpoint.
type ProcessReceiptRequest struct {
	// Retailer is the name of the seller where the purchase was made.
	Retailer string `json:"retailer"`
	// PurchaseDate is the date that the purchase was made, e.g "2006-01-02".
	PurchaseDate string `json:"purchaseDate"`
	// PurchaseTime is the time that the purchase was made. The time should be
	// represented in 24-hour time format without timezone, e.g. "14:30".
	PurchaseTime string `json:"purchaseTime"`
	// Items are the individual line items on the receipt.
	Items []ProcessReceiptItem `json:"items"`
	// Total is the sum of all costs of line items on the receipt, represented
	// as a string monetary value, e.g. "15.30".
	Total string `json:"total"`
}

// ProcessReceiptItem is an individual line item in [ProcessReceiptRequest].
type ProcessReceiptItem struct {
	// ShortDescription is the description of the line item.
	ShortDescription string `json:"shortDescription"`
	// Price represents the cost of the line item, represented as a string
	// monetary value, e.g. "2.50".
	Price string `json:"price"`
}

// ProcessReceiptResponse is the response body that is returned from
// the [ProcessReceipt] endpoint.
type ProcessReceiptResponse struct {
	// ID is the unique ID of the receipt.
	ID string `json:"id"`
}

// GetPointsResponse is the response body that is returned from the
// [GetPoints] endpoint.
type GetPointsResponse struct {
	// Points are the number of Fetch rewards points assigned to the receipt.
	Points int `json:"points"`
}

// Error is the response body that is returned from API endpoints when the
// request could not be completed successfully.
type Error struct {
	// Message is the human-readable error message.
	Message string `json:"error"`
}

// NewAPI creates a new Fetch API.
func NewAPI() *API {
	api := &API{
		mux:      http.NewServeMux(),
		receipts: make(map[string]*Receipt),
	}

	api.mux.HandleFunc("/receipts/process", api.ProcessReceipt)
	api.mux.HandleFunc("/receipts/{id}/points", api.GetPoints)

	return api
}

// ServeHTTP serves as the entrypoint of the API for an [http.Server].
func (api *API) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	api.mux.ServeHTTP(rw, req)
}

// Error writes the HTTP response with the given status and message in the
// error response body.
func (api *API) Error(rw http.ResponseWriter, status int, format string, args ...any) error {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)

	return json.NewEncoder(rw).Encode(&Error{
		Message: fmt.Sprintf(format, args...),
	})
}

// ProcessReceipt is an [http.HandlerFunc] that receives a request representing
// a receipt, processes the receipt, assigns its point value, and stores the
// receipt in non-durable storage for retrieval.
func (api *API) ProcessReceipt(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		api.Error(rw, http.StatusMethodNotAllowed, "invalid request method, must be 'POST'")
		return
	}

	var prreq ProcessReceiptRequest
	if err := json.NewDecoder(req.Body).Decode(&prreq); err != nil {
		api.Error(rw, http.StatusBadRequest, "failed to parse process receipt request, %v", err)
		return
	}

	receipt, err := receiptFrom(&prreq)
	if err != nil {
		api.Error(rw, http.StatusBadRequest, "invalid process receipt request, %v", err)
		return
	}

	api.mu.Lock()
	api.receipts[receipt.ID] = receipt
	api.mu.Unlock()

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(&ProcessReceiptResponse{
		ID: receipt.ID,
	})
}

// GetPoints is an [http.HandlerFunc] that returns the point value for a receipt
// specified by the `id` path parameter.
//
// If no receipt exists for the given `id` the endpoint responds with `404 Not
// Found`.
func (api *API) GetPoints(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		api.Error(rw, http.StatusMethodNotAllowed, "invalid request method, must be 'GET'")
		return
	}

	id := req.PathValue("id")
	if id == "" {
		api.Error(rw, http.StatusBadRequest, "missing receipt ID")
		return
	}

	api.mu.RLock()
	receipt, ok := api.receipts[id]
	api.mu.RUnlock()

	if !ok {
		api.Error(rw, http.StatusNotFound, "no receipt with ID %q exists", id)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(&GetPointsResponse{
		Points: receipt.Points,
	})
}

// receiptFrom creates a new [Receipt] from the [ProcessReceiptRequest].
func receiptFrom(req *ProcessReceiptRequest) (*Receipt, error) {
	receipt, err := NewReceipt()
	if err != nil {
		return nil, fmt.Errorf("failed to create receipt, %w", err)
	}

	receipt.Retailer = req.Retailer

	if receipt.Purchased, err = parsePurchased(req.PurchaseDate, req.PurchaseTime); err != nil {
		return nil, fmt.Errorf("invalid purchase date/time, %w", err)
	}

	for _, item := range req.Items {
		price, err := parseAmount(item.Price)
		if err != nil {
			return nil, fmt.Errorf("invalid item price %q, %w", item.Price, err)
		}

		receipt.Items = append(receipt.Items, ReceiptItem{
			Description: item.ShortDescription,
			Price:       price,
		})
	}

	if receipt.Total, err = parseAmount(req.Total); err != nil {
		return nil, fmt.Errorf("invalid receipt total %q, %w", receipt.Total, err)
	}

	receipt.Points = CalculatePoints(receipt)

	return receipt, nil
}

// parsePurchased parses date strings in the format "2006-01-02" and 24-hour
// time strings in the format "13:30" and converts them into a single
// [time.Time] representation.
func parsePurchased(purchaseDate, purchaseTime string) (time.Time, error) {
	purchased, err := time.Parse("2006-01-02", purchaseDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse purchase date %q, %w", purchaseDate, err)
	}

	var hours, minutes int
	if _, err := fmt.Sscanf(purchaseTime, "%d:%d", &hours, &minutes); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse purchase time %q, %w", purchaseTime, err)
	}

	if hours < 0 || hours > 23 {
		return time.Time{}, fmt.Errorf("invalid hour value '%d', must be >= 0 and <= 23", hours)
	}
	if minutes < 0 || minutes > 59 {
		return time.Time{}, fmt.Errorf("invalid minute value '%d', must be >= 0 and <= 59", hours)
	}

	purchased = purchased.
		Add(time.Duration(hours) * time.Hour).
		Add(time.Duration(minutes) * time.Minute)

	return purchased, nil
}

// parseAmount parses a string representing a money value and converts it to an
// integer representing the value as cents, e.g. "67.10" to 6710.
func parseAmount(amount string) (int, error) {
	var dollars, cents int
	if _, err := fmt.Sscanf(amount, "%d.%d", &dollars, &cents); err != nil {
		return 0, fmt.Errorf("failed to parse amount %q, %w", amount, err)
	}

	// Truncate fractional cents if present.
	return dollars*100 + cents%100, nil
}
