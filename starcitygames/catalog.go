package starcitygames

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// scgCatalogURL is the HawkSearch catalog export. It returns the full product
// catalog (both retail price/qty and buylist sell_list_price per variant) as a
// single JSON array, authenticated with an x-api-key header.
const scgCatalogURL = "https://api.starcitygames.com/hawksearch/catalog/download/json"

// catalogAttempts bounds the replays of a broken export, matching the retry
// budget the client gives an ordinary request.
const catalogAttempts = 10

// Product types the catalog reports. Everything SCG sells shares one
// export, so this is what separates cards from boxes from playmats.
const (
	ProductTypeSingles = "Singles"
	ProductTypeSealed  = "Sealed"
)

// CatalogProduct is a single card printing in the catalog export.
type CatalogProduct struct {
	ID              int              `json:"id"`
	SKU             string           `json:"sku"`
	ProductType     string           `json:"product_type"`
	ScryfallID      string           `json:"scryfall_id"`
	TCGPlayerID     string           `json:"tcgplayer_id"`
	URL             string           `json:"url"`
	Name            string           `json:"name"`
	Game            string           `json:"game"`
	Set             string           `json:"set"`
	Finish          string           `json:"finish"`
	FinishGroup     string           `json:"finish_group"`
	Language        string           `json:"language"`
	Rarity          string           `json:"rarity"`
	CollectorNumber string           `json:"collector_number"`
	Variants        []CatalogVariant `json:"variants"`
}

// CatalogVariant is a product in a specific condition, with its own retail
// price/quantity and buylist (sell_list) price.
type CatalogVariant struct {
	ID            int    `json:"id"`
	SKU           string `json:"sku"`
	URL           string `json:"url"`
	Condition     string `json:"condition"`
	Qty           int    `json:"qty"`
	Price         string `json:"price"`
	IsOnDiscount  bool   `json:"is_on_discount"`
	SellListPrice string `json:"sell_list_price"`
}

// DownloadCatalog fetches the catalog export stream. The caller must close the
// returned reader.
func (scg *SCGClient) DownloadCatalog(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scg.catalogURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", scg.apiKey)

	resp, err := scg.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("catalog download failed: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.Body, nil
}

// productRefused marks an error that came out of the product callback rather
// than out of the stream, which tells the replay apart from the one failure
// re-reading the export cannot mend.
type productRefused struct {
	err error
}

func (p productRefused) Error() string { return p.err.Error() }

func (p productRefused) Unwrap() error { return p.err }

// StreamCatalog hands every product in the catalog export to fn, restarting
// the download if the stream breaks partway through. An error fn returns ends
// the stream and comes back as it is; only a broken one is read again.
//
// The export is one long-lived response of a hundred-odd megabytes, and the
// client's own retry can only replay a request that never returned a status.
// A connection dropped after the header - "stream error: stream ID 1;
// INTERNAL_ERROR" is the one seen in practice - lands past it, and used to end
// the run with an empty catalog. Replaying from the top costs a second
// download and is the only recovery available, since the export is not
// resumable; reset undoes whatever the abandoned pass accumulated.
func (scg *SCGClient) StreamCatalog(ctx context.Context, reset func(), fn func(CatalogProduct) error) error {
	var err error
	for attempt := range catalogAttempts {
		if attempt > 0 {
			reset()
		}
		err = scg.streamCatalogOnce(ctx, fn)
		if err == nil {
			return nil
		}
		// A cancelled context is the caller giving up, not a flaky peer.
		if ctx.Err() != nil {
			return err
		}
		// Neither is a product the caller turned down. Downloading the
		// export again would hand it the very same product to turn down
		// again, at a hundred megabytes a go, so it comes straight back.
		var refused productRefused
		if errors.As(err, &refused) {
			return refused.err
		}
	}
	return err
}

func (scg *SCGClient) streamCatalogOnce(ctx context.Context, fn func(CatalogProduct) error) error {
	body, err := scg.DownloadCatalog(ctx)
	if err != nil {
		return err
	}
	defer body.Close()
	return decodeCatalog(body, fn)
}

// decodeCatalog streams the catalog array, invoking fn for each product without
// buffering the whole (large) response in memory.
func decodeCatalog(r io.Reader, fn func(CatalogProduct) error) error {
	dec := json.NewDecoder(r)

	// Opening '['
	if _, err := dec.Token(); err != nil {
		return err
	}
	for dec.More() {
		var p CatalogProduct
		if err := dec.Decode(&p); err != nil {
			return err
		}
		if err := fn(p); err != nil {
			return productRefused{err}
		}
	}
	// Closing ']'
	_, err := dec.Token()
	return err
}

// gameFromCatalog maps the catalog game string to the internal game constant.
// An unknown string maps to 0, which matches no configured scraper, so those
// products are skipped.
func gameFromCatalog(game string) int {
	switch game {
	case "Magic: The Gathering":
		return GameMagic
	case "Flesh and Blood":
		return GameFleshAndBlood
	case "Lorcana":
		return GameLorcana
	case "Riftbound", "Riftbound: League of Legends TCG":
		// The catalog dropped the subtitle in August 2026; accept both
		// spellings so a flip back does not zero the scraper again.
		return GameRiftbound
	default:
		return 0
	}
}

// catalogFoil reports the foil flag from the broad finish grouping. "Non-foil"
// is plain; "Foil" and "Alt Foil" (etched, surge, rainbow, cold, …) are foil.
func catalogFoil(p CatalogProduct) bool {
	return p.FinishGroup != "Non-foil"
}

// catalogHit synthesizes the minimal Hit that preprocess needs from a catalog
// product, used as the fallback when the Scryfall shortcut doesn't apply.
func catalogHit(p CatalogProduct, foil bool) Hit {
	finishType := 1
	if foil {
		finishType = 2
	}
	return Hit{
		Name:                p.Name,
		SetName:             p.Set,
		Language:            p.Language,
		CollectorNumber:     p.CollectorNumber,
		FinishPricingTypeID: finishType,
		Variants:            []Variant{{Sku: p.SKU}},
	}
}

// resolveProduct returns the mtgban card id for a catalog product.
//
// The Scryfall id is authoritative: when present it resolves directly through
// the identifier index, skipping preprocess entirely. Etched is the only
// alt-foil that changes the printing (and only for two sets); it shares the
// plain foil's Scryfall id, so it is detected from the finish name and handed to
// MatchID, which associates the foil id with its etched sibling. Every other
// alt-foil (surge/rainbow/cold) resolves to the plain foil. When the id is
// missing or unresolved, it falls back to the SKU-driven preprocess path.
func resolveProduct(game int, p CatalogProduct) (string, error) {
	id, err := resolveProductID(game, p)
	if err != nil {
		return "", err
	}

	// The inherently foreign sets hold a single printing each - FBB is
	// Italian and 4BB Japanese - while SCG sells them in six or seven
	// languages. Every language other than the one the printing actually
	// is collapses onto that uuid, and the products then fight over the
	// same key. Keep only the language that matches.
	co, cerr := mtgmatcher.GetUUID(id)
	if cerr == nil && !languageMatches(p.Language, co.Language) {
		return "", mtgmatcher.ErrUnsupported
	}
	return id, nil
}

// languageMatches reports whether the language a product is sold in is
// the language of the printing it resolved to. The catalog spells the
// two-part languages with a dash that mtgjson does not use.
func languageMatches(catalogLanguage, cardLanguage string) bool {
	if catalogLanguage == "" {
		catalogLanguage = "English"
	}
	if cardLanguage == "" {
		cardLanguage = "English"
	}
	return strings.EqualFold(strings.ReplaceAll(catalogLanguage, " - ", " "), cardLanguage)
}

// skuSetCode returns the set segment of a catalog sku, which carries
// detail the product's own set name has lost.
func skuSetCode(sku string) string {
	fields := strings.Split(sku, "-")
	if len(fields) < 3 {
		return ""
	}
	return fields[2]
}

// skuNumber returns the collector number segment of a catalog sku,
// which keeps the variant letter the product's number field drops.
func skuNumber(sku string) string {
	fields := strings.Split(sku, "-")
	if len(fields) < 4 {
		return ""
	}
	return fields[3]
}

// fabPrintRunMarkers are the print-run suffixes Star City Games glues onto a
// Flesh and Blood set code in its skus, listed longest first so the trim takes
// the whole marker. The datastore numbers a card with the bare code and crosses
// the run with the treatment instead, so "ARC1036" is its ARC036.
var fabPrintRunMarkers = []string{"12", "1", "2", "U"}

// fabNumbers returns the collector numbers to try for a Flesh and Blood sku,
// most specific first.
//
// The sku carries the datastore's own number split across two segments:
// "SGL-FAB-BVO-005-ENN" is BVO005, which names exactly one printing, while the
// bare "005" is every set's fifth card. That matters because the sets Star City
// Games sells these under are decks and blister packs whose names match no
// datastore set, so nothing else narrows the candidates and the reprints alias.
//
// A number segment that opens with a code of its own ("HER_001" under the
// catch-all PRM) already names its set, and the sku's set segment is only the
// shelf it sits on. The fused pairs join with the separator the datastore
// spells them with, and the letter parts ("019_CC") stay wording.
//
// The bare number stays last rather than being dropped: it is what resolves the
// listings whose set segment is neither a datastore code nor a marked-up one,
// and keeping it last costs nothing since a prefixed number that resolves is
// always the more specific answer.
func fabNumbers(sku string) []string {
	number := skuNumber(sku)
	bare := strings.ReplaceAll(number, "_", " ")
	if number == "" {
		return []string{bare}
	}

	code := skuSetCode(sku)
	parts := strings.Split(number, "_")
	if len(parts) > 1 && isAllLetters(parts[0]) {
		code, parts = parts[0], parts[1:]
	}

	var candidates []string
	for _, prefix := range []string{code, trimPrintRun(code)} {
		candidate := prefixNumber(prefix, parts)
		if candidate != "" && candidate != bare && !slices.Contains(candidates, candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return append(candidates, bare)
}

// fabSingleNumbered reports whether a sku's number segment names exactly one
// collector number, the way "NUU-026" and "PRM-FAB_233" do and the pairs
// ("OMN-048_047") do not.
func fabSingleNumbered(sku string) bool {
	parts := strings.Split(skuNumber(sku), "_")
	if len(parts) > 1 && isAllLetters(parts[0]) {
		parts = parts[1:]
	}
	var count int
	for _, part := range parts {
		if numberedPart(part) {
			count++
		}
	}
	return count == 1
}

// numberedPart reports whether a part of a sku's number segment is a
// collector number rather than the wording the letter parts are.
func numberedPart(part string) bool {
	return part != "" && part[0] >= '0' && part[0] <= '9'
}

// prefixNumber glues the set code onto every numeric part of a sku's number
// segment, pairing them the way the datastore does ("MST002//MST158") and
// leaving the non-numeric parts as the wording they are.
func prefixNumber(code string, parts []string) string {
	if code == "" {
		return ""
	}
	var numbers, words []string
	for _, part := range parts {
		if numberedPart(part) {
			numbers = append(numbers, code+part)
			continue
		}
		words = append(words, part)
	}
	if len(numbers) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(numbers, " // ") + " " + strings.Join(words, " "))
}

// trimPrintRun removes a print-run marker from a sku's set code. The trim is
// only ever a second guess, and it never shortens a code past the three
// letters every Flesh and Blood code is at least: "KSU", "NUU" and "UZU" are
// whole codes that end in a marker letter, so the untrimmed code has to be
// tried first and what the trim leaves has to still look like a code.
func trimPrintRun(code string) string {
	for _, marker := range fabPrintRunMarkers {
		trimmed := strings.TrimSuffix(code, marker)
		if trimmed != code && len(trimmed) >= 3 {
			return trimmed
		}
	}
	return code
}

// fabVariantMarker is the digit Star City Games glues onto a Flesh and Blood
// set code to give the second printing of a collector number a sku of its own:
// EVO056 is War Machine and "SGL-FAB-EVO2-056-ENR" is the Extended Art beside
// it. Nothing else in the catalog divides the two - same set, same number,
// same finish, same rarity - so the marker is the whole of what says a listing
// is not the plain printing.
const fabVariantMarker = "2"

// fabVariantMarked reports whether a sku carries the marker. A code is only
// read as marked where taking the marker off still leaves a code, so that a
// genuine code ending in the digit would be left alone.
func fabVariantMarked(sku string) bool {
	code := skuSetCode(sku)
	if !strings.HasSuffix(code, fabVariantMarker) {
		return false
	}
	return trimPrintRun(code) != code
}

// fabMarkedSibling returns the printing a marked sku names, given the plain
// one the match landed on. The marker says that a second printing of this
// number exists and not which it is, so the datastore is what names it: where
// exactly one sibling carries a treatment the plain printing does not, that is
// what is being sold. Where several do - a number with a Marvel and an
// Extended Art both - the marker cannot choose between them, and the treatment
// the catalog does spell (through the rarity, see fabTiers) has already had
// its say, so the answer already reached stands.
func fabMarkedSibling(id string, p CatalogProduct) string {
	if !fabVariantMarked(p.SKU) {
		return id
	}
	co, err := mtgmatcher.GetUUID(id)
	if err != nil || len(co.PromoTypes) > 0 {
		return id
	}

	var found string
	for _, card := range mtgmatcher.MatchWithNumber(co.Name, co.SetCode, co.Number) {
		if len(card.PromoTypes) == 0 {
			continue
		}
		// A treatment the catalog has a field for is that field's to name.
		// The marker is on the sku of a marvel too, and letting it choose
		// one would price a listing as a marvel on the strength of a digit
		// where the rarity beside it says the ordinary card.
		if fabTiers[card.Rarity] {
			continue
		}
		sibling, serr := mtgmatcher.MatchIDFinish(card.UUID, p.Finish)
		if serr != nil || sibling == id {
			continue
		}
		if found != "" && found != sibling {
			return id
		}
		found = sibling
	}
	if found == "" {
		return id
	}
	return found
}

func isAllLetters(field string) bool {
	if field == "" {
		return false
	}
	for _, r := range field {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// catalogNames are the names Star City Games spells differently from the
// datastore, one entry per misspelling. The matcher used to forgive a stray
// s on any name, which is what carried these - at the price of reading Nest
// Ball as Net Ball and Swoobat as Woobat. A storefront that misspells a name
// says so here instead.
var catalogNames = map[string]string{
	"Bandana of the Blue Beyond": "Bandana of the Blue Beyonds",
}

func resolveProductID(game int, p CatalogProduct) (string, error) {
	// Duel Masters crossover promos are catalogued under Magic but aren't Magic
	// cards, so there's nothing to match; discard them.
	if strings.Contains(p.Name, "(Duel Masters)") {
		return "", mtgmatcher.ErrUnsupported
	}

	if spelled, found := catalogNames[p.Name]; found {
		p.Name = spelled
	}

	foil := catalogFoil(p)
	etched := strings.Contains(strings.ToLower(p.Finish), "etched")

	// Portal printed two versions of six cards. SCG marks them a and b in
	// the sku while mtgjson numbers the second with a d suffix, and sends
	// the same collector number and the same scryfall id for both - an id
	// that names the first version - so the b product has to be steered by
	// its number before the identifiers get a say.
	if game == GameMagic && p.Set == "Portal" && strings.HasSuffix(skuNumber(p.SKU), "b") {
		number := strings.TrimSuffix(skuNumber(p.SKU), "b") + "d"
		if out := mtgmatcher.MatchWithNumber(p.Name, "POR", number); len(out) == 1 {
			return mtgmatcher.MatchID(out[0].UUID, foil, etched)
		}
	}

	// Duel Decks: Anthology reprints four earlier duel decks, and mtgjson
	// keeps them under their original codes. The product's set name says
	// only "Anthology", so the deck it belongs to is read from the sku.
	if game == GameMagic && p.Set == "Duel Decks: Anthology" {
		number := strings.TrimLeft(p.CollectorNumber, "0")
		if out := mtgmatcher.MatchWithNumber(p.Name, skuSetCode(p.SKU), number); len(out) == 1 {
			return mtgmatcher.MatchID(out[0].UUID, foil, etched)
		}
	}

	// The authoritative identifiers resolve directly through the identifier
	// index, regardless of game: Scryfall id first, then the TCGplayer id
	// (MatchID resolves a bare product id through the external-id index and
	// applies the finish exactly like the scryfall path). Etched is the only
	// alt-foil that changes the printing; every other alt-foil shares the plain
	// foil's id. (SCG sends null ids today, so in practice this fires only once
	// they start populating them.)
	for _, id := range []string{p.ScryfallID, p.TCGPlayerID} {
		if id == "" {
			continue
		}
		if out, err := mtgmatcher.MatchID(id, foil, etched); err == nil {
			return out, nil
		}
	}

	// Magic needs catalog-specific fixups before the generic matcher.
	if game == GameMagic {
		// SCG's "-WAR2-" is the War of the Spark Japanese planeswalker
		// (jpwalker), whose Japanese-language Scryfall id isn't in the index and
		// which preprocess rejects as non-english. It maps to WAR #NNN★.
		if strings.Contains(p.SKU, "-WAR2-") {
			num := strings.TrimLeft(p.CollectorNumber, "0") + "★"
			if out := mtgmatcher.MatchWithNumber(p.Name, "WAR", num); len(out) == 1 {
				if id, err := mtgmatcher.MatchID(out[0].UUID, foil, false); err == nil {
					return id, nil
				}
			}
		}

		card, err := preprocess(catalogHit(p, foil))
		if err != nil {
			return "", err
		}
		// Inherently foreign sets (Foreign Black Border, Rinascimento, ...)
		// store the foreign printing as their canonical card, so a resolved id
		// whose primary language isn't English is the right match. Match's
		// English-only language validation would reject it, so use it directly.
		// English-primary cards fall through so a foreign single isn't wrongly
		// collapsed onto the English printing.
		if card.ID != "" {
			if co, e := mtgmatcher.GetUUID(card.ID); e == nil && co.Language != "" && co.Language != "English" {
				return mtgmatcher.MatchID(card.ID, foil, etched)
			}
		}
		return mtgmatcher.Match(card)
	}

	// Flesh and Blood reads its number off the sku instead: the segments
	// keep the set code, the fused-card pair ("077_112"), the promo-pack
	// prefix ("JDG_001") and the variant letter ("155b") that the product's
	// bare number field drops. The candidates run most specific first, so
	// the bare number only decides what the set-prefixed one could not.
	// The catalog's own finish name rides beside the flag: a product is one
	// printing in one treatment, and only the name says which.
	if game == GameFleshAndBlood {
		edition, finish := fabPrintRun(p.Set, p.Finish)
		numbers := fabNumbers(p.SKU)
		id, err := fabMatch(p.Name, edition, finish, p.Rarity, foil, numbers)
		if err == nil {
			return fabMarkedSibling(id, p), nil
		}
		// A product named by both its faces at a single collector number
		// is one printing plus the token printed on its back, not a
		// two-numbered double-sided card: the catalog writes "Pass Over //
		// Inner Chi" for NUU026, which the datastore files under the front
		// face alone. The number segment is what tells the two apart, and
		// it has to hold exactly one number - the genuine double-sided
		// cards carry a pair ("048_047") and a front-face retry would
		// flatten them onto the ordinary single.
		front, _, twoFaced := strings.Cut(p.Name, " // ")
		if twoFaced && fabSingleNumbered(p.SKU) {
			retry, rerr := fabMatch(front, edition, finish, p.Rarity, foil, numbers)
			if rerr == nil {
				return fabMarkedSibling(retry, p), nil
			}
		}
		return "", err
	}

	// Lorcana reads its number off the sku, which spells it more fully than
	// the product's own number field does.
	if game == GameLorcana {
		return resolveLorcana(p, foil)
	}

	// Riftbound identifies a card by name + collector number + finish; the
	// catalog set narrows same-name-and-number collisions across sets.
	return mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:      p.Name,
		Edition:   p.Set,
		Variation: p.CollectorNumber,
		Foil:      foil,
	})
}

// secondBucketMarker is what Star City Games appends to a sku's number segment
// to give a listing a second product record of its own.
const secondBucketMarker = "_CC"

// secondBucket reports whether a sku is the second record of a listing the
// catalog also carries plainly.
func secondBucket(sku string) bool {
	return strings.HasSuffix(skuNumber(sku), secondBucketMarker)
}

// bucketKey names the listing a record belongs to: the sku with the marker
// taken off its number segment, which both records of a pair answer with and
// no other sku does. It is what lets the two be folded together whichever of
// them the catalog streams first.
func bucketKey(sku string) string {
	fields := strings.Split(sku, "-")
	if len(fields) < 4 {
		return sku
	}
	fields[3] = strings.TrimSuffix(fields[3], secondBucketMarker)
	return strings.Join(fields, "-")
}

// lorcanaNumber returns the collector number to match a Lorcana product by.
//
// The sku's number segment is the more specific of the two numbers a product
// carries. It keeps the printing marker the number field drops ("117M" beside
// its "117"), and for a promotional printing it spells the number under the
// promo series that issued it ("P3_031"), where the number field sometimes
// carries the whole undivided segment and sometimes only the tail. The series
// is a heading rather than part of the number - the datastore numbers that
// card 31 in Winterspell - so only what follows the last underscore is the
// number, and a segment whose tail names no number at all (the "T03" of a
// token) leaves the product's own field to answer.
//
// More specific is not the same as more reliable, and the sku only speaks
// where the product's own number does not contradict it: a stray digit
// ("0142" on the 042 it prices) and a marker the datastore numbers as a card
// of its own ("032B" for the 33 it files) each name a different card rather
// than the same card more precisely.
func lorcanaNumber(p CatalogProduct) string {
	number := skuNumber(p.SKU)
	idx := strings.LastIndexByte(number, '_')
	if idx >= 0 {
		number = number[idx+1:]
	}
	if number == "" || number[0] < '0' || number[0] > '9' {
		return p.CollectorNumber
	}
	digits := numberDigits(p.CollectorNumber)
	if digits != "" && digits != numberDigits(number) {
		return p.CollectorNumber
	}
	return number
}

// numberDigits returns the number a collector number opens with, as the
// datastore writes it: the leading run of digits without its padding.
func numberDigits(number string) string {
	end := 0
	for end < len(number) && number[end] >= '0' && number[end] <= '9' {
		end++
	}
	return strings.TrimLeft(number[:end], "0")
}

// lorcanaFinish names a Lorcana treatment the way the datastore names it, and
// answers "" for the ones that need no naming.
//
// Star City Games sells a Lorcana treatment under a marketing name of its own,
// and the name is the set's rather than the treatment's: its "Inkwash Foil" is
// the datastore's Lava in one set, Magma in another and VerticalWave in a
// third. None of that has to be translated, because every printing behind
// those names is sold in one foil and the foil flag already reaches it.
//
// One does. A printing sold in its standard foil and in a second one beside it
// has two foils for the flag to choose between, and the flag always picks the
// standard - so the second foil's sku and the standard's landed on one uuid,
// and a $16.78 buylist competed with a $12.56 one on the same card. That
// second foil is the datastore's RainbowPillars throughout, which the catalog
// names Rainbow Foil, and naming it is what separates the two skus. A printing
// not sold in it is unaffected: the name reaches no uuid and the flag decides
// as before.
func lorcanaFinish(finish string) string {
	if finish == "Rainbow Foil" {
		return "RainbowPillars"
	}
	return ""
}

// lorcanaMarker returns the letters a Lorcana collector number ends in, which
// name a printing beside the one the digits alone name.
func lorcanaMarker(number string) string {
	return strings.TrimLeft(number, "0123456789")
}

// resolveLorcana returns the mtgban card id for a Lorcana catalog product.
//
// A number ending in letters names a printing beside the base one, and the
// datastore answers that in either of two ways. It numbers the sibling with
// the same letter, the way Into the Inklands numbers the five Dalmatian
// Puppies "4a" through "4e", and then the number resolves it outright. Or it
// numbers both the same and tells them apart in the name instead - "Bucky -
// Squirrel Squeak Tutor (Errata Version)" is its own printing at Rise of the
// Floodborn 73 - and then the number falls back to the base card and the two
// products land on one uuid, where a $12.99 errata printing prices a $0.39
// common.
//
// So a marker the resolved printing does not wear is asked of the name: the
// printing at that set and number whose name is the product's plus a suffix
// is the sibling the marker names. A marker with no such printing behind it
// is refused rather than folded onto the base card, which is a product Star
// City Games sells and the datastore does not carry - a missing price, where
// folding it in corrupts the price of a card that is carried.
func resolveLorcana(p CatalogProduct, foil bool) (string, error) {
	number := lorcanaNumber(p)
	id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
		Name:      p.Name,
		Edition:   p.Set,
		Variation: number,
		Foil:      foil,
		Finish:    lorcanaFinish(p.Finish),
	})
	if err != nil {
		return "", err
	}
	if lorcanaMarker(number) == "" {
		return id, nil
	}
	co, cerr := mtgmatcher.GetUUID(id)
	if cerr != nil || strings.EqualFold(co.Number, strings.TrimLeft(number, "0")) {
		return id, nil
	}
	return lorcanaSibling(p, co, foil)
}

// otherFormats are what the Lorcana datastore adds to a card's name to file
// something that is not that card: a jumbo print of it, and a lot of several
// cards sold as one. Both sit at the base card's own set and number, so the
// set and the number cannot tell them from the printing a marker asks for.
var otherFormats = []string{"(Oversized)", "(Set of "}

// namesAnotherFormat reports whether what a longer name adds to a base one
// names a different physical product rather than another printing of the same
// card. Star City Games sells a jumbo card as a listing of its own, never as a
// marked variant of the card, so a marker must not be routed onto one.
func namesAnotherFormat(extension string) bool {
	for _, format := range otherFormats {
		if strings.Contains(extension, format) {
			return true
		}
	}
	return false
}

// lorcanaSibling answers with the printing the datastore files at the same set
// and number under a longer name, which is where it puts the errata reprint a
// sku marks with a letter. Exactly one such name may answer: two would leave
// the marker naming neither in particular.
func lorcanaSibling(p CatalogProduct, base *mtgmatcher.CardObject, foil bool) (string, error) {
	missing := fmt.Errorf("no printing beside %s %s for the sku marker", base.SetCode, base.Number)
	uuids, err := mtgmatcher.SearchHasPrefix(p.Name)
	if err != nil {
		return "", missing
	}
	var candidates []*mtgmatcher.CardObject
	for _, uuid := range uuids {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		if !siblingCandidate(base, co) {
			continue
		}
		candidates = append(candidates, co)
	}
	found := soleSibling(candidates)
	if found == nil {
		return "", missing
	}
	return mtgmatcher.MatchID(found.UUID, foil, false)
}

// siblingCandidate reports whether a printing can be the one a sku marker
// names beside its base: the same slot in the same set, spelled as the
// base's name extended - and not extended into another physical format,
// which is a different product wearing the same slot.
func siblingCandidate(base, co *mtgmatcher.CardObject) bool {
	if co.Sealed || co.Name == base.Name {
		return false
	}
	if co.SetCode != base.SetCode || co.Number != base.Number {
		return false
	}
	return !namesAnotherFormat(strings.TrimPrefix(co.Name, base.Name))
}

// soleSibling answers the printing a set of candidates agrees on, and nothing
// where they name two cards, which would leave the marker naming neither in
// particular. They are counted by name because one printing is spelled once
// per treatment it is sold in, and those are the same card.
func soleSibling(candidates []*mtgmatcher.CardObject) *mtgmatcher.CardObject {
	var found *mtgmatcher.CardObject
	for _, co := range candidates {
		if found != nil && found.Name != co.Name {
			return nil
		}
		found = co
	}
	return found
}

// fabMatch answers the first printing one of the sku's collector numbers
// names under the given name, the candidates running most specific first.
// fabTiers are the rarities that name a printing rather than describe one.
// A marvel is a separate product sharing its card's number and treatment, so
// a listing that does not say which of the two it is has not been told apart:
// the catalog sells the Dynasty marvel of Construct Nitro Mechanoid beside
// the ordinary cold foil at DYN092, and only this word divides them.
var fabTiers = map[string]bool{
	"Marvel": true,
}

func fabMatch(name, edition, finish, rarity string, foil bool, numbers []string) (string, error) {
	var err error
	for _, number := range numbers {
		if fabTiers[rarity] {
			number = strings.TrimSpace(number + " " + rarity)
		}
		var id string
		id, err = mtgmatcher.Match(&mtgmatcher.InputCard{
			Name:      name,
			Edition:   edition,
			Variation: number,
			Finish:    finish,
			Foil:      foil,
		})
		if err == nil {
			return id, nil
		}
	}
	return "", err
}

// fabPrintRun moves the print run out of a Flesh and Blood set name and into
// the finish. The catalog spells the run as part of the set, "Tales of Aria
// (1st Edition)", where the datastore keeps one set and crosses the run with
// the treatment, so the two runs of a card are two printings and the set name
// alone reaches neither in particular. A set without a run is left as it is.
func fabPrintRun(set, finish string) (string, string) {
	var run string
	switch {
	case strings.HasSuffix(set, " (1st Edition)"):
		set, run = strings.TrimSuffix(set, " (1st Edition)"), "1st Edition"
	case strings.HasSuffix(set, " (Unlimited)"):
		set, run = strings.TrimSuffix(set, " (Unlimited)"), "Unlimited Edition"
	default:
		return set, finish
	}
	// The catalog names the plain treatment for what it is not; the
	// datastore names it for what it is.
	if finish == "" || finish == "Non-foil" {
		finish = "Normal"
	}
	return set, run + " " + finish
}

// catalogCondition maps a catalog condition string to an mtgban grade.
func catalogCondition(condition string) (string, error) {
	switch condition {
	case "Near Mint":
		return "NM", nil
	case "Played":
		return "SP", nil
	case "Heavily Played":
		return "HP", nil
	default:
		return "", fmt.Errorf("unknown condition %q", condition)
	}
}
