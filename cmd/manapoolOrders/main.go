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
	ID            string  `json:"id"`
	CreatedAt     string  `json:"created_at"`
	OrderNumber   string  `json:"order_number"`
	SubtotalCents float64 `json:"subtotal_cents"`
	TaxCents      float64 `json:"tax_cents"`
	ShippingCents float64 `json:"shipping_cents"`
	TotalCents    float64 `json:"total_cents"`
}

type orderDetailResponse struct {
	Order orderDetail `json:"order"`
}

type orderDetail struct {
	ID                 string         `json:"id"`
	CreatedAt          string         `json:"created_at"`
	OrderNumber        string         `json:"order_number"`
	SubtotalCents      float64        `json:"subtotal_cents"`
	TaxCents           float64        `json:"tax_cents"`
	ShippingCents      float64        `json:"shipping_cents"`
	TotalCents         float64        `json:"total_cents"`
	OrderSellerDetails []sellerDetail `json:"order_seller_details"`
}

type sellerDetail struct {
	SellerID       string         `json:"seller_id"`
	SellerUsername string         `json:"seller_username"`
	OrderNumber    string         `json:"order_number"`
	ShippingCents  float64        `json:"shipping_cents"`
	Fulfillments   []fulfillment  `json:"fulfillments"`
	Items          []orderItem    `json:"items"`
	RefundedItems  []refundedItem `json:"refunded_items"`
}

type fulfillment struct {
	Status          string  `json:"status"`
	TrackingURL     *string `json:"tracking_url"`
	TrackingNumber  *string `json:"tracking_number"`
	TrackingCompany string  `json:"tracking_company"`
}

// orderItem is an active (paid) line item. ShippedQuantity may be less than
// Quantity while an order is in flight; it is informational and does not affect
// the money. When Replacement is set, some quantity of the item was swapped for
// a different product.
type orderItem struct {
	OrderItemID     string       `json:"order_item_id"`
	PriceCents      float64      `json:"price_cents"`
	Quantity        int          `json:"quantity"`
	ShippedQuantity int          `json:"shipped_quantity"`
	Product         itemProduct  `json:"product"`
	Replacement     *replacement `json:"replacement"`
}

// replacement describes a substitute product sent in place of (part of) an
// item. The buyer still paid for the original line; RefundedCents credits back
// any value difference when the substitute is worth less. SeparateShipment is
// set (and SamePackage false) when the substitute ships on its own.
type replacement struct {
	Quantity         int          `json:"quantity"`
	SamePackage      bool         `json:"same_package"`
	RefundedCents    *float64     `json:"refunded_cents"`
	Product          itemProduct  `json:"product"`
	SeparateShipment *fulfillment `json:"separate_shipment"`
}

// refundedItem is a line that was refunded, in whole or in part. It is a
// standalone record (disjoint from Items) carrying the refunded amount; the net
// paid for the line is PriceCents*Quantity - RefundedCents.
type refundedItem struct {
	OrderItemID   string      `json:"order_item_id"`
	PriceCents    float64     `json:"price_cents"`
	Quantity      int         `json:"quantity"`
	RefundedCents *float64    `json:"refunded_cents"`
	Product       itemProduct `json:"product"`
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

const manapoolFeeRate = 0.042

func sellerStatus(seller sellerDetail) string {
	if len(seller.Fulfillments) > 0 {
		return seller.Fulfillments[len(seller.Fulfillments)-1].Status
	}
	return ""
}

// replacementRefund is the value credited back on an active line when its
// substitute is worth less than the original.
func replacementRefund(item orderItem) float64 {
	if item.Replacement != nil && item.Replacement.RefundedCents != nil {
		return *item.Replacement.RefundedCents
	}
	return 0
}

// hasAdjustments reports whether a seller carries upstream refund/replacement
// data (refunded_items or a per-item replacement). When it does, that data is
// authoritative and the legacy whole-seller status fallback is not applied.
func hasAdjustments(seller sellerDetail) bool {
	if len(seller.RefundedItems) > 0 {
		return true
	}
	for _, item := range seller.Items {
		if item.Replacement != nil {
			return true
		}
	}
	return false
}

// legacyRefunded reports whether a seller should be treated as wholly refunded
// under the pre-adjustment model. Until an order carries refunded_items /
// replacement data, the only refund signal is a seller-level "refunded"
// fulfillment status.
func legacyRefunded(seller sellerDetail) bool {
	return !hasAdjustments(seller) && sellerStatus(seller) == "refunded"
}

// activeNetLine is the amount actually paid for an active item line: zero if the
// whole seller is legacy-refunded, otherwise the gross line total minus any
// replacement credit, floored at zero.
func activeNetLine(item orderItem, legacyRefund bool) float64 {
	if legacyRefund {
		return 0
	}
	line := item.PriceCents*float64(item.Quantity) - replacementRefund(item)
	if line < 0 {
		line = 0
	}
	return line
}

// refundedNetLine is the amount still paid for a refunded line: the gross line
// total minus the refunded amount, floored at zero (usually zero for a full
// refund, positive for a partial one).
func refundedNetLine(r refundedItem) float64 {
	refunded := 0.0
	if r.RefundedCents != nil {
		refunded = *r.RefundedCents
	}
	line := r.PriceCents*float64(r.Quantity) - refunded
	if line < 0 {
		line = 0
	}
	return line
}

// sellerNetSubtotal is the seller's subtotal after refunds and replacement
// crediting, across both active and refunded lines.
func sellerNetSubtotal(seller sellerDetail) float64 {
	legacy := legacyRefunded(seller)
	var total float64
	for _, item := range seller.Items {
		total += activeNetLine(item, legacy)
	}
	for _, r := range seller.RefundedItems {
		total += refundedNetLine(r)
	}
	return total
}

// shippingBySeller returns the shipping (cents) to report per seller order,
// keyed by seller OrderNumber, plus the order-level effective shipping total.
//
// Each surviving seller's shipping comes straight from its shipping_cents; a
// seller whose items were all refunded contributes nothing. The total is capped
// at the order-level shipping_cents — what the buyer was actually charged — so
// an order-level shipping promo (per-seller values summing above the charge)
// doesn't inflate the report. When the cap bites, the excess is trimmed
// proportionally across sellers so the per-seller rows still sum to the total.
func shippingBySeller(order *orderDetail) (map[string]float64, float64) {
	perSeller := make(map[string]float64, len(order.OrderSellerDetails))
	var kept float64
	for _, s := range order.OrderSellerDetails {
		if sellerNetSubtotal(s) > 0 {
			perSeller[s.OrderNumber] = s.ShippingCents
			kept += s.ShippingCents
		} else {
			perSeller[s.OrderNumber] = 0
		}
	}

	charged := order.ShippingCents
	if kept <= charged || kept <= 0 {
		return perSeller, kept
	}

	// Cap bites: scale each contributing seller down and hand the rounding
	// remainder to the last contributor so the parts sum exactly to charged.
	scale := charged / kept
	var running float64
	var last string
	for _, s := range order.OrderSellerDetails {
		if perSeller[s.OrderNumber] > 0 {
			scaled := math.Floor(perSeller[s.OrderNumber] * scale)
			perSeller[s.OrderNumber] = scaled
			running += scaled
			last = s.OrderNumber
		}
	}
	if last != "" {
		perSeller[last] += charged - running
	}
	return perSeller, charged
}

// emitLine is a single CSV line to write for a seller: an active or refunded
// product line with its net (post-adjustment) amount and derived status.
type emitLine struct {
	product  itemProduct
	quantity int
	net      float64
	status   string
}

// sellerLines flattens a seller into the lines to emit: every active item
// followed by every refunded item, each with its net amount and status.
func sellerLines(seller sellerDetail) []emitLine {
	legacy := legacyRefunded(seller)
	status := sellerStatus(seller)

	var lines []emitLine
	for _, item := range seller.Items {
		st := status
		switch {
		case legacy:
			st = "refunded"
		case item.Replacement != nil:
			st = "replaced"
		}
		lines = append(lines, emitLine{
			product:  item.Product,
			quantity: item.Quantity,
			net:      activeNetLine(item, legacy),
			status:   st,
		})
	}
	for _, r := range seller.RefundedItems {
		net := refundedNetLine(r)
		st := "refunded"
		if net > 0 {
			st = "partially-refunded"
		}
		lines = append(lines, emitLine{
			product:  r.Product,
			quantity: r.Quantity,
			net:      net,
			status:   st,
		})
	}
	return lines
}

// productFields extracts the display columns for a product.
func productFields(p itemProduct) (productType, name, set, number, condition, finish, lang string) {
	productType = p.ProductType
	if p.Single != nil {
		s := p.Single
		return productType, s.Name, s.Set, s.Number, s.ConditionID, s.FinishID, s.LanguageID
	}
	if p.Sealed != nil {
		s := p.Sealed
		return productType, s.Name, s.Set, "", "", "", s.LanguageID
	}
	return productType, "", "", "", "", "", ""
}

func orderToRows(order *orderDetail) [][]string {
	var rows [][]string
	multiSeller := len(order.OrderSellerDetails) > 1

	// Flatten every seller once; fees only apply when every emitted line is a
	// single.
	sellerLineSets := make([][]emitLine, len(order.OrderSellerDetails))
	allSingles := true
	for i, seller := range order.OrderSellerDetails {
		sellerLineSets[i] = sellerLines(seller)
		for _, l := range sellerLineSets[i] {
			if l.product.ProductType != "mtg_single" {
				allSingles = false
			}
		}
	}

	// Compute effective totals from net (post-adjustment) subtotals.
	var effectiveSubtotal float64
	for _, seller := range order.OrderSellerDetails {
		effectiveSubtotal += sellerNetSubtotal(seller)
	}

	// Per-seller shipping (from upstream shipping_cents), capped at the
	// order-level charge.
	sellerShipping, effectiveShipping := shippingBySeller(order)

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

	for i, seller := range order.OrderSellerDetails {
		sellerSub := sellerNetSubtotal(seller)
		firstInSeller := true

		for _, line := range sellerLineSets[i] {
			productType, name, set, number, condition, finish, lang := productFields(line.product)

			itemPrice := 0.0
			if line.quantity > 0 {
				itemPrice = line.net / float64(line.quantity)
			}

			orderNumber, orderDate := "", ""
			total, subtotal, fees, tax, shipping := "", "", "", "", ""

			if multiSeller {
				// Per-seller subtotals on the first item of each seller
				if firstInSeller {
					var sellerFee float64
					if allSingles {
						sellerFee = math.Floor(sellerSub * manapoolFeeRate)
					}
					subtotal = formatCents(sellerSub)
					if sellerFee > 0 {
						fees = formatCents(sellerFee)
					}
					if order.SubtotalCents > 0 {
						tax = formatCents(math.Floor(order.TaxCents * sellerSub / order.SubtotalCents))
					}
					shipping = formatCents(sellerShipping[seller.OrderNumber])
					firstInSeller = false
				}
			} else {
				// Single seller: order-level fields on the first row
				if firstInSeller {
					orderNumber = order.OrderNumber
					orderDate = order.CreatedAt
					subtotal = formatCents(effectiveSubtotal)
					tax = formatCents(effectiveTax)
					shipping = formatCents(effectiveShipping)
					if effectiveFee > 0 {
						fees = formatCents(effectiveFee)
					}
					total = formatCents(effectiveTotal)
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
				line.status,
				productType,
				name,
				set,
				number,
				condition,
				finish,
				lang,
				strconv.Itoa(line.quantity),
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
