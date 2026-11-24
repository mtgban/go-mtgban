package abugames

import (
	"cmp"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
)

// slabGraders are the grading services ABU names in magic_features.
var slabGraders = []string{"PSA", "BGS", "CGC", "TAG"}

// reOtherGrader is a feature naming a service outside slabGraders, alone or
// with its bucket: "SGC", "SGC 9.5".
var reOtherGrader = regexp.MustCompile(`^[A-Z]{3,4}( .+)?$`)

func isSlab(doc *ABUCard) bool {
	return slices.Contains(doc.Features, "Graded")
}

// slabCondition reads a slab off its grader and bucket, "BGS 9.5" or
// "CGC 8", and grades it by the bucket. The overall and subgrades are not
// read: ABU copies them between slabs. "BGS Quad 10", four 10 subgrades, is
// a Black Label 10, and a word beside the score is its label: "CGC Pristine
// 10", a 10 with the number or without it. With no bucket it grades by ABU's
// condition, read as on a plain copy of the card (stricter when lowerGrade or
// foil) and held under the grader's 7 for "Less than 8". A scored TAG slab is
// NM, TAG having no grade table, and one with no bucket is unread. A slab it
// cannot grade is NM, an UnreadSlab, with its wording returned to log; that is
// "" for any other.
func slabCondition(doc *ABUCard, lowerGrade, foil bool) (mtgban.Condition, *mtgban.Slab, string) {
	slab := &mtgban.Slab{}
	var bucket, other string
	for _, feature := range doc.Features {
		for _, name := range slabGraders {
			rest, found := strings.CutPrefix(feature, name+" ")
			if feature == name || found {
				slab.Grader = name
			}
			switch {
			case rest == "Quad 10":
				slab.Label = "Black Label"
				bucket = "10"
			case found:
				bucket = rest
			}
		}
		if len(feature) > len(other) && reOtherGrader.MatchString(feature) {
			other = feature
		}
	}
	if slab.Grader == "" && other != "" {
		return mtgban.NM, mtgban.UnreadSlab("", other), other
	}

	// What a slab it cannot read is published with and logged by: its
	// bucket, and ABU's condition where that is what grades it.
	wording := []string{cmp.Or(slab.Grader, "Graded"), bucket}
	shelf, err := abuRetailCondition(doc.Condition, false, lowerGrade, foil)
	read := err == nil
	switch bucket {
	case "":
		if slab.Grader == "TAG" {
			break
		}
		if read {
			return shelf, slab, ""
		}
		wording = append(wording[:1], doc.Condition)
	case "Less than 8":
		slab.Label = bucket
		limit, ok := mtgban.Slab{Grader: slab.Grader, Score: 7}.Fold()
		if read && ok {
			if slices.Index(mtgban.FullGradeTags, shelf) < slices.Index(mtgban.FullGradeTags, limit) {
				return limit, slab, ""
			}
			return shelf, slab, ""
		}
		wording = append(wording, doc.Condition)
	default:
		var label []string
		for _, field := range strings.Fields(bucket) {
			// add only takes the half steps up to 10, so a score between two
			// of them, or past the 10, is the one below. From 11 up it is a typo.
			score, err := strconv.ParseFloat(field, 64)
			if err == nil && slab.Score == 0 && score >= 1 && score < 11 {
				slab.Score = min(math.Floor(score*2)/2, 10)
				continue
			}
			label = append(label, field)
		}
		slab.Label = cmp.Or(slab.Label, strings.Join(label, " "))
		if slab.Grader == "CGC" && slab.Score == 0 && slab.Label == "Pristine" {
			slab.Score = 10
		}
		grade, ok := slab.Fold()
		if ok {
			return grade, slab, ""
		}
		if slab.Grader == "TAG" && slab.Score > 0 {
			return mtgban.NM, slab, ""
		}
	}

	text := strings.Join(strings.Fields(strings.Join(wording, " ")), " ")
	return mtgban.NM, mtgban.UnreadSlab(slab.Grader, text), text
}
