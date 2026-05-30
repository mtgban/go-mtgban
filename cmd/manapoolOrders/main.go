// Command manapoolOrders dumps Mana Pool buyer orders to yearly CSV files.
//
// It fetches all orders from the Mana Pool buyer API, retrieves the detail
// for each order to get line items, and writes a CSV file per year named
// "dump-YEAR.csv".
//
// Usage:
//
//	MANAPOOL_EMAIL=user@example.com MANAPOOL_TOKEN=mpat_xxx manapoolOrders
//	MANAPOOL_EMAIL=user@example.com MANAPOOL_TOKEN=mpat_xxx manapoolOrders -year 2025
//	MANAPOOL_EMAIL=user@example.com MANAPOOL_TOKEN=mpat_xxx manapoolOrders -since 2024
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	apiBase  = "https://manapool.com/api/v1"
	maxLimit = 100
)

// --- API types ---

type ordersResponse struct {
	Orders []orderSummary `json:"orders"`
}

type orderSummary struct {
	ID           string  `json:"id"`
	CreatedAt    string  `json:"created_at"`
	OrderNumber  string  `json:"order_number"`
	SubtotalCents float64 `json:"subtotal_cents"`
	TaxCents     float64 `json:"tax_cents"`
	ShippingCents float64 `json:"shipping_cents"`
	TotalCents   float64 `json:"total_cents"`
}

type orderDetailResponse struct {
	Order orderDetail `json:"order"`
}

type orderDetail struct {
	ID                string              `json:"id"`
	CreatedAt         string              `json:"created_at"`
	OrderNumber       string              `json:"order_number"`
	SubtotalCents     float64             `json:"subtotal_cents"`
	TaxCents          float64             `json:"tax_cents"`
	ShippingCents     float64             `json:"shipping_cents"`
	TotalCents        float64             `json:"total_cents"`
	OrderSellerDetails []sellerDetail     `json:"order_seller_details"`
}

type sellerDetail struct {
	SellerID       string        `json:"seller_id"`
	SellerUsername string        `json:"seller_username"`
	OrderNumber    string        `json:"order_number"`
	Fulfillments   []fulfillment `json:"fulfillments"`
	Items          []orderItem   `json:"items"`
}

type fulfillment struct {
	Status          string  `json:"status"`
	TrackingURL     *string `json:"tracking_url"`
	TrackingNumber  *string `json:"tracking_number"`
	TrackingCompany string  `json:"tracking_company"`
}

type orderItem struct {
	PriceCents float64     `json:"price_cents"`
	Quantity   int         `json:"quantity"`
	Product    itemProduct `json:"product"`
}

type itemProduct struct {
	ProductType string      `json:"product_type"`
	ProductID   string      `json:"product_id"`
	Single      *singleInfo `json:"single"`
	Sealed      *sealedInfo `json:"sealed"`
}

type singleInfo struct {
	ScryfallID  string `json:"scryfall_id"`
	MtgjsonID   string `json:"mtgjson_id"`
	Name        string `json:"name"`
	Set         string `json:"set"`
	Number      string `json:"number"`
	LanguageID  string `json:"language_id"`
	ConditionID string `json:"condition_id"`
	FinishID    string `json:"finish_id"`
}

type sealedInfo struct {
	MtgjsonID  string `json:"mtgjson_id"`
	Name       string `json:"name"`
	Set        string `json:"set"`
	LanguageID string `json:"language_id"`
}

// --- API client ---

type client struct {
	http  *http.Client
	email string
	token string
}

func newClient(email, token string) *client {
	rc := retryablehttp.NewClient()
	rc.Logger = nil
	rc.RetryMax = 5
	rc.RetryWaitMin = 2 * time.Second
	rc.RetryWaitMax = 30 * time.Second
	return &client{
		http:  rc.StandardClient(),
		email: email,
		token: token,
	}
}

func (c *client) get(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-ManaPool-Email", c.email)
	req.Header.Set("X-ManaPool-Access-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *client) listOrders(ctx context.Context, since time.Time) ([]orderSummary, error) {
	var all []orderSummary
	offset := 0

	for {
		url := fmt.Sprintf("%s/buyer/orders?since=%s&limit=%d&offset=%d",
			apiBase, since.Format(time.RFC3339), maxLimit, offset)

		var resp ordersResponse
		if err := c.get(ctx, url, &resp); err != nil {
			return nil, fmt.Errorf("list orders (offset %d): %w", offset, err)
		}

		if len(resp.Orders) == 0 {
			break
		}

		all = append(all, resp.Orders...)
		log.Printf("Fetched %d orders (total %d)", len(resp.Orders), len(all))

		if len(resp.Orders) < maxLimit {
			break
		}
		offset += maxLimit
	}

	return all, nil
}

func (c *client) getOrder(ctx context.Context, id string) (*orderDetail, error) {
	url := fmt.Sprintf("%s/buyer/orders/%s", apiBase, id)
	var resp orderDetailResponse
	if err := c.get(ctx, url, &resp); err != nil {
		return nil, fmt.Errorf("get order %s: %w", id, err)
	}
	return &resp.Order, nil
}

// --- CSV output ---

var csvHeader = []string{
	"order_number",
	"order_date",
	"order_total",
	"order_subtotal",
	"manapool_fees",
	"order_tax",
	"order_shipping",
	"seller",
	"seller_order_number",
	"fulfillment_status",
	"product_type",
	"card_name",
	"set",
	"number",
	"condition",
	"finish",
	"language",
	"quantity",
	"item_price",
}

const (
	manapoolFeeRate = 0.042
	freeShippingThreshold = 4999 // cents ($49.99)
	shippingFirstClass    = 130  // cents ($1.30)
	shippingGround        = 599  // cents ($5.99)
)

func sellerItemCount(seller sellerDetail) int {
	count := 0
	for _, item := range seller.Items {
		count += item.Quantity
	}
	return count
}

// assignShipping distributes the total order shipping across sellers.
// Each seller gets 0 (subtotal > $49.99), $1.30, or $5.99.
//
// Priority: first assign known values (free for large orders,
// $5.99 for MagicStronghold or >14 items), then solve the remainder
// with $1.30 and $5.99, assigning $5.99 last.
func assignShipping(sellers []sellerDetail, totalShipping float64) map[string]float64 {
	result := map[string]float64{}
	remaining := totalShipping

	// First pass: assign 0 to sellers over the free shipping threshold,
	// and 5.99 to known ground-shipping sellers
	var undecided []string
	for _, s := range sellers {
		sub := sellerSubtotal(s)
		if sub > freeShippingThreshold {
			result[s.OrderNumber] = 0
		} else if s.SellerUsername == "MagicStronghold" || sellerItemCount(s) > 14 {
			result[s.OrderNumber] = shippingGround
			remaining -= shippingGround
		} else {
			undecided = append(undecided, s.OrderNumber)
		}
	}

	if len(undecided) == 0 {
		return result
	}

	// Solve: how many 599s and 130s sum to remaining?
	// 599*x + 130*(n-x) = remaining
	// 469*x = remaining - 130*n
	n := len(undecided)
	numGround := 0
	diff := remaining - float64(shippingFirstClass*n)
	if diff > 0 && int(diff)%469 == 0 {
		numGround = int(diff) / 469
	}

	// Assign 1.30 first, 5.99 last
	for i, orderNum := range undecided {
		if i >= n-numGround {
			result[orderNum] = shippingGround
			remaining -= shippingGround
		} else {
			result[orderNum] = shippingFirstClass
			remaining -= shippingFirstClass
		}
	}

	// If we couldn't solve it exactly, adjust the last seller
	if remaining != 0 && len(undecided) > 0 {
		last := undecided[len(undecided)-1]
		result[last] += remaining
	}

	return result
}

func sellerStatus(seller sellerDetail) string {
	if len(seller.Fulfillments) > 0 {
		return seller.Fulfillments[len(seller.Fulfillments)-1].Status
	}
	return ""
}

func sellerSubtotal(seller sellerDetail) float64 {
	var total float64
	for _, item := range seller.Items {
		total += item.PriceCents * float64(item.Quantity)
	}
	return total
}

func orderToRows(order *orderDetail) [][]string {
	var rows [][]string
	multiSeller := len(order.OrderSellerDetails) > 1

	// Check if all items are singles to compute fees
	allSingles := true
	for _, seller := range order.OrderSellerDetails {
		for _, item := range seller.Items {
			if item.Product.ProductType != "mtg_single" {
				allSingles = false
			}
		}
	}

	// Compute per-seller shipping
	sellerShipping := assignShipping(order.OrderSellerDetails, order.ShippingCents)

	// Compute effective totals excluding refunded sellers
	var effectiveSubtotal float64
	var effectiveShipping float64
	for _, seller := range order.OrderSellerDetails {
		if sellerStatus(seller) == "refunded" {
			continue
		}
		effectiveSubtotal += sellerSubtotal(seller)
		effectiveShipping += sellerShipping[seller.OrderNumber]
	}

	var effectiveFee float64
	if allSingles && effectiveSubtotal > 0 {
		effectiveFee = math.Floor(effectiveSubtotal * manapoolFeeRate)
	}

	var effectiveTax float64
	if order.SubtotalCents > 0 {
		effectiveTax = math.Floor(order.TaxCents * effectiveSubtotal / order.SubtotalCents)
	}

	effectiveTotal := effectiveSubtotal + effectiveFee + effectiveTax + effectiveShipping

	if multiSeller {
		// Emit a summary row for the whole order
		rows = append(rows, []string{
			order.OrderNumber,
			order.CreatedAt,
			formatCents(effectiveTotal),
			formatCents(effectiveSubtotal),
			formatCents(effectiveFee),
			formatCents(effectiveTax),
			formatCents(effectiveShipping),
			"(multiple sellers)",
			"", "", "", "", "", "", "", "", "", "", "",
		})
	}

	for _, seller := range order.OrderSellerDetails {
		status := sellerStatus(seller)
		refunded := status == "refunded"

		firstInSeller := true

		for _, item := range seller.Items {
			name, set, number, condition, finish, lang := "", "", "", "", "", ""
			productType := item.Product.ProductType

			if item.Product.Single != nil {
				s := item.Product.Single
				name = s.Name
				set = s.Set
				number = s.Number
				condition = s.ConditionID
				finish = s.FinishID
				lang = s.LanguageID
			} else if item.Product.Sealed != nil {
				s := item.Product.Sealed
				name = s.Name
				set = s.Set
				lang = s.LanguageID
			}

			// Zero out item price for refunded sellers
			itemPrice := item.PriceCents
			if refunded {
				itemPrice = 0
			}

			orderNumber, orderDate := "", ""
			total, subtotal, fees, tax, shipping := "", "", "", "", ""

			if multiSeller {
				// Per-seller subtotals on the first item of each seller
				if firstInSeller {
					if refunded {
						subtotal = "0.00"
						fees = "0.00"
						tax = "0.00"
						shipping = "0.00"
					} else {
						sub := sellerSubtotal(seller)
						var sellerFee float64
						if allSingles {
							sellerFee = math.Floor(sub * manapoolFeeRate)
						}
						subtotal = formatCents(sub)
						if sellerFee > 0 {
							fees = formatCents(sellerFee)
						}
						taxPortion := order.TaxCents * sub / order.SubtotalCents
						tax = formatCents(math.Floor(taxPortion))
						shipping = formatCents(sellerShipping[seller.OrderNumber])
					}
					firstInSeller = false
				}
			} else {
				// Single seller: order-level fields on the first row
				if firstInSeller {
					orderNumber = order.OrderNumber
					orderDate = order.CreatedAt
					if refunded {
						subtotal = "0.00"
						tax = "0.00"
						shipping = "0.00"
						fees = "0.00"
						total = "0.00"
					} else {
						subtotal = formatCents(effectiveSubtotal)
						tax = formatCents(effectiveTax)
						shipping = formatCents(effectiveShipping)
						if effectiveFee > 0 {
							fees = formatCents(effectiveFee)
						}
						total = formatCents(effectiveTotal)
					}
					firstInSeller = false
				}
			}

			rows = append(rows, []string{
				orderNumber,
				orderDate,
				total,
				subtotal,
				fees,
				tax,
				shipping,
				seller.SellerUsername,
				seller.OrderNumber,
				status,
				productType,
				name,
				set,
				number,
				condition,
				finish,
				lang,
				strconv.Itoa(item.Quantity),
				formatCents(itemPrice),
			})
		}
	}
	return rows
}

func formatCents(cents float64) string {
	return fmt.Sprintf("%.2f", cents/100)
}

// --- main ---

// parseDate parses a date string in YYYY, YYYY-MM, or YYYY-MM-DD format
// and returns the corresponding time. For partial dates, missing fields
// default to the start (month=1, day=1).
func parseDate(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q (expected YYYY, YYYY-MM, or YYYY-MM-DD)", s)
}

// endOfPeriod returns the exclusive upper bound for a date string.
// "2025" → 2026-01-01, "2025-03" → 2025-04-01, "2025-03-15" → 2025-03-16.
func endOfPeriod(s string) (time.Time, error) {
	t, err := parseDate(s)
	if err != nil {
		return time.Time{}, err
	}
	switch len(s) {
	case 4: // YYYY
		return t.AddDate(1, 0, 0), nil
	case 7: // YYYY-MM
		return t.AddDate(0, 1, 0), nil
	default: // YYYY-MM-DD
		return t.AddDate(0, 0, 1), nil
	}
}

func main() {
	sinceOpt := flag.String("since", "", "Start date: YYYY, YYYY-MM, or YYYY-MM-DD (default: current year)")
	toOpt := flag.String("to", "", "End date: YYYY, YYYY-MM, or YYYY-MM-DD (default: now)")
	yearOpt := flag.String("year", "", "Shorthand for -since YYYY -to YYYY")
	flag.Parse()

	email := os.Getenv("MANAPOOL_EMAIL")
	token := os.Getenv("MANAPOOL_TOKEN")
	if email == "" || token == "" {
		log.Fatal("MANAPOOL_EMAIL and MANAPOOL_TOKEN environment variables are required")
	}

	now := time.Now()

	// Resolve -year as shorthand for -since/-to
	if *yearOpt != "" {
		*sinceOpt = *yearOpt
		*toOpt = *yearOpt
	}

	// Parse start
	var since time.Time
	if *sinceOpt != "" {
		var err error
		since, err = parseDate(*sinceOpt)
		if err != nil {
			log.Fatalf("-since: %v", err)
		}
	} else {
		since = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	}

	// Parse end (exclusive upper bound)
	var until time.Time
	if *toOpt != "" {
		var err error
		until, err = endOfPeriod(*toOpt)
		if err != nil {
			log.Fatalf("-to: %v", err)
		}
	} else {
		until = now.Add(24 * time.Hour) // include today
	}

	ctx := context.Background()
	c := newClient(email, token)

	log.Printf("Fetching orders from %s to %s", since.Format("2006-01-02"), until.Format("2006-01-02"))

	orders, err := c.listOrders(ctx, since)
	if err != nil {
		log.Fatalf("list orders: %v", err)
	}
	log.Printf("Found %d orders total", len(orders))

	// Build filename from the date range
	sinceTag := since.Format("2006-01-02")
	untilTag := until.AddDate(0, 0, -1).Format("2006-01-02")
	filename := fmt.Sprintf("dump-%s.csv", sinceTag)
	if sinceTag != untilTag {
		filename = fmt.Sprintf("dump-%s_%s.csv", sinceTag, untilTag)
	}
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("create %s: %v", filename, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write(csvHeader)

	for i, summary := range orders {
		t, err := time.Parse(time.RFC3339Nano, summary.CreatedAt)
		if err != nil {
			log.Printf("WARNING: cannot parse date %q for order %s, skipping", summary.CreatedAt, summary.OrderNumber)
			continue
		}

		if t.Before(since) || !t.Before(until) {
			continue
		}

		// Fetch order detail
		detail, err := c.getOrder(ctx, summary.ID)
		if err != nil {
			log.Printf("WARNING: %v, skipping", err)
			continue
		}

		rows := orderToRows(detail)
		for _, row := range rows {
			w.Write(row)
		}

		if (i+1)%10 == 0 {
			log.Printf("Processed %d/%d orders", i+1, len(orders))
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatalf("csv flush error: %v", err)
	}
	log.Printf("Wrote %s", filename)
	log.Println("Done")
}
