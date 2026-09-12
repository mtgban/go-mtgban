package mtgmatcher

import "strings"

// IDValidationOptions controls whether a known identifier may reach a finish
// filed as a separate printing. Ordinary MatchIDFinish stays within one printing.
type IDValidationOptions struct {
	AllowFinishSiblings bool
}

// ValidateID checks a candidate UUID against the supplied name, language and
// finish without changing the input. Unlike Match, it cannot replace conflicting
// wording or clamp an unavailable finish. Descriptors require ValidatePrinting;
// vendors may apply their own documented exceptions before that check.
func (b *Backend) ValidateID(in InputCard, options IDValidationOptions) (string, error) {
	co, err := b.GetUUID(in.ID)
	if err != nil {
		return "", err
	}
	front, _, split := strings.Cut(co.Name, " // ")
	if in.Name == "" || !(Equals(in.Name, co.Name) || Equals(in.Name, co.FaceName) || Equals(in.Name, co.FlavorName) || Equals(in.Name, co.FaceFlavorName) || (split && Equals(in.Name, front))) {
		return "", ErrCardDoesNotExist
	}
	requestedLanguage := in.Language
	in.normalizeLanguage()
	// Match represents the English code as an unspecified language. Strict
	// validation must retain an explicit English request.
	if requestedLanguage != "" && in.Language == "" {
		in.Language = "English"
	}
	if !matchesLanguage(co.Language, in.Language) {
		return "", ErrUnsupported
	}
	finish := in.Finish
	if finish == "" {
		finish = FinishNonfoil
		if in.IsEtched() {
			finish = FinishEtched
		} else if in.Foil {
			finish = FinishFoil
		}
	}
	id, err := b.MatchIDFinish(in.ID, finish)
	if err != nil && options.AllowFinishSiblings {
		for _, sibling := range b.FinishSiblings(in.ID) {
			candidate, finishErr := b.MatchIDFinish(sibling, finish)
			if finishErr != nil {
				continue
			}
			if id != "" && id != candidate {
				return "", NewAliasingError(id, candidate)
			}
			id = candidate
		}
		if id == "" {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	probe := in
	probe.ID, probe.Finish = id, finish
	validated, err := b.Match(&probe)
	if err != nil {
		return "", err
	}
	if validated != id {
		return "", ErrCardUnknownID
	}
	return id, nil
}

// ValidateID checks a candidate UUID through the default backend.
func ValidateID(in InputCard, options IDValidationOptions) (string, error) {
	return defaultBackend.ValidateID(in, options)
}

// printingValidator is optional: games must explicitly implement conservative
// descriptor validation before vendor IDs may bypass ordinary text matching.
type printingValidator interface {
	ValidatePrinting(*Backend, *InputCard, *CardObject) bool
}

// ValidatePrinting reports whether a candidate establishes every supplied
// printing descriptor. Unknown descriptions and unsupported games return false.
// Name, language and finish are validated separately by ValidateID.
func (b *Backend) ValidatePrinting(in InputCard, id string) bool {
	co, err := b.GetUUID(id)
	if err != nil {
		return false
	}
	rules, ok := b.rules.(printingValidator)
	return ok && rules.ValidatePrinting(b, &in, co)
}

// ValidatePrinting validates descriptors through the default backend.
func ValidatePrinting(in InputCard, id string) bool {
	return defaultBackend.ValidatePrinting(in, id)
}
