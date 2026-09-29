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

// readHistory reads, from the newspaper's daily snapshots of CK's price list,
// every snapshot since start of every product CK bought at $1 or more in that
// time, and the oldest and newest day read. The session is read-only.
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
		SELECT date, ck_id, is_foil, price_buy, quantity_buying, quantity_selling
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
		var foil sql.NullBool
		var buy sql.NullFloat64
		var buying, stock sql.NullInt64
		err := rows.Scan(&date, &id, &foil, &buy, &buying, &stock)
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
		day := int32(date.Sub(start).Hours() / 24)
		if first < 0 || day < first {
			first = day
		}
		last = max(last, day)
		p.Days = append(p.Days, snapshot{Day: day, Buy: buy.Float64, Buying: int32(buying.Int64), Stock: int32(stock.Int64)})
	}
	err = rows.Err()
	if err != nil {
		return nil, 0, 0, err
	}
	for _, p := range products {
		slices.SortFunc(p.Days, func(a, b snapshot) int { return int(a.Day - b.Day) })
	}
	return products, max(first, 0), last, nil
}
