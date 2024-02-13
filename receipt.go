package fetch

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Receipt represents the purchase of one or more items at a specific retailer
// on a specific date.
type Receipt struct {
	// ID is the UUID of the receipt.
	ID string
	// Retailer is the name of the seller where the purchase was made.
	Retailer string
	// Purchased represents the date and time the purchase was made. The
	// timezone is not captured and should always be set to UTC.
	Purchased time.Time
	// Items are the individual line items on the receipt.
	Items []ReceiptItem
	// Total is the sum of all costs of line items on the receipt, represented
	// as cents. Tax is either not included, or assumed to be incorporated into
	// the cost of individual line items.
	Total int
	// Points are the number of Fetch rewards points assigned to the
	// receipt.
	//
	// Points are typically calculated and assigned based on a set of rules
	// based on the receipt data, but there may be circumstances, such as
	// fraud, returns, customer satisfaction, bugs, etc. where manual
	// adjustments will be required.
	Points int
}

// ReceiptItem is an individual line item on a receipt.
type ReceiptItem struct {
	// Description is the description of the line item.
	Description string
	// Price is the cost of the line item, represented in cents.
	Price int
}

// NewReceipt creates a new receipt with a UUID.
func NewReceipt() (*Receipt, error) {
	id, err := genUUID()
	if err != nil {
		return nil, err
	}

	return &Receipt{
		ID: id,
	}, nil
}

// CalculatePoints determines the number of Fetch rewards points that a given
// receipt is worth based on data points such as the retailer name, purchase
// date and time, items purchased, etc.
//
// CalculatePoints does NOT recalculate points if the given receipt already has
// points assigned to it. We do this to avoid retroactively changing point
// values on an existing receipt if/when the point calculation algorithm
// changes which may cause discrepencies in accounting when comparing points
// spent vs. points earned.
//
// Current Point Rules:
//   - One point for every alphanumeric character in the retailer name.
//   - 50 points if the total is a round dollar amount with no cents.
//   - 25 points if the total is a multiple of 0.25.
//   - 5 points for every two items on the receipt.
//   - If the trimmed length of the item description is a multiple of 3, multiply
//     the price by 0.2 and round up to the nearest integer. The result is the
//     number of points earned.
//   - 6 points if the day in the purchase date is odd.
//   - 10 points if the time of purchase is after 2:00pm and before 4:00pm.
func CalculatePoints(receipt *Receipt) int {
	// Skip point calculation if points are already assigned and return
	// existing point value. If recalcating points is required then the points
	// should be zero'd out manually to make this desire explicit.
	if receipt.Points > 0 {
		return receipt.Points
	}

	var points int

	// One point for every alphanumeric character in the retailer name.
	for _, r := range receipt.Retailer {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			points++
		}
	}

	// 50 points if the total is a round dollar amount with no cents.
	if receipt.Total%100 == 0 {
		points += 50
	}

	// 25 points if the total is a multiple of 0.25.
	if receipt.Total%25 == 0 {
		points += 25
	}

	// 5 points for every two items on the receipt.
	points += 5 * (len(receipt.Items) / 2)

	// If the trimmed length of the item description is a multiple of 3,
	// multiple the prices by 0.2 and round up to the nearest integer.
	for _, item := range receipt.Items {
		if len(strings.TrimSpace(item.Description))%3 != 0 {
			continue
		}

		// Prices are represented as cents, so to keep everything as integer
		// division we divide by 5 instead of multiply by 0.2 and roll in the
		// divide by 100 to convert the cents to points, leaving us with divide
		// by 500.
		points += item.Price / 500

		// Account for the round up for the truncated integer division by
		// checking the remainder and tacking on an extra point if necessary
		// below.
		if item.Price%500 > 0 {
			points++
		}
	}

	// 6 points if the day in the purchase date is odd.
	if receipt.Purchased.Day()%2 != 0 {
		points += 6
	}

	// 10 points if the time of purchase is after 2:00pm and before 4:00pm.
	if hour := receipt.Purchased.Hour(); hour >= 14 && hour < 16 {
		points += 10
	}

	return points
}

// genUUID generates a UUIDv4.
func genUUID() (string, error) {
	id := make([]byte, 16)

	if _, err := rand.Read(id); err != nil {
		return "", fmt.Errorf("failed to read random bytes, %w", err)
	}

	// UUIDv4 spec: set the four MSBs in the 7th byte to 4-bit version "4" and
	// the two MSBs in the 9th byte to 0 and 1.
	// See: https://www.ietf.org/rfc/rfc4122.txt, section 4.4
	//
	//   version bits                        variant bits
	//   x x x x x x x x                     x x x x x x x x
	// & 0 0 0 0 1 1 1 1  (0x0F = 15)      & 0 0 1 1 1 1 1 1 (0x3F = 63)
	//   ---------------                     ---------------
	//   0 0 0 0 x x x x                     0 0 x x x x x x
	// | 0 1 0 0 0 0 0 0  (0x40 = 64)      | 1 0 0 0 0 0 0 0 (0x80 = 128)
	//   ---------------                     ---------------
	//   0 1 0 0 x x x x                     1 0 x x x x x x
	id[6] = (id[6] & 0x0F) | 0x40
	id[8] = (id[8] & 0x3F) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		id[0:4],
		id[4:6],
		id[6:8],
		id[8:10],
		id[10:],
	), nil
}
