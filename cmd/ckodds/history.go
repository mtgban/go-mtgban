package main

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"time"

	// The newspaper is Postgres.
	_ "github.com/lib/pq"
)

// readHistory reads, from the newspaper, every snapshot since start of every
// CK product CK paid minBuy or more for in that time, with the TCG Market of
// its printing and finish on each day the newspaper has one, and the oldest
// and newest day read. The session is read-only.
func readHistory(ctx context.Context, line string, start time.Time) (map[string]*product, int32, int32, error) {
	db, err := sql.Open("postgres", line)
	if err != nil {
		return nil, 0, 0, err
	}
	defer db.Close()
	// One connection, so the read-only setting covers every query.
	db.SetMaxOpenConns(1)
	_, err = db.ExecContext(ctx, "SET default_transaction_read_only = on")
	if err != nil {
		return nil, 0, 0, err
	}

	// No ORDER BY: the rows are sorted per product here, rather than the whole
	// history on the database.
	rows, err := db.QueryContext(ctx, `
		SELECT date, ck_id, tcgplayer_id, is_foil, price_buy, quantity_buying, quantity_selling, price_retail_nm
		  FROM cardkingdomproductmodel
		 WHERE date >= $1
		   AND ck_id IN (SELECT ck_id FROM cardkingdomproductmodel
		                  WHERE date >= $1 AND quantity_buying > 0 AND price_buy >= $2
		                  GROUP BY ck_id)`, start, minBuy)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	products := map[string]*product{}
	first, last := int32(-1), int32(0)
	for rows.Next() {
		var date time.Time
		var id int64
		var tcgID sql.NullInt64
		var foil sql.NullBool
		var buy, retail sql.NullFloat64
		var buying, stock sql.NullInt64
		err := rows.Scan(&date, &id, &tcgID, &foil, &buy, &buying, &stock, &retail)
		if err != nil {
			return nil, 0, 0, err
		}
		key := strconv.FormatInt(id, 10)
		p := products[key]
		if p == nil {
			p = &product{ID: key}
			products[key] = p
		}
		p.Foil = p.Foil || foil.Bool
		if tcgID.Valid {
			p.TCGID = tcgID.Int64
		}
		day := int32(date.Sub(start).Hours() / 24)
		if first < 0 || day < first {
			first = day
		}
		last = max(last, day)
		p.Days = append(p.Days, snapshot{
			Day: day, Buy: buy.Float64, Buying: int32(buying.Int64), Stock: int32(stock.Int64), Retail: retail.Float64,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, 0, 0, err
	}
	for _, p := range products {
		slices.SortFunc(p.Days, func(a, b snapshot) int { return int(a.Day - b.Day) })
	}

	err = readMarket(ctx, db, start, products)
	if err != nil {
		return nil, 0, 0, err
	}
	return products, max(first, 0), last, nil
}

// readMarket fills in the TCG Market of each product's printing and finish
// (English), on the days the newspaper has both.
func readMarket(ctx context.Context, db *sql.DB, start time.Time, products map[string]*product) error {
	type tcgKey struct {
		ID   int64
		Foil bool
	}
	byTCG := map[tcgKey][]*product{}
	for _, p := range products {
		if p.TCGID != 0 {
			key := tcgKey{p.TCGID, p.Foil}
			byTCG[key] = append(byTCG[key], p)
		}
	}
	rows, err := db.QueryContext(ctx, `
		SELECT date, product_id, variant = 'Foil', market_price
		  FROM tcgplayerproductpricesmodel
		 WHERE date >= $1 AND language = 'English' AND variant IN ('Normal', 'Foil') AND market_price > 0
		   AND product_id IN (SELECT DISTINCT tcgplayer_id FROM cardkingdomproductmodel
		                       WHERE date >= $1 AND quantity_buying > 0 AND price_buy >= $2)`, start, minBuy)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var date time.Time
		var id int64
		var foil bool
		var market float64
		err := rows.Scan(&date, &id, &foil, &market)
		if err != nil {
			return err
		}
		day := int32(date.Sub(start).Hours() / 24)
		for _, p := range byTCG[tcgKey{id, foil}] {
			i, found := slices.BinarySearchFunc(p.Days, day, func(s snapshot, d int32) int { return int(s.Day - d) })
			if found {
				p.Days[i].Market = market
			}
		}
	}
	return rows.Err()
}
