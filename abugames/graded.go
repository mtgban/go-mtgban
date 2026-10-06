package abugames

import (
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgban"
)

// slabGraders are the grading services ABU names in magic_features.
var slabGraders = []string{"PSA", "BGS", "CGC"}

func isSlab(doc *ABUCard) bool {
	return slices.Contains(doc.Features, "Graded")
}

// slabCondition grades a slab by its bucket, "BGS 9.5" or "CGC 8". The
// overall and subgrades are not read: ABU copies them between slabs. With no
// numeric bucket it falls back on ABU's condition, read as on a plain copy of
// the card (stricter when lowerGrade or foil) and held under the grader's 7
// for "Less than 8". It returns "" and the text it could not grade.
func slabCondition(doc *ABUCard, lowerGrade, foil bool) (mtgban.Condition, string) {
	var grader, score string
	for _, feature := range doc.Features {
		for _, name := range slabGraders {
			bucket, found := strings.CutPrefix(feature, name+" ")
			if found {
				grader, score = name, bucket
			}
		}
	}

	shelf, err := abuRetailCondition(doc.Condition, false, lowerGrade, foil)
	found := err == nil
	switch score {
	case "":
		if !found {
			return "", doc.Condition
		}
		return shelf, ""
	case "Less than 8":
		limit, ok := mtgban.SlabCondition(grader, "7")
		if !found || !ok {
			return "", grader + " " + score
		}
		if slices.Index(mtgban.FullGradeTags, shelf) < slices.Index(mtgban.FullGradeTags, limit) {
			return limit, ""
		}
		return shelf, ""
	}

	grade, ok := mtgban.SlabCondition(grader, score)
	if !ok {
		return "", grader + " " + score
	}
	return grade, ""
}
