package mtgban

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

func writeVendorInfoToDB(vendor Vendor, db *sql.DB, flags ...bool) error {
	inf := vendor.Info()
	query := fmt.Sprintf(`
		INSERT IGNORE INTO vendors ( name, vendor_id, country, metadata, credit )
		VALUES ( '%s', '%s', '%s', %v, %v )`,
		inf.Name, inf.Shorthand, inf.CountryFlag, inf.MetadataOnly, !inf.NoCredit)

	if len(flags) > 0 {
		if flags[0] {
			query = strings.Replace(query, "INSERT IGNORE", "REPLACE", 1)
		}
	}

	insert, err := db.Query(query)
	if err != nil {
		return err
	}
	return insert.Close()
}

func writeCardToDB(uuid string, db *sql.DB, flags ...bool) error {
	co, err := mtgmatcher.GetUUID(uuid)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT IGNORE INTO cards ( ban_id, mtgjson_id, set_code )
		VALUES ( '%s', '%s', '%s' )`,
		uuid, co.Card.UUID, co.SetCode)

	if len(flags) > 0 {
		if flags[0] {
			query = strings.Replace(query, "INSERT IGNORE", "REPLACE", 1)
		}
	}

	insert, err := db.Query(query)
	if err != nil {
		return err
	}
	return insert.Close()
}

func WriteBuylistToDB(vendor Vendor, db *sql.DB) error {
	buylist, err := vendor.Buylist()
	if err != nil {
		return err
	}
	if len(buylist) == 0 {
		return nil
	}

	err = writeVendorInfoToDB(vendor, db)
	if err != nil {
		return err
	}

	vendorId := vendor.Info().Shorthand

	blDate := vendor.Info().BuylistTimestamp.UTC()
	for uuid, entry := range buylist {
		err = writeCardToDB(uuid, db)
		if err != nil {
			log.Println(err)
			continue
		}

		// Check if there is a record for this card/price, looking at date
		query := fmt.Sprintf(`
			SELECT date
			FROM prices
			WHERE ban_id = %d`, uuid)
		var date time.Time
		_ = db.QueryRow(query).Scan(&date)

		// If there is, update with the new values
		// else insert a new record
		if DateEqual(date, blDate) {
			query = fmt.Sprintf(`
					UPDATE prices
					SET amount = %f, date = '%s', qty = %d
					WHERE ban_id = %d AND vendor_id = %d AND date = '%s'`,
				entry.BuyPrice, entry.Quantity, blDate.Format("2006-01-02 03:04:05"),
				uuid, vendorId, date.Format("2006-01-02 03:04:05"))
		} else {
			query = fmt.Sprintf(`
					INSERT INTO prices ( ban_id, vendor_id, amount, qty, date )
					VALUES ( '%s', '%s', %f, %d, '%s' )`,
				uuid, vendorId, entry.BuyPrice, entry.Quantity, blDate.Format("2006-01-02 03:04:05"))
		}

		write, err := db.Query(query)
		if err != nil {
			log.Println(err)
			continue
		}
		write.Close()
	}

	return nil
}
